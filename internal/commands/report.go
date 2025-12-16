package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/metadata"
	"github.com/StevenBock/docdiff/internal/report"
	"github.com/StevenBock/docdiff/internal/scanner"
)

var (
	ErrStaleDocsFound    = errors.New("stale documentation found")
	ErrOrphanedFilesFound = errors.New("orphaned files found")
)

var (
	reportStale    bool
	reportOrphaned bool
	reportJSON     bool
	reportSARIF    bool
	reportCI       bool
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show documentation coverage and staleness report",
	Long: `Show a report of documentation coverage and staleness.

By default, shows a full report including stale docs, coverage by doc file,
orphaned files, and summary statistics.`,
	RunE: runReport,
}

func init() {
	reportCmd.Flags().BoolVar(&reportStale, "stale", false, "only show stale docs")
	reportCmd.Flags().BoolVar(&reportOrphaned, "orphaned", false, "only show orphaned files")
	reportCmd.Flags().BoolVar(&reportJSON, "json", false, "output as JSON")
	reportCmd.Flags().BoolVar(&reportSARIF, "sarif", false, "output as SARIF for CI integration")
	reportCmd.Flags().BoolVar(&reportCI, "ci", false, "enable CI mode (exit 1 on stale docs)")
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
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
	staleDocs := make(map[string]*report.StaleDoc)

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
			commitInfo, _ := g.CommitInfo(lastHash)
			staleDocs[doc] = &report.StaleDoc{
				Path:           doc,
				LastHash:       lastHash,
				LastCommitInfo: commitInfo,
				FilesChanged:   len(changed),
				ChangedFiles:   changed,
			}
		}
	}

	rpt := report.NewReport()
	rpt.Metadata = versions
	rpt.StaleDocs = staleDocs
	rpt.FilesByDoc = scanResult.FilesByDoc
	rpt.OrphanedFiles = scanResult.OrphanedFiles()
	rpt.CalculateSummary(len(scanResult.AllFiles), len(scanResult.Annotations))

	var formatter report.Formatter
	switch {
	case reportJSON:
		formatter = &report.JSONFormatter{}
	case reportSARIF:
		formatter = &report.SARIFFormatter{}
	default:
		formatter = &report.HumanFormatter{
			ShowStaleOnly:    reportStale,
			ShowOrphanedOnly: reportOrphaned,
			Tag:              cfg.AnnotationTag,
		}
	}

	output, err := formatter.Format(rpt)
	if err != nil {
		return err
	}

	cmd.Print(string(output))

	if reportCI || isCI() {
		if cfg.CI.FailOnStale && len(staleDocs) > 0 {
			return ErrStaleDocsFound
		}
		if cfg.CI.FailOnOrphaned && len(rpt.OrphanedFiles) > 0 {
			return ErrOrphanedFilesFound
		}
	}

	return nil
}

func isCI() bool {
	ciEnvVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE"}
	for _, env := range ciEnvVars {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}
