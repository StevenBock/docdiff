package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/metadata"
)

var syncCmd = &cobra.Command{
	Use:   "sync [doc]",
	Short: "Update documentation version metadata to current HEAD",
	Long: `Update documentation version metadata after reviewing and updating docs.

Without arguments, updates all docs to current HEAD.
With a doc path argument, updates only that specific doc.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSync,
}

func init() {
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
	currentHead, err := g.HeadShort()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	out := cmd.OutOrStdout()

	if len(args) == 1 {
		return syncSingleDoc(out, meta, versions, args[0], currentHead)
	}

	return syncAllDocs(out, meta, versions, currentHead)
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

	fmt.Fprintf(out, "Syncing all docs to HEAD (%s)...\n\n", currentHead)

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
