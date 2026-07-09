package config

func DefaultConfig() *Config {
	respectGitignore := true
	return &Config{
		RespectGitignore: &respectGitignore,
		AnnotationTag:    "@doc",
		DocsDirectory:    "docs",
		Include:          []string{},
		Exclude: []string{
			"vendor/**",
			"node_modules/**",
			".git/**",
			"dist/**",
			"build/**",
			"target/**",
			"**/target/**",
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
