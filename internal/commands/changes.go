package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/metadata"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var (
	changesCommits bool
	changesSummary bool
	changesAI      bool
)

var changesCmd = &cobra.Command{
	Use:   "changes <doc>",
	Short: "Show code changes since doc was last updated",
	Long: `Show what code has changed since a documentation file was last updated.

This helps you understand what changes need to be reflected in the documentation.`,
	Args: cobra.ExactArgs(1),
	RunE: runChanges,
}

func init() {
	changesCmd.Flags().BoolVar(&changesCommits, "commits", false, "show commit list only")
	changesCmd.Flags().BoolVar(&changesSummary, "summary", false, "output summary format")
	changesCmd.Flags().BoolVar(&changesAI, "ai", false, "output format optimized for AI documentation updates")
	rootCmd.AddCommand(changesCmd)
}

func runChanges(cmd *cobra.Command, args []string) error {
	doc := args[0]

	metaPath := cfg.MetadataPath(rootDir)
	meta := metadata.New(metaPath)

	if !meta.Exists() {
		return fmt.Errorf("metadata file not found: %s\nRun 'docdiff init' first", cfg.MetadataFile)
	}

	versions, err := meta.Load()
	if err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	lastHash, ok := versions[doc]
	if !ok {
		return fmt.Errorf("document not found in metadata: %s\nAvailable docs: %v", doc, versions.SortedDocs())
	}

	s := scanner.New(cfg, registry)
	scanResult, err := s.Scan(rootDir)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	files := scanResult.FilesByDoc[doc]
	out := cmd.OutOrStdout()
	if len(files) == 0 {
		fmt.Fprintf(out, "No source files have %s annotations pointing to %s\n", cfg.AnnotationTag, doc)
		return nil
	}

	g := git.New(rootDir)

	if changesAI {
		return outputAI(out, g, doc, lastHash, files)
	}

	if changesSummary {
		return outputSummary(out, g, doc, lastHash, files)
	}

	if changesCommits {
		return outputCommitsOnly(out, g, doc, lastHash, files)
	}

	return outputDefault(out, g, doc, lastHash, files)
}

func outputDefault(out io.Writer, g *git.Git, doc, lastHash string, files []string) error {
	lastCommitInfo, _ := g.CommitInfo(lastHash)
	currentHead, _ := g.HeadShort()

	fmt.Fprintf(out, "Changes to %s files since %s\n", doc, lastCommitInfo)
	fmt.Fprintln(out, strings.Repeat("=", 60))
	fmt.Fprintln(out)

	commits, _ := g.CommitsBetween(lastHash, "HEAD", files)
	changed, _ := g.ChangedFilesBetween(lastHash, "HEAD", files)

	fmt.Fprintf(out, "Commits: %d\n", len(commits))
	fmt.Fprintf(out, "Files changed: %d\n", len(changed))
	fmt.Fprintf(out, "Current HEAD: %s\n\n", currentHead)

	if len(commits) == 0 {
		fmt.Fprintln(out, "No changes since last documentation update.")
		return nil
	}

	fmt.Fprintln(out, "--- Commits ---")
	for _, c := range commits {
		fmt.Fprintln(out, c)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "--- Diff ---")
	diff, _ := g.Diff(lastHash, "HEAD", files)
	fmt.Fprintln(out, diff)

	return nil
}

func outputCommitsOnly(out io.Writer, g *git.Git, doc, lastHash string, files []string) error {
	commits, _ := g.CommitsBetween(lastHash, "HEAD", files)

	if len(commits) == 0 {
		fmt.Fprintln(out, "No commits since last documentation update.")
		return nil
	}

	fmt.Fprintf(out, "Commits affecting %s files since %s:\n\n", doc, lastHash)
	for _, c := range commits {
		fmt.Fprintln(out, c)
	}

	return nil
}

func outputSummary(out io.Writer, g *git.Git, doc, lastHash string, files []string) error {
	currentHead, _ := g.HeadShort()
	lastDate, _ := g.CommitDate(lastHash)
	currentDate, _ := g.CommitDate(currentHead)

	fmt.Fprintln(out, "# Documentation Update Context")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "**Document:** %s\n", doc)
	fmt.Fprintf(out, "**Last updated:** %s (%s)\n", lastHash, lastDate)
	fmt.Fprintf(out, "**Current:** %s (%s)\n\n", currentHead, currentDate)

	fmt.Fprintln(out, "## Files covered by this doc:")
	for _, file := range files {
		fmt.Fprintf(out, "- %s\n", file)
	}
	fmt.Fprintln(out)

	commits, _ := g.CommitsBetween(lastHash, "HEAD", files)
	if len(commits) == 0 {
		fmt.Fprintln(out, "## No changes since last doc update")
		return nil
	}

	fmt.Fprintln(out, "## Changes since last doc update:")
	fmt.Fprintln(out)

	details, _ := g.CommitDetails(lastHash, "HEAD", files)
	for _, commit := range details {
		fmt.Fprintf(out, "### %s - %s\n\n", commit.Short, commit.Subject)

		changedFiles, _ := g.FilesChangedInCommit(commit.Hash, files)
		if len(changedFiles) > 0 {
			fmt.Fprintf(out, "**Files:** %s\n\n", strings.Join(changedFiles, ", "))
		}

		diff, _ := g.ShowCommitDiff(commit.Hash, files)
		if diff != "" {
			fmt.Fprintln(out, "```diff")
			fmt.Fprintln(out, strings.TrimSpace(diff))
			fmt.Fprintln(out, "```")
			fmt.Fprintln(out)
		}
	}

	return nil
}

func outputAI(out io.Writer, g *git.Git, doc, lastHash string, files []string) error {
	currentHead, _ := g.HeadShort()
	lastDate, _ := g.CommitDate(lastHash)
	currentDate, _ := g.CommitDate(currentHead)

	fmt.Fprintln(out, "# Documentation Update Context")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "## Document")
	fmt.Fprintln(out, doc)
	fmt.Fprintln(out)

	docPath := filepath.Join(rootDir, doc)
	docContent, err := os.ReadFile(docPath)
	if err == nil {
		fmt.Fprintln(out, "## Current Documentation")
		fmt.Fprintln(out, "```markdown")
		fmt.Fprint(out, string(docContent))
		if !strings.HasSuffix(string(docContent), "\n") {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, "```")
		fmt.Fprintln(out)
	}

	fmt.Fprintln(out, "## Tracked Source Files")
	for _, file := range files {
		fmt.Fprintf(out, "- %s\n", file)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "## Changes Since Last Update")
	fmt.Fprintf(out, "Last synced: %s (%s)\n", lastHash, lastDate)
	fmt.Fprintf(out, "Current HEAD: %s (%s)\n", currentHead, currentDate)

	commits, _ := g.CommitsBetween(lastHash, "HEAD", files)
	fmt.Fprintf(out, "Commits: %d\n", len(commits))
	fmt.Fprintln(out)

	if len(commits) == 0 {
		fmt.Fprintln(out, "No changes since last documentation update.")
		return nil
	}

	details, _ := g.CommitDetails(lastHash, "HEAD", files)
	for _, commit := range details {
		fmt.Fprintf(out, "### Commit: %s - %s\n", commit.Short, commit.Subject)

		changedFiles, _ := g.FilesChangedInCommit(commit.Hash, files)
		if len(changedFiles) > 0 {
			fmt.Fprintf(out, "**Files:** %s\n\n", strings.Join(changedFiles, ", "))
		}

		diff, _ := g.ShowCommitDiff(commit.Hash, files)
		if diff != "" {
			fmt.Fprintln(out, "```diff")
			fmt.Fprintln(out, strings.TrimSpace(diff))
			fmt.Fprintln(out, "```")
			fmt.Fprintln(out)
		}
	}

	fmt.Fprintln(out, "## Instructions")
	fmt.Fprintln(out, "Update the documentation to reflect these code changes.")
	fmt.Fprintln(out, "Focus on: new features, changed behavior, removed functionality.")

	return nil
}
