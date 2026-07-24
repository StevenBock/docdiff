package config

// @doc README.md

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AnnotationTag    string                    `yaml:"annotation_tag" json:"annotation_tag"`
	DocsDirectory    string                    `yaml:"docs_directory" json:"docs_directory"`
	Include          []string                  `yaml:"include" json:"include"`
	Exclude          []string                  `yaml:"exclude" json:"exclude"`
	RespectGitignore *bool                     `yaml:"respect_gitignore" json:"respect_gitignore"`
	Languages        map[string]LanguageConfig `yaml:"languages" json:"languages"`
	CI               CIConfig                  `yaml:"ci" json:"ci"`
	Lint             LintConfig                `yaml:"lint" json:"lint"`
	Docs             map[string]DocPolicy      `yaml:"docs" json:"docs"`
	Check            CheckConfig               `yaml:"check" json:"check"`
}

type CheckConfig struct {
	// Scope sets check's default changed-set source for this repo: "staged"
	// or "working-tree" (the built-in default). A deterministic per-repo
	// default — check never switches scope based on tree state.
	Scope string `yaml:"scope" json:"scope"`
}

// DocPolicy is per-doc behavior declared in the config's `docs:` map.
type DocPolicy struct {
	// Gate inverts what flags the doc when set to "structural": annotation-map
	// changes and linked-file add/delete/rename gate; body edits do not. Docs
	// that inventory files (module maps, ownership indexes) want this.
	Gate string `yaml:"gate" json:"gate"`

	// PathPrefix is what this doc's relative paths hang off. A doc that states
	// "Rust paths are relative to src-tauri/src/" writes `commands/mod.rs`,
	// which resolves against no real file; declaring the prefix here lets the
	// back-link reader find it. Tried only after the literal path fails, so a
	// repo-root-relative path in the same doc keeps working.
	PathPrefix string `yaml:"path_prefix" json:"path_prefix"`
}

// DocPathPrefix returns the declared path prefix for a doc, normalized to end
// in "/" (empty when none is declared).
func (c *Config) DocPathPrefix(doc string) string {
	p := c.Docs[doc].PathPrefix
	if p == "" || strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}

// StructuralDocs returns the docs declared with `gate: structural`.
func (c *Config) StructuralDocs() map[string]bool {
	out := make(map[string]bool)
	for doc, p := range c.Docs {
		if p.Gate == "structural" {
			out[doc] = true
		}
	}
	return out
}

type LintConfig struct {
	// FanOutThreshold is the number of distinct whole-file doc links on one
	// source file at which `lint` flags a fan-out smell. Defaults to 3.
	FanOutThreshold int `yaml:"fan_out_threshold" json:"fan_out_threshold"`
}

func (c *Config) LintFanOutThreshold() int {
	if c.Lint.FanOutThreshold > 0 {
		return c.Lint.FanOutThreshold
	}
	return 3
}

// GitignoreRespected reports whether gitignored files should be skipped during
// scanning. Defaults to true when unset.
func (c *Config) GitignoreRespected() bool {
	if c.RespectGitignore == nil {
		return true
	}
	return *c.RespectGitignore
}

type LanguageConfig struct {
	Enabled         *bool    `yaml:"enabled" json:"enabled"`
	Extensions      []string `yaml:"extensions" json:"extensions"`
	CommentPatterns []string `yaml:"comment_patterns" json:"comment_patterns"`
}

type CIConfig struct {
	FailOnStale            bool `yaml:"fail_on_stale" json:"fail_on_stale"`
	FailOnOrphaned         bool `yaml:"fail_on_orphaned" json:"fail_on_orphaned"`
	FailOnUndocumentedRefs bool `yaml:"fail_on_undocumented_refs" json:"fail_on_undocumented_refs"`
}

func (lc *LanguageConfig) IsEnabled() bool {
	if lc.Enabled == nil {
		return true
	}
	return *lc.Enabled
}

func Load(dir string) (*Config, error) {
	cfg := DefaultConfig()

	yamlPath := filepath.Join(dir, ".docdiff.yaml")
	ymlPath := filepath.Join(dir, ".docdiff.yml")
	jsonPath := filepath.Join(dir, ".docdiff.json")

	var configPath string
	if _, err := os.Stat(yamlPath); err == nil {
		configPath = yamlPath
	} else if _, err := os.Stat(ymlPath); err == nil {
		configPath = ymlPath
	} else if _, err := os.Stat(jsonPath); err == nil {
		configPath = jsonPath
	} else {
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(configPath)
	if ext == ".json" {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func (c *Config) DocsPath(rootDir string) string {
	return filepath.Join(rootDir, c.DocsDirectory)
}
