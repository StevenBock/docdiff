package commands

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/metadata"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var (
	syncTo       string
	syncAffected bool
)

var syncCmd = &cobra.Command{
	Use:   "sync [doc]",
	Short: "Update documentation version metadata to a commit",
	Long: `Update documentation version metadata after reviewing and updating docs.

Without arguments, updates all docs to current HEAD.
With a doc path argument, updates only that specific doc.

Sync records a commit hash, so it must run after the code/doc commit exists.
The clean pre-commit flow is to commit first, then:

  docdiff sync <doc> --to HEAD

Use --to to point at any ref (HEAD, a branch, or an explicit sha) instead of
the current HEAD.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringVar(&syncTo, "to", "", "ref to sync to (HEAD, branch, or sha); default current HEAD")
	syncCmd.Flags().BoolVar(&syncAffected, "affected", false, "sync only docs whose linked code changed (the stale set), not all docs")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	metaPath := cfg.MetadataPath(rootDir)
	meta := metadata.New(metaPath)

	if !meta.Exists() {
		return fmt.Errorf("metadata file not found: %s\nRun 'docdiff init' first", cfg.MetadataFile)
	}

	versions, err := meta.Load()
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	g := git.New(rootDir)

	target := syncTo
	if target == "" {
		target = "HEAD"
	}
	resolved, err := g.ResolveShort(target)
	if err != nil {
		return fmt.Errorf("failed to resolve %q: %w", target, err)
	}

	out := cmd.OutOrStdout()

	if syncAffected {
		return syncAffectedDocs(out, cmd.ErrOrStderr(), meta, versions, g, resolved)
	}

	if len(args) == 1 {
		return syncSingleDoc(out, meta, versions, args[0], resolved)
	}

	return syncAllDocs(out, meta, versions, resolved)
}

// syncAffectedDocs updates only the docs whose linked source files changed
// since their last synced hash (the set `check`/`report` flag as stale). This
// is the post-commit "I reviewed the affected docs, record them" path.
func syncAffectedDocs(out, errOut io.Writer, meta *metadata.Manager, versions metadata.DocVersions, g *git.Git, target string) error {
	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	stale := computeStaleDocs(g, versions, scanResult.FilesByDoc, errOut)
	if len(stale) == 0 {
		fmt.Fprintln(out, "No affected docs to sync (no doc's linked code changed).")
		return nil
	}

	docs := make([]string, 0, len(stale))
	for doc := range stale {
		docs = append(docs, doc)
	}
	sort.Strings(docs)

	fmt.Fprintf(out, "Syncing affected docs to %s...\n\n", target)
	updated := 0
	for _, doc := range docs {
		oldHash := versions[doc]
		if oldHash == target {
			continue
		}
		versions[doc] = target
		fmt.Fprintf(out, "  %s: %s -> %s\n", doc, oldHash, target)
		updated++
	}

	if updated == 0 {
		fmt.Fprintln(out, "Affected docs already current.")
		return nil
	}

	if err := meta.Save(versions); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Fprintf(out, "\nSynced %d affected doc(s) to %s.\n", updated, target)
	return nil
}

func syncSingleDoc(out io.Writer, meta *metadata.Manager, versions metadata.DocVersions, doc, currentHead string) error {
	oldHash, ok := versions[doc]
	if !ok {
		return fmt.Errorf("document not found in metadata: %s\nAvailable docs: %v", doc, versions.SortedDocs())
	}

	if oldHash == currentHead {
		fmt.Fprintf(out, "%s is already at %s\n", doc, currentHead)
		return nil
	}

	versions[doc] = currentHead
	if err := meta.Save(versions); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Fprintf(out, "Updated %s: %s -> %s\n", doc, oldHash, currentHead)
	return nil
}

func syncAllDocs(out io.Writer, meta *metadata.Manager, versions metadata.DocVersions, currentHead string) error {
	updated := 0
	skipped := 0

	fmt.Fprintf(out, "Syncing all docs to %s...\n\n", currentHead)

	for _, doc := range versions.SortedDocs() {
		oldHash := versions[doc]
		if oldHash == currentHead {
			skipped++
			continue
		}

		versions[doc] = currentHead
		fmt.Fprintf(out, "  %s: %s -> %s\n", doc, oldHash, currentHead)
		updated++
	}

	if updated == 0 {
		fmt.Fprintln(out, "All docs already at current HEAD.")
		return nil
	}

	if err := meta.Save(versions); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Fprintf(out, "\nUpdated %d docs, %d already current.\n", updated, skipped)
	return nil
}
