package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("default annotation tag", func(t *testing.T) {
		if cfg.AnnotationTag != "@doc" {
			t.Errorf("AnnotationTag = %s, want @doc", cfg.AnnotationTag)
		}
	})

	t.Run("default docs directory", func(t *testing.T) {
		if cfg.DocsDirectory != "docs" {
			t.Errorf("DocsDirectory = %s, want docs", cfg.DocsDirectory)
		}
	})

	t.Run("default metadata file", func(t *testing.T) {
		if cfg.MetadataFile != "docs/.doc-versions.json" {
			t.Errorf("MetadataFile = %s, want docs/.doc-versions.json", cfg.MetadataFile)
		}
	})

	t.Run("default include is empty", func(t *testing.T) {
		if len(cfg.Include) != 0 {
			t.Errorf("Include = %v, want empty", cfg.Include)
		}
	})

	t.Run("default exclude has entries", func(t *testing.T) {
		if len(cfg.Exclude) == 0 {
			t.Error("Exclude should have default entries")
		}

		hasVendor := false
		for _, e := range cfg.Exclude {
			if e == "vendor/**" {
				hasVendor = true
				break
			}
		}
		if !hasVendor {
			t.Error("Exclude should include vendor/**")
		}
	})

	t.Run("default CI fail on stale", func(t *testing.T) {
		if !cfg.CI.FailOnStale {
			t.Error("CI.FailOnStale should be true by default")
		}
	})

	t.Run("default CI fail on orphaned", func(t *testing.T) {
		if cfg.CI.FailOnOrphaned {
			t.Error("CI.FailOnOrphaned should be false by default")
		}
	})
}

func TestLanguageConfig_IsEnabled(t *testing.T) {
	t.Run("nil enabled returns true", func(t *testing.T) {
		lc := LanguageConfig{}
		if !lc.IsEnabled() {
			t.Error("IsEnabled() should return true when Enabled is nil")
		}
	})

	t.Run("true enabled returns true", func(t *testing.T) {
		enabled := true
		lc := LanguageConfig{Enabled: &enabled}
		if !lc.IsEnabled() {
			t.Error("IsEnabled() should return true when Enabled is true")
		}
	})

	t.Run("false enabled returns false", func(t *testing.T) {
		enabled := false
		lc := LanguageConfig{Enabled: &enabled}
		if lc.IsEnabled() {
			t.Error("IsEnabled() should return false when Enabled is false")
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("no config file returns defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg, err := Load(tmpDir)

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.AnnotationTag != "@doc" {
			t.Errorf("AnnotationTag = %s, want @doc", cfg.AnnotationTag)
		}
	})

	t.Run("load yaml config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := `
annotation_tag: "@track"
docs_directory: documentation
metadata_file: documentation/.versions.json
include:
  - "src/**"
exclude:
  - "test/**"
ci:
  fail_on_stale: false
  fail_on_orphaned: true
`
		err := os.WriteFile(filepath.Join(tmpDir, ".docdiff.yaml"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.AnnotationTag != "@track" {
			t.Errorf("AnnotationTag = %s, want @track", cfg.AnnotationTag)
		}
		if cfg.DocsDirectory != "documentation" {
			t.Errorf("DocsDirectory = %s, want documentation", cfg.DocsDirectory)
		}
		if cfg.MetadataFile != "documentation/.versions.json" {
			t.Errorf("MetadataFile = %s, want documentation/.versions.json", cfg.MetadataFile)
		}
		if len(cfg.Include) != 1 || cfg.Include[0] != "src/**" {
			t.Errorf("Include = %v, want [src/**]", cfg.Include)
		}
		if len(cfg.Exclude) != 1 || cfg.Exclude[0] != "test/**" {
			t.Errorf("Exclude = %v, want [test/**]", cfg.Exclude)
		}
		if cfg.CI.FailOnStale {
			t.Error("CI.FailOnStale should be false")
		}
		if !cfg.CI.FailOnOrphaned {
			t.Error("CI.FailOnOrphaned should be true")
		}
	})

	t.Run("load yml config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := `annotation_tag: "@yml-tag"`
		err := os.WriteFile(filepath.Join(tmpDir, ".docdiff.yml"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.AnnotationTag != "@yml-tag" {
			t.Errorf("AnnotationTag = %s, want @yml-tag", cfg.AnnotationTag)
		}
	})

	t.Run("load json config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := `{"annotation_tag": "@json-tag", "docs_directory": "doc"}`
		err := os.WriteFile(filepath.Join(tmpDir, ".docdiff.json"), []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.AnnotationTag != "@json-tag" {
			t.Errorf("AnnotationTag = %s, want @json-tag", cfg.AnnotationTag)
		}
		if cfg.DocsDirectory != "doc" {
			t.Errorf("DocsDirectory = %s, want doc", cfg.DocsDirectory)
		}
	})

	t.Run("yaml takes precedence over json", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, ".docdiff.yaml"), []byte(`annotation_tag: "@yaml"`), 0644)
		os.WriteFile(filepath.Join(tmpDir, ".docdiff.json"), []byte(`{"annotation_tag": "@json"}`), 0644)

		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.AnnotationTag != "@yaml" {
			t.Errorf("YAML should take precedence: got %s, want @yaml", cfg.AnnotationTag)
		}
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, ".docdiff.yaml"), []byte(`invalid: yaml: content:`), 0644)

		_, err := Load(tmpDir)
		if err == nil {
			t.Error("Load() should return error for invalid YAML")
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, ".docdiff.json"), []byte(`{invalid json}`), 0644)

		_, err := Load(tmpDir)
		if err == nil {
			t.Error("Load() should return error for invalid JSON")
		}
	})
}

func TestConfig_Paths(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("MetadataPath", func(t *testing.T) {
		path := cfg.MetadataPath("/project")
		expected := filepath.Join("/project", "docs/.doc-versions.json")
		if path != expected {
			t.Errorf("MetadataPath() = %s, want %s", path, expected)
		}
	})

	t.Run("DocsPath", func(t *testing.T) {
		path := cfg.DocsPath("/project")
		expected := filepath.Join("/project", "docs")
		if path != expected {
			t.Errorf("DocsPath() = %s, want %s", path, expected)
		}
	})
}

func TestLoad_LanguageConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := `
languages:
  php:
    enabled: false
  python:
    enabled: true
    extensions:
      - ".py"
      - ".pyw"
      - ".pyi"
`
	os.WriteFile(filepath.Join(tmpDir, ".docdiff.yaml"), []byte(configContent), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	t.Run("php disabled", func(t *testing.T) {
		phpCfg, ok := cfg.Languages["php"]
		if !ok {
			t.Fatal("php language config not found")
		}
		if phpCfg.IsEnabled() {
			t.Error("PHP should be disabled")
		}
	})

	t.Run("python enabled with custom extensions", func(t *testing.T) {
		pyCfg, ok := cfg.Languages["python"]
		if !ok {
			t.Fatal("python language config not found")
		}
		if !pyCfg.IsEnabled() {
			t.Error("Python should be enabled")
		}
		if len(pyCfg.Extensions) != 3 {
			t.Errorf("Python extensions = %v, want 3 items", pyCfg.Extensions)
		}
	})
}
