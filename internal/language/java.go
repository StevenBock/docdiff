package language

import "regexp"

type JavaStrategy struct {
	BaseStrategy
}

func NewJavaStrategy() *JavaStrategy {
	return &JavaStrategy{
		BaseStrategy: BaseStrategy{
			name:       "java",
			extensions: []string{".java"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?s)/\*\*.*?\*/`),
				regexp.MustCompile(`//[^\n]*`),
				regexp.MustCompile(`(?s)/\*.*?\*/`),
			},
		},
	}
}

func (j *JavaStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return j.ExtractFromPatterns(content, tag, j.patterns)
}
