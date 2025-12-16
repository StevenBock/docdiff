package language

import "regexp"

type RubyStrategy struct {
	BaseStrategy
}

func NewRubyStrategy() *RubyStrategy {
	return &RubyStrategy{
		BaseStrategy: BaseStrategy{
			name:       "ruby",
			extensions: []string{".rb", ".rake"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`#[^\n]*`),
				regexp.MustCompile(`(?s)=begin.*?=end`),
			},
		},
	}
}

func (r *RubyStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return r.ExtractFromPatterns(content, tag, r.patterns)
}
