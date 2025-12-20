package language

import "regexp"

type VueStrategy struct {
	BaseStrategy
}

func NewVueStrategy() *VueStrategy {
	return &VueStrategy{
		BaseStrategy: BaseStrategy{
			name:       "vue",
			extensions: []string{".vue"},
			patterns: []*regexp.Regexp{
				regexp.MustCompile(`<!--[\s\S]*?-->`),
				regexp.MustCompile(`//[^\n]*`),
				regexp.MustCompile(`(?s)/\*.*?\*/`),
			},
		},
	}
}

func (v *VueStrategy) ExtractAnnotations(content []byte, tag string) []string {
	return v.ExtractFromPatterns(content, tag, v.patterns)
}
