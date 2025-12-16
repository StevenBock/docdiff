package report

import (
	"encoding/json"
	"testing"

	"github.com/StevenBock/docdiff/internal/metadata"
)

func TestJSONFormatter_Format(t *testing.T) {
	r := &Report{
		Metadata: metadata.DocVersions{
			"docs/API.md": "abc123",
		},
		StaleDocs: map[string]*StaleDoc{
			"docs/API.md": {
				Path:           "docs/API.md",
				LastHash:       "abc123",
				LastCommitInfo: "abc123 (2 days ago)",
				FilesChanged:   3,
				ChangedFiles:   []string{"src/a.go", "src/b.go", "src/c.go"},
			},
		},
		FilesByDoc: map[string][]string{
			"docs/API.md": {"src/api.go"},
		},
		OrphanedFiles: []string{"src/orphan.go"},
		Summary: Summary{
			TotalDocs:       1,
			TotalFiles:      2,
			DocumentedFiles: 1,
			OrphanedFiles:   1,
			StaleDocs:       1,
			CoveragePercent: 50.0,
		},
	}

	f := &JSONFormatter{}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	t.Run("contains metadata", func(t *testing.T) {
		meta, ok := parsed["metadata"].(map[string]interface{})
		if !ok {
			t.Fatal("metadata field missing or wrong type")
		}
		if meta["docs/API.md"] != "abc123" {
			t.Error("metadata should contain doc versions")
		}
	})

	t.Run("contains stale_docs", func(t *testing.T) {
		stale, ok := parsed["stale_docs"].(map[string]interface{})
		if !ok {
			t.Fatal("stale_docs field missing or wrong type")
		}
		apiStale, ok := stale["docs/API.md"].(map[string]interface{})
		if !ok {
			t.Fatal("stale doc entry missing")
		}
		if apiStale["Path"] != "docs/API.md" {
			t.Error("stale doc should have path")
		}
		if int(apiStale["FilesChanged"].(float64)) != 3 {
			t.Error("stale doc should have files changed count")
		}
	})

	t.Run("contains files_by_doc", func(t *testing.T) {
		files, ok := parsed["files_by_doc"].(map[string]interface{})
		if !ok {
			t.Fatal("files_by_doc field missing or wrong type")
		}
		apiFiles, ok := files["docs/API.md"].([]interface{})
		if !ok {
			t.Fatal("files_by_doc entry missing")
		}
		if len(apiFiles) != 1 {
			t.Error("should have 1 file for docs/API.md")
		}
	})

	t.Run("contains orphaned_files", func(t *testing.T) {
		orphaned, ok := parsed["orphaned_files"].([]interface{})
		if !ok {
			t.Fatal("orphaned_files field missing or wrong type")
		}
		if len(orphaned) != 1 {
			t.Error("should have 1 orphaned file")
		}
	})

	t.Run("contains summary", func(t *testing.T) {
		summary, ok := parsed["summary"].(map[string]interface{})
		if !ok {
			t.Fatal("summary field missing or wrong type")
		}
		if int(summary["TotalDocs"].(float64)) != 1 {
			t.Error("summary should have TotalDocs")
		}
		if summary["CoveragePercent"].(float64) != 50.0 {
			t.Error("summary should have CoveragePercent")
		}
	})
}

func TestJSONFormatter_Format_Empty(t *testing.T) {
	r := NewReport()

	f := &JSONFormatter{}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(output, &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if parsed["metadata"] == nil {
		t.Error("Empty report should still have metadata field")
	}
	if parsed["stale_docs"] == nil {
		t.Error("Empty report should still have stale_docs field")
	}
}

func TestJSONFormatter_Format_PrettyPrint(t *testing.T) {
	r := NewReport()
	r.Metadata["docs/API.md"] = "abc123"

	f := &JSONFormatter{}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	str := string(output)
	if str[0] != '{' {
		t.Error("Should start with {")
	}
	if str[1] != '\n' {
		t.Error("Should be pretty-printed with newlines")
	}
}
