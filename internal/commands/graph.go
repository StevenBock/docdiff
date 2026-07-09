package commands

// @doc README.md

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/graph"
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
	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	g := git.New(rootDir)
	staleDocs := make(map[string]bool)
	for doc := range computeStaleDocs(g, scanResult.FilesByDoc, cmd.ErrOrStderr()) {
		staleDocs[doc] = true
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
