package language

import "regexp"

type RustStrategy struct {
	BaseStrategy
}

func NewRustStrategy() *RustStrategy {
	return &RustStrategy{
		BaseStrategy: BaseStrategy{
			name:       "rust",
			extensions: []string{".rs"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`//[^\n]*`),
				regexp.MustCompile(`(?s)/\*.*?\*/`),
			},
		},
	}
}

func (r *RustStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return r.ExtractFromPatterns(content, tag, r.patterns)
}
