package language

import (
	"regexp"
)

type Strategy interface {
	Name() string
	Extensions() []string
	CommentPatterns() []*regexp.Regexp
	ExtractAnnotations(content []byte, tag string) []string
}

type BaseStrategy struct {
	name       string
	extensions []string
	patterns   []*regexp.Regexp
}

func (b *BaseStrategy) Name() string {
	return b.name
}

func (b *BaseStrategy) Extensions() []string {
	return b.extensions
}

func (b *BaseStrategy) CommentPatterns() []*regexp.Regexp {
	return b.patterns
}

func (b *BaseStrategy) ExtractFromPatterns(content []byte, tag string, patterns []*regexp.Regexp) []string {
	var annotations []string
	seen := make(map[string]bool)

	tagPattern := regexp.MustCompile(regexp.QuoteMeta(tag) + `\s+(\S+)`)

	for _, pattern := range patterns {
		matches := pattern.FindAll(content, -1)
		for _, match := range matches {
			tagMatches := tagPattern.FindAllSubmatch(match, -1)
			for _, tm := range tagMatches {
				if len(tm) > 1 {
					doc := string(tm[1])
					if !seen[doc] {
						seen[doc] = true
						annotations = append(annotations, doc)
					}
				}
			}
		}
	}

	return annotations
}
