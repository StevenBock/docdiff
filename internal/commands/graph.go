package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/graph"
	"github.com/StevenBock/docdiff/internal/metadata"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var (
	graphMermaid bool
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Output doc-to-file relationship graph",
	Long: `Output a graph showing relationships between documentation and source files.

By default outputs in DOT (GraphViz) format. Use --mermaid for Mermaid format.
Stale documentation relationships are highlighted in red.`,
	RunE: runGraph,
}

func init() {
	graphCmd.Flags().BoolVar(&graphMermaid, "mermaid", false, "output in Mermaid format instead of DOT")
	rootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	metaPath := cfg.MetadataPath(rootDir)
	meta := metadata.New(metaPath)

	if !meta.Exists() {
		return fmt.Errorf("metadata file not found: %s\nRun 'docdiff init' first", cfg.MetadataFile)
	}

	versions, err := meta.Load()
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	g := git.New(rootDir)
	staleDocs := make(map[string]bool)

	for doc, lastHash := range versions {
		files := scanResult.FilesByDoc[doc]
		if len(files) == 0 {
			continue
		}

		changed, err := g.ChangedFilesBetween(lastHash, "HEAD", files)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to check changes for %s (%s..HEAD): %v\n", doc, lastHash, err)
			continue
		}

		if len(changed) > 0 {
			staleDocs[doc] = true
		}
	}

	gr := graph.Build(scanResult.FilesByDoc, staleDocs)

	var formatter graph.GraphFormatter
	if graphMermaid {
		formatter = &graph.MermaidFormatter{}
	} else {
		formatter = &graph.DOTFormatter{}
	}

	output, err := formatter.Format(gr)
	if err != nil {
		return fmt.Errorf("failed to format graph: %w", err)
	}

	cmd.Print(string(output))
	return nil
}
