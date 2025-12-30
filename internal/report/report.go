package report

import (
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

type Report struct {
	Metadata         metadata.DocVersions
	StaleDocs        map[string]*StaleDoc
	FilesByDoc       map[string][]string
	OrphanedFiles    []string
	UndocumentedRefs []scanner.UndocumentedRef
	Summary          Summary
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
