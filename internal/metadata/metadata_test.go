package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManager_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("returns false when file does not exist", func(t *testing.T) {
		m := New(filepath.Join(tmpDir, "nonexistent.json"))
		if m.Exists() {
			t.Error("Exists() should return false for nonexistent file")
		}
	})

	t.Run("returns true when file exists", func(t *testing.T) {
		path := filepath.Join(tmpDir, "existing.json")
		os.WriteFile(path, []byte("{}"), 0644)

		m := New(path)
		if !m.Exists() {
			t.Error("Exists() should return true for existing file")
		}
	})
}

func TestManager_Save(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates file with versions", func(t *testing.T) {
		path := filepath.Join(tmpDir, "versions.json")
		m := New(path)

		versions := DocVersions{
			"docs/API.md":   "abc123",
			"docs/GUIDE.md": "def456",
		}

		err := m.Save(versions)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read saved file: %v", err)
		}

		if len(content) == 0 {
			t.Error("Saved file is empty")
		}

		if string(content[len(content)-1]) != "\n" {
			t.Error("Saved file should end with newline")
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nested", "dir", "versions.json")
		m := New(path)

		versions := DocVersions{"docs/API.md": "abc123"}

		err := m.Save(versions)
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("Save() did not create file")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "overwrite.json")
		m := New(path)

		m.Save(DocVersions{"docs/OLD.md": "old123"})
		m.Save(DocVersions{"docs/NEW.md": "new456"})

		loaded, err := m.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if _, ok := loaded["docs/OLD.md"]; ok {
			t.Error("Old entry should be gone after overwrite")
		}
		if _, ok := loaded["docs/NEW.md"]; !ok {
			t.Error("New entry should exist after overwrite")
		}
	})
}

func TestManager_Load(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("loads valid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "valid.json")
		content := `{
    "docs/API.md": "abc123",
    "docs/GUIDE.md": "def456"
}`
		os.WriteFile(path, []byte(content), 0644)

		m := New(path)
		versions, err := m.Load()

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if versions["docs/API.md"] != "abc123" {
			t.Errorf("docs/API.md = %s, want abc123", versions["docs/API.md"])
		}
		if versions["docs/GUIDE.md"] != "def456" {
			t.Errorf("docs/GUIDE.md = %s, want def456", versions["docs/GUIDE.md"])
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		m := New(filepath.Join(tmpDir, "nonexistent.json"))
		_, err := m.Load()

		if err == nil {
			t.Error("Load() should return error for nonexistent file")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.json")
		os.WriteFile(path, []byte("{invalid}"), 0644)

		m := New(path)
		_, err := m.Load()

		if err == nil {
			t.Error("Load() should return error for invalid JSON")
		}
	})

	t.Run("loads empty object", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty.json")
		os.WriteFile(path, []byte("{}"), 0644)

		m := New(path)
		versions, err := m.Load()

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if len(versions) != 0 {
			t.Errorf("Expected empty versions, got %v", versions)
		}
	})
}

func TestManager_Path(t *testing.T) {
	path := "/some/path/versions.json"
	m := New(path)

	if m.Path() != path {
		t.Errorf("Path() = %s, want %s", m.Path(), path)
	}
}

func TestDocVersions_SortedDocs(t *testing.T) {
	versions := DocVersions{
		"docs/ZEBRA.md":   "z",
		"docs/APPLE.md":   "a",
		"docs/BANANA.md":  "b",
		"docs/CHERRY.md":  "c",
	}

	sorted := versions.SortedDocs()

	expected := []string{
		"docs/APPLE.md",
		"docs/BANANA.md",
		"docs/CHERRY.md",
		"docs/ZEBRA.md",
	}

	if len(sorted) != len(expected) {
		t.Fatalf("SortedDocs() returned %d items, want %d", len(sorted), len(expected))
	}

	for i, doc := range sorted {
		if doc != expected[i] {
			t.Errorf("SortedDocs()[%d] = %s, want %s", i, doc, expected[i])
		}
	}
}

func TestManager_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "roundtrip.json")
	m := New(path)

	original := DocVersions{
		"docs/API.md":      "abc123",
		"docs/GUIDE.md":    "def456",
		"docs/TUTORIAL.md": "ghi789",
	}

	if err := m.Save(original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := m.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded) != len(original) {
		t.Fatalf("Round trip changed length: got %d, want %d", len(loaded), len(original))
	}

	for doc, hash := range original {
		if loaded[doc] != hash {
			t.Errorf("Round trip changed %s: got %s, want %s", doc, loaded[doc], hash)
		}
	}
}
