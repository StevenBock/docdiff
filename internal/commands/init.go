package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/metadata"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize documentation version tracking",
	Long: `Initialize documentation version tracking with current HEAD.

Creates a metadata file that tracks which git commit each documentation
file was last verified against.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing metadata file")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	metaPath := cfg.MetadataPath(rootDir)
	meta := metadata.New(metaPath)

	if meta.Exists() && !initForce {
		return fmt.Errorf("metadata file already exists: %s\nUse --force to overwrite", cfg.MetadataFile)
	}

	g := git.New(rootDir)
	if !g.IsRepo() {
		return fmt.Errorf("not a git repository: %s", rootDir)
	}

	head, err := g.HeadShort()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	docsPath := cfg.DocsPath(rootDir)
	entries, err := os.ReadDir(docsPath)
	if err != nil {
		return fmt.Errorf("failed to read docs directory '%s': %w", cfg.DocsDirectory, err)
	}

	versions := make(metadata.DocVersions)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		docPath := filepath.Join(cfg.DocsDirectory, entry.Name())
		docPath = filepath.ToSlash(docPath)
		versions[docPath] = head
	}

	if len(versions) == 0 {
		return fmt.Errorf("no markdown files found in %s", cfg.DocsDirectory)
	}

	if err := meta.Save(versions); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	fmt.Printf("Initialized documentation tracking with %d docs at %s\n\n", len(versions), head)

	docs := make([]string, 0, len(versions))
	for doc := range versions {
		docs = append(docs, doc)
	}
	sort.Strings(docs)

	for _, doc := range docs {
		fmt.Printf("  %s: %s\n", doc, versions[doc])
	}

	return nil
}
