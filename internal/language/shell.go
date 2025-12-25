package language

import "regexp"

type ShellStrategy struct {
	BaseStrategy
}

func NewShellStrategy() *ShellStrategy {
	return &ShellStrategy{
		BaseStrategy: BaseStrategy{
			name:       "shell",
			extensions: []string{".sh", ".bash"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`#[^\n]*`),
			},
		},
	}
}

func (s *ShellStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return s.ExtractFromPatterns(content, tag, s.patterns)
}
