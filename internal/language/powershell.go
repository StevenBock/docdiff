package language

import "regexp"

type PowerShellStrategy struct {
	BaseStrategy
}

func NewPowerShellStrategy() *PowerShellStrategy {
	return &PowerShellStrategy{
		BaseStrategy: BaseStrategy{
			name:       "powershell",
			extensions: []string{".ps1", ".psm1"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`#[^\n]*`),
				regexp.MustCompile(`(?s)<#.*?#>`),
			},
		},
	}
}

func (p *PowerShellStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return p.ExtractFromPatterns(content, tag, p.patterns)
}
