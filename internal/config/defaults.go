package config

func DefaultConfig() *Config {
	return &Config{
		AnnotationTag: "@doc",
		DocsDirectory: "docs",
		MetadataFile:  "docs/.doc-versions.json",
		Include:       []string{},
		Exclude: []string{
			"vendor/**",
			"node_modules/**",
			".git/**",
			"dist/**",
			"build/**",
			"target/**",
			"__pycache__/**",
			".venv/**",
			"venv/**",
		},
		Languages: make(map[string]LanguageConfig),
		CI: CIConfig{
			FailOnStale:    true,
			FailOnOrphaned: false,
		},
	}
}
