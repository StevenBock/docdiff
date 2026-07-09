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
	checkStaged      bool
	checkFiles       []string
	checkJSON        bool
	checkNoBacklinks bool
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
	checkCmd.Flags().BoolVar(&checkNoBacklinks, "no-backlinks", false, "hide missing back-link (hygiene) suggestions")
	rootCmd.AddCommand(checkCmd)
}

type checkResult struct {
	Doc             string                 `json:"doc"`
	Status          string                 `json:"status"` // "updated" | "needs update"
	DocInChangeset  bool                   `json:"doc_in_changeset"`
	ChangedFiles    []string               `json:"changed_files"`
	LinkedFileCount int                    `json:"linked_file_count"`
	Annotations     []annotationProvenance `json:"annotations,omitempty"`
}

type annotationProvenance struct {
	File  string `json:"file"`
	Line  int    `json:"line,omitempty"`
	Kind  string `json:"kind"`
	Scope string `json:"scope,omitempty"`
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

	// Changed line ranges per file, so scoped annotations only flag docs whose
	// owned region was actually touched. Only diff-backed modes have hunks;
	// --files has none, so it falls back to whole-file ownership.
	var hunks map[string][]git.LineRange
	switch source {
	case "working tree":
		hunks, _ = g.ChangedHunksSince("HEAD", false, changed)
	case "staged changes":
		hunks, _ = g.ChangedHunksSince("HEAD", true, changed)
	}

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	acks, err := loadAcks(rootDir)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to load %s: %v\n", acksFile, err)
		acks = map[string]string{}
	}

	results := make([]checkResult, 0)
	affected := make(map[string]bool)
	warnings := make([]string, 0)
	seenWarnings := make(map[string]bool)
	for doc, files := range scanResult.FilesByDoc {
		var linkedChanged []string
		for _, f := range files {
			if inChange[f] && fileHitsDoc(scanResult.Annotations[f], doc, hunks) {
				linkedChanged = append(linkedChanged, f)
			}
		}
		if len(linkedChanged) == 0 {
			continue
		}
		sort.Strings(linkedChanged)

		if !inChange[doc] {
			linkedChanged = changedFilesSinceBaseline(g, doc, linkedChanged, source, acks, cmd.ErrOrStderr())
			if len(linkedChanged) == 0 {
				continue
			}
		}
		affected[doc] = true

		status := "needs update"
		if inChange[doc] {
			status = "updated"
		}
		provenance := make([]annotationProvenance, 0)
		for _, f := range linkedChanged {
			provenance = append(provenance, provenanceForDoc(scanResult.Annotations[f], doc, f)...)
			for _, warning := range annotationWarnings(scanResult.Annotations[f], doc) {
				if !seenWarnings[warning] {
					seenWarnings[warning] = true
					warnings = append(warnings, warning)
				}
			}
		}
		results = append(results, checkResult{
			Doc:             doc,
			Status:          status,
			DocInChangeset:  inChange[doc],
			ChangedFiles:    linkedChanged,
			LinkedFileCount: len(files),
			Annotations:     provenance,
		})
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Doc < results[j].Doc })
	sort.Strings(warnings)

	// Count stale docs that are NOT related to the current change, so we can
	// say how much noise we hid. Best-effort: needs the metadata file.
	unrelatedStale := unrelatedStaleCount(g, scanResult.FilesByDoc, affected, cmd.ErrOrStderr())

	// Surface missing back-links for the docs this change touches, so they're
	// caught in the normal flow instead of a separate `report --undocumented`.
	// This is hygiene, not a blocker: it never gates the exit code.
	var undocRefs []scanner.UndocumentedRef
	if !checkNoBacklinks {
		for _, ref := range scanResult.UndocumentedRefs {
			if affected[ref.DocPath] {
				undocRefs = append(undocRefs, ref)
			}
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
		writeCheckJSON(out, source, results, undocRefs, unrelatedStale, needsUpdate, warnings)
	} else {
		writeCheckHuman(out, source, results, undocRefs, unrelatedStale, needsUpdate, warnings)
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

func changedFilesSinceBaseline(g *git.Git, doc string, files []string, source string, acks map[string]string, errOut io.Writer) []string {
	if source == "working tree" {
		return files
	}

	baseline, err := baselineForDoc(g, doc, acks)
	if err != nil {
		fmt.Fprintf(errOut, "Warning: failed to find last commit for %s: %v\n", doc, err)
		return files
	}
	if baseline.Effective == "" {
		return files
	}

	staged := source == "staged changes"
	changed, err := g.ChangedFilesSince(baseline.Effective, staged, files)
	if err != nil {
		fmt.Fprintf(errOut, "Warning: failed to check changes for %s (%s..%s): %v\n", doc, baseline.Effective, source, err)
		return files
	}
	if source == "files" {
		untracked, err := g.UntrackedFiles(files)
		if err == nil {
			changed = append(changed, untracked...)
		}
	}

	changedSet := make(map[string]bool, len(changed))
	for _, f := range changed {
		changedSet[f] = true
	}
	filtered := make([]string, 0, len(files))
	for _, f := range files {
		if changedSet[f] {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func writeCheckHuman(out io.Writer, source string, results []checkResult, undocRefs []scanner.UndocumentedRef, unrelatedStale, needsUpdate int, warnings []string) {
	if len(results) == 0 {
		fmt.Fprintf(out, "No docs are linked to your %s changes.\n", source)
		if unrelatedStale > 0 {
			fmt.Fprintf(out, "(%d unrelated stale docs hidden — run 'docdiff report' to see them.)\n", unrelatedStale)
		}
		return
	}

	// Section 1 — required: docs whose linked code changed but weren't edited.
	// This is the only section that drives the exit code.
	fmt.Fprintf(out, "Required for %s changes (%d):\n", source, needsUpdate)
	if needsUpdate == 0 {
		fmt.Fprintln(out, "  none — every affected doc was edited alongside its code.")
	}
	for _, r := range results {
		if r.Status == "needs update" {
			fmt.Fprintf(out, "  %s: needs update%s\n", r.Doc, broadHint(r))
			writeProvenance(out, r)
		}
	}

	// Section 2 — already satisfied: edited in this changeset. Informational.
	updated := len(results) - needsUpdate
	if updated > 0 {
		fmt.Fprintf(out, "\nAlready updated in this changeset (%d):\n", updated)
		for _, r := range results {
			if r.Status == "updated" {
				fmt.Fprintf(out, "  %s%s\n", r.Doc, broadHint(r))
				writeProvenance(out, r)
			}
		}
	}

	if len(warnings) > 0 {
		fmt.Fprintf(out, "\nAnnotation warnings:\n")
		for _, warning := range warnings {
			fmt.Fprintf(out, "  %s\n", warning)
		}
	}

	// Section 3 — hygiene: missing back-links. Never gates the exit code.
	if len(undocRefs) > 0 {
		fmt.Fprintf(out, "\nBack-link hygiene — optional (%d):\n", len(undocRefs))
		for _, ref := range undocRefs {
			fmt.Fprintf(out, "  %s references %s (add: %s %s)\n", ref.DocPath, ref.SourceFile, cfg.AnnotationTag, ref.DocPath)
		}
	}

	if unrelatedStale > 0 {
		fmt.Fprintf(out, "\nUnrelated stale docs: %d hidden (run 'docdiff report' for the full picture).\n", unrelatedStale)
	}

	if needsUpdate > 0 {
		fmt.Fprintln(out, "\nNext: update the required docs above and commit them together with your code —")
		fmt.Fprintln(out, "editing a doc in the same commit as its linked source marks it reviewed.")
		fmt.Fprintln(out, "New to docdiff? Run 'docdiff onboard' for agent-ready workflow instructions.")
	}
}

const broadDocLinkedFileThreshold = 20

func broadHint(r checkResult) string {
	if r.LinkedFileCount >= broadDocLinkedFileThreshold {
		return fmt.Sprintf(" (broad: %d linked files)", r.LinkedFileCount)
	}
	return ""
}

func writeProvenance(out io.Writer, r checkResult) {
	for _, p := range r.Annotations {
		switch {
		case p.Kind == "scoped":
			fmt.Fprintf(out, "    via %s:%d scoped %s #%s\n", p.File, p.Line, cfg.AnnotationTag, p.Scope)
		case p.Line > 0:
			fmt.Fprintf(out, "    via %s:%d whole-file %s\n", p.File, p.Line, cfg.AnnotationTag)
		default:
			fmt.Fprintf(out, "    via %s %s\n", p.File, p.Kind)
		}
	}
}

func provenanceForDoc(ann *scanner.Annotation, doc, file string) []annotationProvenance {
	if ann == nil {
		return []annotationProvenance{{File: file, Kind: "unknown annotation"}}
	}

	out := make([]annotationProvenance, 0)
	for _, d := range ann.Details {
		if d.Path != doc {
			continue
		}
		kind := "whole-file"
		if d.Scope != "" {
			kind = "scoped"
		}
		out = append(out, annotationProvenance{
			File:  file,
			Line:  d.Line,
			Kind:  kind,
			Scope: d.Scope,
		})
	}
	if len(out) == 0 {
		return []annotationProvenance{{File: file, Kind: "unknown annotation"}}
	}
	return out
}

func annotationWarnings(ann *scanner.Annotation, doc string) []string {
	if ann == nil {
		return nil
	}
	hasWholeFile := false
	hasScoped := false
	for _, d := range ann.Details {
		if d.Path != doc {
			continue
		}
		if d.Scope == "" {
			hasWholeFile = true
		} else {
			hasScoped = true
		}
	}
	if hasWholeFile && hasScoped {
		return []string{fmt.Sprintf("%s mixes whole-file and scoped annotations for %s; whole-file ownership wins for that doc.", ann.FilePath, doc)}
	}
	return nil
}

func writeCheckJSON(out io.Writer, source string, results []checkResult, undocRefs []scanner.UndocumentedRef, unrelatedStale, needsUpdate int, warnings []string) {
	if undocRefs == nil {
		undocRefs = []scanner.UndocumentedRef{}
	}
	if warnings == nil {
		warnings = []string{}
	}
	payload := struct {
		Source           string                    `json:"source"`
		Affected         []checkResult             `json:"affected"`
		UndocumentedRefs []scanner.UndocumentedRef `json:"undocumented_refs"`
		NeedsUpdate      int                       `json:"needs_update"`
		UnrelatedStale   int                       `json:"unrelated_stale"`
		Warnings         []string                  `json:"warnings"`
	}{
		Source:           source,
		Affected:         results,
		UndocumentedRefs: undocRefs,
		NeedsUpdate:      needsUpdate,
		UnrelatedStale:   unrelatedStale,
		Warnings:         warnings,
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Fprintln(out, string(data))
}
