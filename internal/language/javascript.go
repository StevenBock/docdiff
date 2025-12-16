package language

import "regexp"

type JavaScriptStrategy struct {
	BaseStrategy
}

func NewJavaScriptStrategy() *JavaScriptStrategy {
	return &JavaScriptStrategy{
		BaseStrategy: BaseStrategy{
			name:       "javascript",
			extensions: []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`//[^\n]*`),
				regexp.MustCompile(`(?s)/\*.*?\*/`),
			},
		},
	}
}

func (j *JavaScriptStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return j.ExtractFromPatterns(content, tag, j.patterns)
}
