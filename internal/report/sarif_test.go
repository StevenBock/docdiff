package report

import (
	"encoding/json"
	"testing"

	"github.com/StevenBock/docdiff/internal/metadata"
)

func TestSARIFFormatter_Format(t *testing.T) {
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
	}

	f := &SARIFFormatter{Version: "1.0.0"}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(output, &sarif); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	t.Run("has schema", func(t *testing.T) {
		schema, ok := sarif["$schema"].(string)
		if !ok || schema == "" {
			t.Error("Should have $schema field")
		}
	})

	t.Run("has version", func(t *testing.T) {
		version, ok := sarif["version"].(string)
		if !ok || version != "2.1.0" {
			t.Errorf("version = %v, want 2.1.0", version)
		}
	})

	t.Run("has runs", func(t *testing.T) {
		runs, ok := sarif["runs"].([]interface{})
		if !ok || len(runs) != 1 {
			t.Fatal("Should have exactly 1 run")
		}

		run := runs[0].(map[string]interface{})

		tool := run["tool"].(map[string]interface{})
		driver := tool["driver"].(map[string]interface{})

		if driver["name"] != "docdiff" {
			t.Errorf("tool name = %v, want docdiff", driver["name"])
		}
		if driver["version"] != "1.0.0" {
			t.Errorf("tool version = %v, want 1.0.0", driver["version"])
		}

		rules := driver["rules"].([]interface{})
		if len(rules) != 2 {
			t.Fatalf("Should have 2 rules, got %d", len(rules))
		}

		rule := rules[0].(map[string]interface{})
		if rule["id"] != "stale-doc" {
			t.Errorf("rule[0] id = %v, want stale-doc", rule["id"])
		}

		rule2 := rules[1].(map[string]interface{})
		if rule2["id"] != "undocumented-ref" {
			t.Errorf("rule[1] id = %v, want undocumented-ref", rule2["id"])
		}
	})

	t.Run("has results for stale docs", func(t *testing.T) {
		runs := sarif["runs"].([]interface{})
		run := runs[0].(map[string]interface{})
		results := run["results"].([]interface{})

		if len(results) != 1 {
			t.Fatalf("Should have 1 result (1 stale doc), got %d", len(results))
		}

		result := results[0].(map[string]interface{})
		if result["ruleId"] != "stale-doc" {
			t.Errorf("result ruleId = %v, want stale-doc", result["ruleId"])
		}

		message := result["message"].(map[string]interface{})
		text := message["text"].(string)
		if text == "" {
			t.Error("Result should have message text")
		}

		locations := result["locations"].([]interface{})
		if len(locations) != 1 {
			t.Fatalf("Result should have 1 location")
		}

		location := locations[0].(map[string]interface{})
		physical := location["physicalLocation"].(map[string]interface{})
		artifact := physical["artifactLocation"].(map[string]interface{})

		if artifact["uri"] != "docs/API.md" {
			t.Errorf("location uri = %v, want docs/API.md", artifact["uri"])
		}
	})
}

func TestSARIFFormatter_Format_NoStale(t *testing.T) {
	r := &Report{
		Metadata:  metadata.DocVersions{"docs/API.md": "abc123"},
		StaleDocs: map[string]*StaleDoc{},
	}

	f := &SARIFFormatter{}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(output, &sarif); err != nil {
		t.Fatalf("Failed to unmarshal SARIF output: %v", err)
	}

	runs := sarif["runs"].([]interface{})
	run := runs[0].(map[string]interface{})
	results := run["results"].([]interface{})

	if len(results) != 0 {
		t.Errorf("Should have 0 results when no stale docs, got %d", len(results))
	}
}

func TestSARIFFormatter_Format_DefaultVersion(t *testing.T) {
	r := NewReport()

	f := &SARIFFormatter{}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(output, &sarif); err != nil {
		t.Fatalf("Failed to unmarshal SARIF output: %v", err)
	}

	runs := sarif["runs"].([]interface{})
	run := runs[0].(map[string]interface{})
	tool := run["tool"].(map[string]interface{})
	driver := tool["driver"].(map[string]interface{})

	if driver["version"] != "0.1.0" {
		t.Errorf("Default version = %v, want 0.1.0", driver["version"])
	}
}

func TestSARIFFormatter_Format_MultipleStale(t *testing.T) {
	r := &Report{
		Metadata: metadata.DocVersions{
			"docs/API.md":   "abc123",
			"docs/GUIDE.md": "def456",
		},
		StaleDocs: map[string]*StaleDoc{
			"docs/API.md": {
				Path:         "docs/API.md",
				FilesChanged: 2,
			},
			"docs/GUIDE.md": {
				Path:         "docs/GUIDE.md",
				FilesChanged: 1,
			},
		},
	}

	f := &SARIFFormatter{}
	output, err := f.Format(r)

	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(output, &sarif); err != nil {
		t.Fatalf("Failed to unmarshal SARIF output: %v", err)
	}

	runs := sarif["runs"].([]interface{})
	run := runs[0].(map[string]interface{})
	results := run["results"].([]interface{})

	if len(results) != 2 {
		t.Errorf("Should have 2 results for 2 stale docs, got %d", len(results))
	}
}
