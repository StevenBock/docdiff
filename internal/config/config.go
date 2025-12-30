package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AnnotationTag string                    `yaml:"annotation_tag" json:"annotation_tag"`
	DocsDirectory string                    `yaml:"docs_directory" json:"docs_directory"`
	MetadataFile  string                    `yaml:"metadata_file" json:"metadata_file"`
	Include       []string                  `yaml:"include" json:"include"`
	Exclude       []string                  `yaml:"exclude" json:"exclude"`
	Languages     map[string]LanguageConfig `yaml:"languages" json:"languages"`
	CI            CIConfig                  `yaml:"ci" json:"ci"`
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

func (c *Config) MetadataPath(rootDir string) string {
	return filepath.Join(rootDir, c.MetadataFile)
}

func (c *Config) DocsPath(rootDir string) string {
	return filepath.Join(rootDir, c.DocsDirectory)
}
