package scanner

import (
	"os/exec"
	"testing"

	"github.com/StevenBock/docdiff/internal/config"
	"github.com/StevenBock/docdiff/internal/language"
)

func gitInit(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
}

func contains(files []string, want string) bool {
	for _, f := range files {
		if f == want {
			return true
		}
	}
	return false
}

func TestScanIgnore(t *testing.T) {
	files := map[string]string{
		"src/app.go":         "package main\n// @doc docs/A.md\nfunc main() {}",
		"generated/cache.go": "package gen\nfunc X() {}",
		"internal/notes.go":  "package internal\nfunc N() {}",
		".gitignore":         "generated/\n",
		".docdiffignore":     "# extra excludes\nnotes.go\n",
	}
	tmpDir := setupTestDir(t, files)
	gitInit(t, tmpDir)

	registry := language.DefaultRegistry()

	t.Run("gitignore on by default skips ignored + .docdiffignore", func(t *testing.T) {
		result, err := New(config.DefaultConfig(), registry).Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		if !contains(result.AllFiles, "src/app.go") {
			t.Errorf("expected src/app.go to be scanned, got %v", result.AllFiles)
		}
		if contains(result.AllFiles, "generated/cache.go") {
			t.Error("gitignored generated/cache.go should be skipped")
		}
		if contains(result.AllFiles, "internal/notes.go") {
			t.Error(".docdiffignore notes.go should be skipped")
		}
	})

	t.Run("respect_gitignore=false scans ignored files", func(t *testing.T) {
		cfg := config.DefaultConfig()
		off := false
		cfg.RespectGitignore = &off
		result, err := New(cfg, registry).Scan(tmpDir)
		if err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		if !contains(result.AllFiles, "generated/cache.go") {
			t.Error("with respect_gitignore=false, generated/cache.go should be scanned")
		}
		// .docdiffignore is independent of gitignore — still excluded.
		if contains(result.AllFiles, "internal/notes.go") {
			t.Error(".docdiffignore should still exclude notes.go")
		}
	})
}

func TestScanSkipsNestedTargetByDefault(t *testing.T) {
	tmpDir := setupTestDir(t, map[string]string{
		"src/app.go":                     "package main\n// @doc docs/A.md\nfunc main() {}",
		"crates/api/target/debug/tmp.rs": "// @doc docs/TARGET.md\nfn main() {}",
	})

	result, err := New(config.DefaultConfig(), language.DefaultRegistry()).Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if contains(result.AllFiles, "crates/api/target/debug/tmp.rs") {
		t.Fatalf("nested target file should be skipped by default, got %v", result.AllFiles)
	}
	if _, ok := result.FilesByDoc["docs/TARGET.md"]; ok {
		t.Fatal("nested target annotation should not be scanned")
	}
}
