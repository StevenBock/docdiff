package report

import (
	"sort"
	"strings"

	"github.com/StevenBock/docdiff/internal/metadata"
	"github.com/StevenBock/docdiff/internal/scanner"
)

type StaleDoc struct {
	Path           string
	LastHash       string
	LastCommitInfo string
	FilesChanged   int
	ChangedFiles   []string
}

type DirectoryCoverage struct {
	Path            string
	TotalFiles      int
	DocumentedFiles int
	CoveragePercent float64
}

type Report struct {
	Metadata          metadata.DocVersions
	StaleDocs         map[string]*StaleDoc
	FilesByDoc        map[string][]string
	OrphanedFiles     []string
	UndocumentedRefs  []scanner.UndocumentedRef
	DirectoryCoverage []DirectoryCoverage
	Summary           Summary
}

type Summary struct {
	TotalDocs        int
	TotalFiles       int
	DocumentedFiles  int
	OrphanedFiles    int
	StaleDocs        int
	UndocumentedRefs int
	CoveragePercent  float64
}

type Formatter interface {
	Format(report *Report) ([]byte, error)
}

func NewReport() *Report {
	return &Report{
		Metadata:         make(metadata.DocVersions),
		StaleDocs:        make(map[string]*StaleDoc),
		FilesByDoc:       make(map[string][]string),
		OrphanedFiles:    make([]string, 0),
		UndocumentedRefs: make([]scanner.UndocumentedRef, 0),
	}
}

func (r *Report) CalculateSummary(totalFiles, documentedFiles int) {
	r.Summary = Summary{
		TotalDocs:        len(r.Metadata),
		TotalFiles:       totalFiles,
		DocumentedFiles:  documentedFiles,
		OrphanedFiles:    len(r.OrphanedFiles),
		StaleDocs:        len(r.StaleDocs),
		UndocumentedRefs: len(r.UndocumentedRefs),
	}

	if r.Summary.TotalFiles > 0 {
		r.Summary.CoveragePercent = float64(r.Summary.DocumentedFiles) / float64(r.Summary.TotalFiles) * 100
	}
}

func (r *Report) CalculateDirectoryCoverage(allFiles []string, documentedFiles map[string]bool, depth int) {
	if depth <= 0 {
		return
	}

	type dirStats struct {
		total      int
		documented int
	}
	stats := make(map[string]*dirStats)

	for _, file := range allFiles {
		dir := extractDirectory(file, depth)
		if stats[dir] == nil {
			stats[dir] = &dirStats{}
		}
		stats[dir].total++
		if documentedFiles[file] {
			stats[dir].documented++
		}
	}

	r.DirectoryCoverage = make([]DirectoryCoverage, 0, len(stats))
	for path, s := range stats {
		pct := 0.0
		if s.total > 0 {
			pct = float64(s.documented) / float64(s.total) * 100
		}
		r.DirectoryCoverage = append(r.DirectoryCoverage, DirectoryCoverage{
			Path:            path,
			TotalFiles:      s.total,
			DocumentedFiles: s.documented,
			CoveragePercent: pct,
		})
	}

	sort.Slice(r.DirectoryCoverage, func(i, j int) bool {
		return r.DirectoryCoverage[i].Path < r.DirectoryCoverage[j].Path
	})
}

func extractDirectory(filePath string, depth int) string {
	parts := strings.Split(filePath, "/")
	if len(parts) == 1 {
		return "."
	}
	if len(parts)-1 <= depth {
		return strings.Join(parts[:len(parts)-1], "/") + "/"
	}
	return strings.Join(parts[:depth], "/") + "/"
}
