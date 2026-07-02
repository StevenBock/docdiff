package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var ErrDocsNeedUpdate = fmt.Errorf("docs linked to changed files need updating")

var (
	checkStaged bool
	checkFiles  []string
	checkJSON   bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Show only docs affected by your current changes",
	Long: `Check which documentation is affected by your current changes, ignoring
unrelated stale docs across the rest of the repo.

By default it inspects working-tree changes (staged, unstaged, and untracked).
Use --staged for index-only changes, or --files to check an explicit set.

A doc is "updated" if it was edited alongside its linked source files, and
"needs update" if its linked files changed but the doc itself did not. The exit
code is non-zero when any affected doc needs updating, so it answers a single
question for agents: did I satisfy docs for my change?`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkStaged, "staged", false, "only consider staged (index) changes")
	checkCmd.Flags().StringSliceVar(&checkFiles, "files", nil, "check an explicit list of files instead of git changes")
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(checkCmd)
}

type checkResult struct {
	Doc            string   `json:"doc"`
	Status         string   `json:"status"` // "updated" | "needs update"
	DocInChangeset bool     `json:"doc_in_changeset"`
	ChangedFiles   []string `json:"changed_files"`
}

func runCheck(cmd *cobra.Command, args []string) error {
	g := git.New(rootDir)

	changed, source, err := changedSet(g)
	if err != nil {
		return err
	}

	inChange := make(map[string]bool, len(changed))
	for _, f := range changed {
		inChange[f] = true
	}

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	results := make([]checkResult, 0)
	affected := make(map[string]bool)
	for doc, files := range scanResult.FilesByDoc {
		var linkedChanged []string
		for _, f := range files {
			if inChange[f] {
				linkedChanged = append(linkedChanged, f)
			}
		}
		if len(linkedChanged) == 0 {
			continue
		}
		sort.Strings(linkedChanged)
		affected[doc] = true

		status := "needs update"
		if inChange[doc] {
			status = "updated"
		}
		results = append(results, checkResult{
			Doc:            doc,
			Status:         status,
			DocInChangeset: inChange[doc],
			ChangedFiles:   linkedChanged,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Doc < results[j].Doc })

	// Count stale docs that are NOT related to the current change, so we can
	// say how much noise we hid. Best-effort: needs the metadata file.
	unrelatedStale := unrelatedStaleCount(g, scanResult.FilesByDoc, affected, cmd.ErrOrStderr())

	// Surface missing back-links for the docs this change touches, so they're
	// caught in the normal flow instead of a separate `report --undocumented`.
	var undocRefs []scanner.UndocumentedRef
	for _, ref := range scanResult.UndocumentedRefs {
		if affected[ref.DocPath] {
			undocRefs = append(undocRefs, ref)
		}
	}

	needsUpdate := 0
	for _, r := range results {
		if r.Status == "needs update" {
			needsUpdate++
		}
	}

	out := cmd.OutOrStdout()
	if checkJSON {
		writeCheckJSON(out, source, results, undocRefs, unrelatedStale, needsUpdate)
	} else {
		writeCheckHuman(out, source, results, undocRefs, unrelatedStale, needsUpdate)
	}

	if needsUpdate > 0 {
		return ErrDocsNeedUpdate
	}
	return nil
}

// changedSet resolves the set of changed source files and a label for it.
func changedSet(g *git.Git) ([]string, string, error) {
	switch {
	case len(checkFiles) > 0:
		files := make([]string, 0, len(checkFiles))
		for _, f := range checkFiles {
			if filepath.IsAbs(f) {
				if rel, err := filepath.Rel(rootDir, f); err == nil {
					f = rel
				}
			}
			files = append(files, filepath.ToSlash(f))
		}
		return files, "files", nil
	case checkStaged:
		files, err := g.StagedFiles()
		return files, "staged changes", err
	default:
		files, err := g.WorkingTreeFiles()
		return files, "working tree", err
	}
}

func unrelatedStaleCount(g *git.Git, filesByDoc map[string][]string, affected map[string]bool, errOut io.Writer) int {
	count := 0
	for doc := range computeStaleDocs(g, filesByDoc, errOut) {
		if !affected[doc] {
			count++
		}
	}
	return count
}

func writeCheckHuman(out io.Writer, source string, results []checkResult, undocRefs []scanner.UndocumentedRef, unrelatedStale, needsUpdate int) {
	if len(results) == 0 {
		fmt.Fprintf(out, "No docs are linked to your %s changes.\n", source)
		if unrelatedStale > 0 {
			fmt.Fprintf(out, "(%d unrelated stale docs hidden — run 'docdiff report' to see them.)\n", unrelatedStale)
		}
		return
	}

	fmt.Fprintf(out, "Relevant docs for %s:\n", source)
	for _, r := range results {
		fmt.Fprintf(out, "  %s: %s\n", r.Doc, r.Status)
	}
	fmt.Fprintln(out)

	updated := len(results) - needsUpdate
	fmt.Fprintf(out, "%d affected (%d updated, %d needs-update).\n", len(results), updated, needsUpdate)
	if unrelatedStale > 0 {
		fmt.Fprintf(out, "Unrelated stale docs: %d hidden (run 'docdiff report' for the full picture).\n", unrelatedStale)
	}

	if len(undocRefs) > 0 {
		fmt.Fprintf(out, "\nMissing back-links in affected docs (%d):\n", len(undocRefs))
		for _, ref := range undocRefs {
			fmt.Fprintf(out, "  %s references %s (add: %s %s)\n", ref.DocPath, ref.SourceFile, cfg.AnnotationTag, ref.DocPath)
		}
	}

	if needsUpdate > 0 {
		fmt.Fprintln(out, "\nNext: update the docs above and commit them together with your code —")
		fmt.Fprintln(out, "editing a doc in the same commit as its linked source marks it reviewed.")
	}
}

func writeCheckJSON(out io.Writer, source string, results []checkResult, undocRefs []scanner.UndocumentedRef, unrelatedStale, needsUpdate int) {
	if undocRefs == nil {
		undocRefs = []scanner.UndocumentedRef{}
	}
	payload := struct {
		Source           string                    `json:"source"`
		Affected         []checkResult             `json:"affected"`
		UndocumentedRefs []scanner.UndocumentedRef `json:"undocumented_refs"`
		NeedsUpdate      int                       `json:"needs_update"`
		UnrelatedStale   int                       `json:"unrelated_stale"`
	}{
		Source:           source,
		Affected:         results,
		UndocumentedRefs: undocRefs,
		NeedsUpdate:      needsUpdate,
		UnrelatedStale:   unrelatedStale,
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Fprintln(out, string(data))
}
