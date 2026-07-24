package commands

// @doc README.md

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/report"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var (
	ackTo       string
	ackAmend    bool
	ackRequired bool
)

var ackCmd = &cobra.Command{
	Use:   "ack <doc>... | ack --required",
	Short: "Mark a doc reviewed even though it needed no change",
	Long: `Acknowledge that a doc is up to date even though its linked code changed.

Normally a doc is marked reviewed by editing it in the same commit as its code.
When you review the code and decide the doc needs NO edit, there is nothing to
commit against it — so 'ack' records a floor commit (default HEAD) for the doc.
Staleness is then measured from the newer of the doc's own last commit and this
floor, so the doc stops being reported stale until its linked code changes again.

The floor is an existing commit, so there is no chicken-and-egg: commit your
code first, then run 'docdiff ack <doc>' and commit .docdiff-acks.json.

To keep code and its ack in ONE commit (per the commit-together rule), use
--amend: commit your code, then 'docdiff ack <doc> --amend' folds the floor
into that same commit so it points at exactly the code it reviews.

--required acks, in one command, exactly the docs 'check' would have flagged
as Required for HEAD's own diff (scope matching included) and that are still
stale — i.e. "I reviewed everything my last commit affected; no docs needed
edits". Composes with --amend.`,
	Args: cobra.ArbitraryArgs,
	RunE: runAck,
}

func init() {
	ackCmd.Flags().StringVar(&ackTo, "to", "", "floor commit to ack at (HEAD, branch, or sha); default HEAD")
	ackCmd.Flags().BoolVar(&ackAmend, "amend", false, "fold .docdiff-acks.json into the current HEAD commit (single-commit review)")
	ackCmd.Flags().BoolVar(&ackRequired, "required", false, "ack the docs check would require for HEAD's own diff (no doc args)")
	rootCmd.AddCommand(ackCmd)
}

func runAck(cmd *cobra.Command, args []string) error {
	g := git.New(rootDir)

	switch {
	case ackRequired && len(args) > 0:
		return fmt.Errorf("--required computes its own doc set; drop the doc arguments")
	case !ackRequired && len(args) == 0:
		return fmt.Errorf("requires at least 1 doc argument (or --required)")
	}

	if ackRequired {
		if ackTo != "" {
			return fmt.Errorf("--required always acks at HEAD; drop --to")
		}
		required, err := requiredDocsAtHead(g, cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		if len(required) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Nothing to ack: HEAD's diff requires no doc updates.")
			return nil
		}
		args = required
	}

	ref := ackTo
	if ref == "" {
		ref = "HEAD"
	}
	sha, err := g.ResolveShort(ref)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", ref, err)
	}

	acks, err := loadAcks(rootDir)
	if err != nil {
		return fmt.Errorf("failed to load acks: %w", err)
	}

	out := cmd.OutOrStdout()
	for _, doc := range args {
		doc = filepath.ToSlash(doc)
		if _, statErr := os.Stat(filepath.Join(rootDir, doc)); statErr != nil {
			return fmt.Errorf("doc not found: %s", doc)
		}
		acks[doc] = sha
		fmt.Fprintf(out, "Acked %s at %s (won't report stale until its linked code changes again)\n", doc, sha)
	}

	if err := saveAcks(rootDir, acks); err != nil {
		return fmt.Errorf("failed to save acks: %w", err)
	}

	sort.Strings(args)
	if ackAmend {
		if err := g.StageAndAmend(acksFile); err != nil {
			return fmt.Errorf("failed to amend %s into HEAD: %w", acksFile, err)
		}
		fmt.Fprintf(out, "\nFolded %s into HEAD — code and its ack now share one commit.\n", acksFile)
		return nil
	}
	fmt.Fprintf(out, "\nCommit %s to share these acknowledgements.\n", acksFile)
	return nil
}

// requiredDocsAtHead computes the docs `check` would have flagged as Required
// for HEAD's own diff — same scope matching, same annotation-only filtering —
// intersected with the currently-stale set. Required is pre-commit vocabulary;
// this replays it post-commit so one command acks exactly what the last commit
// put on review.
func requiredDocsAtHead(g *git.Git, errOut io.Writer) ([]string, error) {
	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}
	return docsRequiredByHead(g, scanResult, computeStaleDocs(g, scanResult.FilesByDoc, errOut))
}

// docsRequiredByHead is requiredDocsAtHead's core, taking an existing scan and
// stale set so `check` can reuse it to label its hidden-stale docs without
// re-scanning.
func docsRequiredByHead(g *git.Git, scanResult *scanner.Result, stale map[string]*report.StaleDoc) ([]string, error) {
	changedList, err := g.FilesChangedInCommit("HEAD", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list HEAD's changed files: %w", err)
	}
	changedInHead := make(map[string]bool, len(changedList))
	for _, f := range changedList {
		changedInHead[f] = true
	}

	rawHunks, err := g.ShowCommitHunks("HEAD", cfg.AnnotationTag)
	if err != nil {
		return nil, fmt.Errorf("failed to diff HEAD: %w", err)
	}
	hunks, annOnlyFiles := splitAnnotationHunks(rawHunks)

	var docs []string
	for doc, files := range scanResult.FilesByDoc {
		if changedInHead[doc] {
			continue // doc was edited in HEAD: updated, not required
		}
		if stale[doc] == nil {
			continue // already fresh (or already acked): nothing to do
		}
		for _, f := range files {
			if !changedInHead[f] || annOnlyFiles[f] {
				continue
			}
			if hit, _ := fileHitsDoc(scanResult.Annotations[f], doc, hunks); hit {
				docs = append(docs, doc)
				break
			}
		}
	}
	sort.Strings(docs)
	return docs, nil
}
