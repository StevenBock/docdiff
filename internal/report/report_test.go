package report

import (
	"testing"

	"github.com/StevenBock/docdiff/internal/metadata"
)

func TestNewReport(t *testing.T) {
	r := NewReport()

	if r.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
	if r.StaleDocs == nil {
		t.Error("StaleDocs should be initialized")
	}
	if r.FilesByDoc == nil {
		t.Error("FilesByDoc should be initialized")
	}
	if r.OrphanedFiles == nil {
		t.Error("OrphanedFiles should be initialized")
	}
}

func TestReport_CalculateSummary(t *testing.T) {
	r := NewReport()
	r.Metadata = metadata.DocVersions{
		"docs/API.md":   "abc123",
		"docs/GUIDE.md": "def456",
	}
	r.StaleDocs = map[string]*StaleDoc{
		"docs/API.md": {Path: "docs/API.md"},
	}
	r.OrphanedFiles = []string{"src/orphan1.go", "src/orphan2.go"}

	r.CalculateSummary(10, 5)

	if r.Summary.TotalDocs != 2 {
		t.Errorf("TotalDocs = %d, want 2", r.Summary.TotalDocs)
	}
	if r.Summary.TotalFiles != 10 {
		t.Errorf("TotalFiles = %d, want 10", r.Summary.TotalFiles)
	}
	if r.Summary.DocumentedFiles != 5 {
		t.Errorf("DocumentedFiles = %d, want 5", r.Summary.DocumentedFiles)
	}
	if r.Summary.StaleDocs != 1 {
		t.Errorf("StaleDocs = %d, want 1", r.Summary.StaleDocs)
	}
	if r.Summary.OrphanedFiles != 2 {
		t.Errorf("OrphanedFiles = %d, want 2", r.Summary.OrphanedFiles)
	}
	if r.Summary.CoveragePercent != 50.0 {
		t.Errorf("CoveragePercent = %f, want 50.0", r.Summary.CoveragePercent)
	}
}

func TestReport_CalculateSummary_ZeroFiles(t *testing.T) {
	r := NewReport()

	r.CalculateSummary(0, 0)

	if r.Summary.CoveragePercent != 0 {
		t.Errorf("CoveragePercent should be 0 when no files, got %f", r.Summary.CoveragePercent)
	}
}
