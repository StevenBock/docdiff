package scanner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestExcludedDirPatternPrunesDirectory(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		dir     string
	}{
		{
			name:    "explicit nested target",
			pattern: "src-tauri/target/**",
			dir:     "src-tauri/target",
		},
		{
			name:    "any nested target",
			pattern: "**/target/**",
			dir:     "crates/api/target",
		},
		{
			name:    "basename match",
			pattern: "target",
			dir:     "crates/api/target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !isExcludedDir(tt.dir, []string{tt.pattern}) {
				t.Fatalf("isExcludedDir(%q, %q) = false, want true", tt.dir, tt.pattern)
			}
		})
	}
}

func TestScanPrunesExcludedUnreadableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not reliably make directories unreadable on Windows")
	}

	tmpDir := setupTestDir(t, map[string]string{
		"src/app.go": "package main\n// @doc docs/A.md\nfunc main() {}",
		"src-tauri/target/debug/deps/rustcvjsssN.rs": "// @doc docs/TARGET.md\nfn main() {}",
	})

	targetDir := filepath.Join(tmpDir, "src-tauri", "target")
	if err := os.Chmod(targetDir, 0); err != nil {
		t.Fatalf("chmod target dir unreadable: %v", err)
	}
	defer os.Chmod(targetDir, 0755)

	cfg := config.DefaultConfig()
	cfg.Exclude = []string{"src-tauri/target/**"}
	result, err := New(cfg, language.DefaultRegistry()).Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("excluded unreadable dir should be pruned before walk errors, got %v", result.Errors)
	}
	if _, ok := result.FilesByDoc["docs/TARGET.md"]; ok {
		t.Fatal("excluded target annotation should not be scanned")
	}
}

func TestScanPrunesGitignoredUnreadableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits do not reliably make directories unreadable on Windows")
	}

	tmpDir := setupTestDir(t, map[string]string{
		"src/app.go":         "package main\n// @doc docs/A.md\nfunc main() {}",
		"generated/cache.go": "package gen\n// @doc docs/GENERATED.md\nfunc X() {}",
		"generated/deep/transient-rust-temp-file": "package gen\n",
		".gitignore": "generated/\n",
	})
	gitInit(t, tmpDir)

	generatedDir := filepath.Join(tmpDir, "generated")
	if err := os.Chmod(generatedDir, 0); err != nil {
		t.Fatalf("chmod generated dir unreadable: %v", err)
	}
	defer os.Chmod(generatedDir, 0755)

	result, err := New(config.DefaultConfig(), language.DefaultRegistry()).Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("gitignored unreadable dir should be pruned before walk errors, got %v", result.Errors)
	}
	if _, ok := result.FilesByDoc["docs/GENERATED.md"]; ok {
		t.Fatal("gitignored generated annotation should not be scanned")
	}
}
