package report

import (
	"strings"
	"testing"

	"github.com/StevenBock/docdiff/internal/metadata"
)

func TestHumanFormatter_Format(t *testing.T) {
	t.Run("full report", func(t *testing.T) {
		r := &Report{
			Metadata: metadata.DocVersions{
				"docs/API.md":   "abc123",
				"docs/GUIDE.md": "def456",
			},
			StaleDocs: map[string]*StaleDoc{
				"docs/API.md": {
					Path:           "docs/API.md",
					LastHash:       "abc123",
					LastCommitInfo: "abc123 (2 days ago)",
					FilesChanged:   3,
				},
			},
			FilesByDoc: map[string][]string{
				"docs/API.md":   {"src/api.go", "src/handler.go"},
				"docs/GUIDE.md": {"src/guide.go"},
			},
			OrphanedFiles: []string{"src/orphan.go"},
			Summary: Summary{
				TotalDocs:       2,
				TotalFiles:      4,
				DocumentedFiles: 3,
				OrphanedFiles:   1,
				StaleDocs:       1,
				CoveragePercent: 75.0,
			},
		}

		f := &HumanFormatter{Tag: "@doc"}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)

		if !strings.Contains(str, "Documentation Coverage Report") {
			t.Error("Should contain title")
		}
		if !strings.Contains(str, "STALE DOCS") {
			t.Error("Should contain stale docs section")
		}
		if !strings.Contains(str, "docs/API.md") {
			t.Error("Should contain stale doc path")
		}
		if !strings.Contains(str, "abc123 (2 days ago)") {
			t.Error("Should contain last commit info")
		}
		if !strings.Contains(str, "Files changed since: 3") {
			t.Error("Should contain files changed count")
		}
		if !strings.Contains(str, "docdiff changes docs/API.md") {
			t.Error("Should contain command hint")
		}
		if !strings.Contains(str, "Orphaned Files") {
			t.Error("Should contain orphaned files section")
		}
		if !strings.Contains(str, "src/orphan.go") {
			t.Error("Should list orphaned files")
		}
		if !strings.Contains(str, "75.0%") {
			t.Error("Should contain coverage percentage")
		}
	})

	t.Run("no stale docs", func(t *testing.T) {
		r := &Report{
			Metadata:      metadata.DocVersions{"docs/API.md": "abc123"},
			StaleDocs:     map[string]*StaleDoc{},
			FilesByDoc:    map[string][]string{"docs/API.md": {"src/api.go"}},
			OrphanedFiles: []string{},
			Summary:       Summary{TotalFiles: 1, DocumentedFiles: 1, CoveragePercent: 100},
		}

		f := &HumanFormatter{}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "No stale docs found") {
			t.Error("Should indicate no stale docs")
		}
	})

	t.Run("stale only mode", func(t *testing.T) {
		r := &Report{
			Metadata: metadata.DocVersions{"docs/API.md": "abc123"},
			StaleDocs: map[string]*StaleDoc{
				"docs/API.md": {
					Path:           "docs/API.md",
					LastCommitInfo: "abc123",
					FilesChanged:   1,
				},
			},
		}

		f := &HumanFormatter{ShowStaleOnly: true}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "STALE DOCS") {
			t.Error("Should show stale docs")
		}
		if strings.Contains(str, "By Documentation File") {
			t.Error("Should not show full report sections")
		}
	})

	t.Run("stale only mode no stale", func(t *testing.T) {
		r := &Report{
			StaleDocs: map[string]*StaleDoc{},
		}

		f := &HumanFormatter{ShowStaleOnly: true}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "No stale docs found") {
			t.Error("Should indicate no stale docs")
		}
	})

	t.Run("orphaned only mode", func(t *testing.T) {
		r := &Report{
			OrphanedFiles: []string{"src/orphan1.go", "src/orphan2.go"},
		}

		f := &HumanFormatter{ShowOrphanedOnly: true, Tag: "@doc"}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "Orphaned Files") {
			t.Error("Should show orphaned files")
		}
		if !strings.Contains(str, "src/orphan1.go") {
			t.Error("Should list orphaned files")
		}
		if strings.Contains(str, "Documentation Coverage Report") {
			t.Error("Should not show full report title")
		}
	})

	t.Run("orphaned only mode no orphans", func(t *testing.T) {
		r := &Report{
			OrphanedFiles: []string{},
		}

		f := &HumanFormatter{ShowOrphanedOnly: true, Tag: "@doc"}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "No orphaned files") {
			t.Error("Should indicate no orphaned files")
		}
	})

	t.Run("truncates long file lists", func(t *testing.T) {
		files := make([]string, 20)
		for i := 0; i < 20; i++ {
			files[i] = "src/file" + string(rune('a'+i)) + ".go"
		}

		r := &Report{
			Metadata:   metadata.DocVersions{"docs/API.md": "abc123"},
			StaleDocs:  map[string]*StaleDoc{},
			FilesByDoc: map[string][]string{"docs/API.md": files},
		}

		f := &HumanFormatter{}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "... and 15 more") {
			t.Error("Should truncate long file lists")
		}
	})

	t.Run("uses default tag when not set", func(t *testing.T) {
		r := &Report{
			OrphanedFiles: []string{"src/orphan.go"},
		}

		f := &HumanFormatter{ShowOrphanedOnly: true}
		output, err := f.Format(r)

		if err != nil {
			t.Fatalf("Format() error = %v", err)
		}

		str := string(output)
		if !strings.Contains(str, "@doc") {
			t.Error("Should use default @doc tag")
		}
	})
}
