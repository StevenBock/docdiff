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

func TestReport_CalculateDirectoryCoverage(t *testing.T) {
	t.Run("depth 1", func(t *testing.T) {
		r := NewReport()
		allFiles := []string{
			"src/api/handler.go",
			"src/api/router.go",
			"src/utils/helper.go",
			"main.go",
		}
		documented := map[string]bool{
			"src/api/handler.go": true,
			"src/api/router.go":  true,
		}

		r.CalculateDirectoryCoverage(allFiles, documented, 1)

		if len(r.DirectoryCoverage) != 2 {
			t.Fatalf("expected 2 directories, got %d", len(r.DirectoryCoverage))
		}

		// Results are sorted, so "." comes before "src/"
		if r.DirectoryCoverage[0].Path != "." {
			t.Errorf("first dir = %q, want \".\"", r.DirectoryCoverage[0].Path)
		}
		if r.DirectoryCoverage[0].TotalFiles != 1 || r.DirectoryCoverage[0].DocumentedFiles != 0 {
			t.Errorf("root: got %d/%d, want 0/1", r.DirectoryCoverage[0].DocumentedFiles, r.DirectoryCoverage[0].TotalFiles)
		}

		if r.DirectoryCoverage[1].Path != "src/" {
			t.Errorf("second dir = %q, want \"src/\"", r.DirectoryCoverage[1].Path)
		}
		if r.DirectoryCoverage[1].TotalFiles != 3 || r.DirectoryCoverage[1].DocumentedFiles != 2 {
			t.Errorf("src/: got %d/%d, want 2/3", r.DirectoryCoverage[1].DocumentedFiles, r.DirectoryCoverage[1].TotalFiles)
		}
	})

	t.Run("depth 2", func(t *testing.T) {
		r := NewReport()
		allFiles := []string{
			"src/api/handler.go",
			"src/api/router.go",
			"src/utils/helper.go",
		}
		documented := map[string]bool{
			"src/api/handler.go": true,
		}

		r.CalculateDirectoryCoverage(allFiles, documented, 2)

		if len(r.DirectoryCoverage) != 2 {
			t.Fatalf("expected 2 directories, got %d", len(r.DirectoryCoverage))
		}

		// src/api/ should have 1/2 documented
		if r.DirectoryCoverage[0].Path != "src/api/" {
			t.Errorf("first dir = %q, want \"src/api/\"", r.DirectoryCoverage[0].Path)
		}
		if r.DirectoryCoverage[0].DocumentedFiles != 1 || r.DirectoryCoverage[0].TotalFiles != 2 {
			t.Errorf("src/api/: got %d/%d, want 1/2", r.DirectoryCoverage[0].DocumentedFiles, r.DirectoryCoverage[0].TotalFiles)
		}

		// src/utils/ should have 0/1 documented
		if r.DirectoryCoverage[1].Path != "src/utils/" {
			t.Errorf("second dir = %q, want \"src/utils/\"", r.DirectoryCoverage[1].Path)
		}
	})

	t.Run("depth 0 disables", func(t *testing.T) {
		r := NewReport()
		r.CalculateDirectoryCoverage([]string{"a.go"}, map[string]bool{}, 0)

		if len(r.DirectoryCoverage) != 0 {
			t.Error("depth 0 should not calculate coverage")
		}
	})

	t.Run("empty files", func(t *testing.T) {
		r := NewReport()
		r.CalculateDirectoryCoverage([]string{}, map[string]bool{}, 1)

		if len(r.DirectoryCoverage) != 0 {
			t.Error("empty files should return empty coverage")
		}
	})

	t.Run("calculates percentage", func(t *testing.T) {
		r := NewReport()
		allFiles := []string{"src/a.go", "src/b.go", "src/c.go", "src/d.go"}
		documented := map[string]bool{"src/a.go": true, "src/b.go": true}

		r.CalculateDirectoryCoverage(allFiles, documented, 1)

		if len(r.DirectoryCoverage) != 1 {
			t.Fatalf("expected 1 directory, got %d", len(r.DirectoryCoverage))
		}
		if r.DirectoryCoverage[0].CoveragePercent != 50.0 {
			t.Errorf("coverage = %.1f%%, want 50.0%%", r.DirectoryCoverage[0].CoveragePercent)
		}
	})
}

func TestExtractDirectory(t *testing.T) {
	tests := []struct {
		path  string
		depth int
		want  string
	}{
		{"src/api/handler.go", 1, "src/"},
		{"src/api/handler.go", 2, "src/api/"},
		{"src/api/v1/handler.go", 2, "src/api/"},
		{"src/api/v1/handler.go", 3, "src/api/v1/"},
		{"main.go", 1, "."},
		{"main.go", 2, "."},
		{"a/b.go", 1, "a/"},
		{"a/b.go", 2, "a/"},
		{"a/b/c/d.go", 1, "a/"},
		{"a/b/c/d.go", 3, "a/b/c/"},
	}

	for _, tt := range tests {
		got := extractDirectory(tt.path, tt.depth)
		if got != tt.want {
			t.Errorf("extractDirectory(%q, %d) = %q, want %q", tt.path, tt.depth, got, tt.want)
		}
	}
}
