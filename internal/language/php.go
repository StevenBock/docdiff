package language

import "regexp"

type PHPStrategy struct {
	BaseStrategy
}

func NewPHPStrategy() *PHPStrategy {
	return &PHPStrategy{
		BaseStrategy: BaseStrategy{
			name:       "php",
			extensions: []string{".php"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`(?s)/\*\*.*?\*/`),
				regexp.MustCompile(`//[^\n]*`),
				regexp.MustCompile(`#[^\n]*`),
			},
		},
	}
}

func (p *PHPStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return p.ExtractFromPatterns(content, tag, p.patterns)
}
