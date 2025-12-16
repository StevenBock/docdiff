package language

import "regexp"

type GoStrategy struct {
	BaseStrategy
}

func NewGoStrategy() *GoStrategy {
	return &GoStrategy{
		BaseStrategy: BaseStrategy{
			name:       "go",
			extensions: []string{".go"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`//[^\n]*`),
				regexp.MustCompile(`(?s)/\*.*?\*/`),
			},
		},
	}
}

func (g *GoStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return g.ExtractFromPatterns(content, tag, g.patterns)
}
