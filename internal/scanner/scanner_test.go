package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/StevenBock/docdiff/internal/config"
	"github.com/StevenBock/docdiff/internal/language"
)

func setupTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	return tmpDir
}

func TestScanner_Scan(t *testing.T) {
	registry := language.DefaultRegistry()

	t.Run("finds annotations in PHP files", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/Service.php": `<?php
/**
 * @doc docs/API.md
 */
class Service {}`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.Annotations) != 1 {
			t.Errorf("Expected 1 annotation, got %d", len(result.Annotations))
		}

		ann, ok := result.Annotations["src/Service.php"]
		if !ok {
			t.Fatal("Annotation not found for src/Service.php")
		}

		if len(ann.DocPaths) != 1 || ann.DocPaths[0] != "docs/API.md" {
			t.Errorf("DocPaths = %v, want [docs/API.md]", ann.DocPaths)
		}
	})

	t.Run("finds annotations in multiple languages", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go": `package main
// @doc docs/GO.md
func main() {}`,
			"src/Service.java": `package com.example;
/** @doc docs/JAVA.md */
public class Service {}`,
			"src/service.py": `# @doc docs/PYTHON.md
def main(): pass`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.Annotations) != 3 {
			t.Errorf("Expected 3 annotations, got %d", len(result.Annotations))
		}

		expectedDocs := map[string]string{
			"src/service.go":   "docs/GO.md",
			"src/Service.java": "docs/JAVA.md",
			"src/service.py":   "docs/PYTHON.md",
		}

		for file, expectedDoc := range expectedDocs {
			ann, ok := result.Annotations[file]
			if !ok {
				t.Errorf("Annotation not found for %s", file)
				continue
			}
			if len(ann.DocPaths) != 1 || ann.DocPaths[0] != expectedDoc {
				t.Errorf("%s: DocPaths = %v, want [%s]", file, ann.DocPaths, expectedDoc)
			}
		}
	})

	t.Run("groups files by doc", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/handler1.go": `package main
// @doc docs/API.md
func handler1() {}`,
			"src/handler2.go": `package main
// @doc docs/API.md
func handler2() {}`,
			"src/util.go": `package main
// @doc docs/UTIL.md
func util() {}`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		apiFiles := result.FilesByDoc["docs/API.md"]
		if len(apiFiles) != 2 {
			t.Errorf("docs/API.md should have 2 files, got %d", len(apiFiles))
		}

		utilFiles := result.FilesByDoc["docs/UTIL.md"]
		if len(utilFiles) != 1 {
			t.Errorf("docs/UTIL.md should have 1 file, got %d", len(utilFiles))
		}
	})

	t.Run("tracks all scanned files", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/annotated.go":   `// @doc docs/API.md`,
			"src/unannotated.go": `package main`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.AllFiles) != 2 {
			t.Errorf("AllFiles should have 2 files, got %d", len(result.AllFiles))
		}
	})

	t.Run("identifies orphaned files", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/annotated.go":   `// @doc docs/API.md`,
			"src/orphan1.go":     `package main`,
			"src/orphan2.go":     `package main`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		orphaned := result.OrphanedFiles()
		if len(orphaned) != 2 {
			t.Errorf("OrphanedFiles() should return 2 files, got %d", len(orphaned))
		}
	})

	t.Run("respects exclude patterns", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go":    `// @doc docs/API.md`,
			"vendor/lib/lib.go": `// @doc docs/VENDOR.md`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.Annotations) != 1 {
			t.Errorf("Expected 1 annotation (vendor excluded), got %d", len(result.Annotations))
		}

		if _, ok := result.FilesByDoc["docs/VENDOR.md"]; ok {
			t.Error("Vendor file should be excluded")
		}
	})

	t.Run("respects include patterns", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go": `// @doc docs/SRC.md`,
			"lib/util.go":    `// @doc docs/LIB.md`,
		})

		cfg := config.DefaultConfig()
		cfg.Include = []string{"src/**"}
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.Annotations) != 1 {
			t.Errorf("Expected 1 annotation (only src included), got %d", len(result.Annotations))
		}

		if _, ok := result.FilesByDoc["docs/SRC.md"]; !ok {
			t.Error("src/service.go should be included")
		}
		if _, ok := result.FilesByDoc["docs/LIB.md"]; ok {
			t.Error("lib/util.go should not be included")
		}
	})

	t.Run("uses custom annotation tag", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go": `// @track-doc docs/API.md`,
		})

		cfg := config.DefaultConfig()
		cfg.AnnotationTag = "@track-doc"
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.Annotations) != 1 {
			t.Errorf("Expected 1 annotation with custom tag, got %d", len(result.Annotations))
		}

		if _, ok := result.FilesByDoc["docs/API.md"]; !ok {
			t.Error("Should find annotation with custom tag")
		}
	})

	t.Run("skips .git directory", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go":     `// @doc docs/API.md`,
			".git/config":        `[core]`,
			".git/hooks/pre.go":  `// @doc docs/HOOK.md`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if _, ok := result.FilesByDoc["docs/HOOK.md"]; ok {
			t.Error(".git directory should be skipped")
		}
	})

	t.Run("skips node_modules directory", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/app.js":              `// @doc docs/APP.md`,
			"node_modules/pkg/pkg.js": `// @doc docs/PKG.md`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if _, ok := result.FilesByDoc["docs/PKG.md"]; ok {
			t.Error("node_modules directory should be skipped")
		}
	})

	t.Run("handles multiple annotations per file", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go": `package main
// @doc docs/API.md
// @doc docs/HANDLERS.md
func handler() {}`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		ann := result.Annotations["src/service.go"]
		if len(ann.DocPaths) != 2 {
			t.Errorf("Expected 2 doc paths, got %d", len(ann.DocPaths))
		}

		if len(result.FilesByDoc["docs/API.md"]) != 1 {
			t.Error("docs/API.md should have 1 file")
		}
		if len(result.FilesByDoc["docs/HANDLERS.md"]) != 1 {
			t.Error("docs/HANDLERS.md should have 1 file")
		}
	})

	t.Run("ignores unsupported file types", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"src/service.go":  `// @doc docs/API.md`,
			"src/data.json":   `{"@doc": "docs/JSON.md"}`,
			"src/style.css":   `/* @doc docs/CSS.md */`,
			"docs/README.md":  `# @doc docs/README.md`,
		})

		cfg := config.DefaultConfig()
		s := New(cfg, registry)

		result, err := s.Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}

		if len(result.Annotations) != 1 {
			t.Errorf("Expected 1 annotation (only .go supported), got %d", len(result.Annotations))
		}
	})
}

func TestResult_AddAnnotation(t *testing.T) {
	r := NewResult()

	r.AddAnnotation("src/service.go", []string{"docs/API.md", "docs/GUIDE.md"}, "go")

	if len(r.Annotations) != 1 {
		t.Errorf("Annotations should have 1 entry, got %d", len(r.Annotations))
	}

	ann := r.Annotations["src/service.go"]
	if ann.FilePath != "src/service.go" {
		t.Errorf("FilePath = %s, want src/service.go", ann.FilePath)
	}
	if ann.Language != "go" {
		t.Errorf("Language = %s, want go", ann.Language)
	}
	if len(ann.DocPaths) != 2 {
		t.Errorf("DocPaths = %v, want 2 entries", ann.DocPaths)
	}

	if len(r.FilesByDoc["docs/API.md"]) != 1 {
		t.Error("docs/API.md should have 1 file")
	}
	if len(r.FilesByDoc["docs/GUIDE.md"]) != 1 {
		t.Error("docs/GUIDE.md should have 1 file")
	}
}

func TestResult_AddFile(t *testing.T) {
	r := NewResult()

	r.AddFile("src/service.go")
	r.AddFile("src/handler.go")

	if len(r.AllFiles) != 2 {
		t.Errorf("AllFiles should have 2 entries, got %d", len(r.AllFiles))
	}
}

func TestResult_OrphanedFiles(t *testing.T) {
	r := NewResult()

	r.AddFile("src/annotated.go")
	r.AddFile("src/orphan1.go")
	r.AddFile("src/orphan2.go")

	r.AddAnnotation("src/annotated.go", []string{"docs/API.md"}, "go")

	orphaned := r.OrphanedFiles()

	if len(orphaned) != 2 {
		t.Errorf("OrphanedFiles() should return 2 files, got %d", len(orphaned))
	}

	orphanedSet := make(map[string]bool)
	for _, f := range orphaned {
		orphanedSet[f] = true
	}

	if orphanedSet["src/annotated.go"] {
		t.Error("Annotated file should not be in orphaned list")
	}
	if !orphanedSet["src/orphan1.go"] {
		t.Error("orphan1.go should be in orphaned list")
	}
	if !orphanedSet["src/orphan2.go"] {
		t.Error("orphan2.go should be in orphaned list")
	}
}
