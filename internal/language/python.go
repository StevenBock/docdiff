package language

import "regexp"

type PythonStrategy struct {
	BaseStrategy
}

func NewPythonStrategy() *PythonStrategy {
	return &PythonStrategy{
		BaseStrategy: BaseStrategy{
			name:       "python",
			extensions: []string{".py", ".pyw"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`#[^\n]*`),
				regexp.MustCompile(`(?s)""".*?"""`),
				regexp.MustCompile(`(?s)'''.*?'''`),
			},
		},
	}
}

func (p *PythonStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return p.ExtractFromPatterns(content, tag, p.patterns)
}
