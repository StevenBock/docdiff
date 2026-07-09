package language

// @doc CLAUDE.md

import (
	"bytes"
	"regexp"
	"strconv"
)

// DocAnnotation is a single @doc reference with its position and optional scope.
// Scope (an optional "#name" suffix, e.g. `@doc docs/X.md #settings.general`)
// narrows ownership: a scoped annotation owns the code region from its line to
// the next annotation, so a change elsewhere in a central file doesn't flag it.
// An empty Scope means whole-file ownership (the original behavior).
type DocAnnotation struct {
	Path  string
	Scope string
	Line  int // 1-based line where the annotation appears
}

type Strategy interface {
	Name() string
	Extensions() []string
	CommentPatterns() []*regexp.Regexp
	ExtractAnnotations(content []byte, tag string) []string
	ExtractDetailed(content []byte, tag string) []DocAnnotation
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

// ExtractDetailed returns every @doc annotation with its scope and line, using
// the strategy's own comment patterns. All strategies embed BaseStrategy, so
// they inherit this for free.
func (b *BaseStrategy) ExtractDetailed(content []byte, tag string) []DocAnnotation {
	return extractDetailed(content, tag, b.patterns)
}

// ExtractFromPatterns keeps the original path-only, deduped-by-path contract
// (used by strategies and their tests). It derives from extractDetailed so the
// parsing logic lives in one place.
func (b *BaseStrategy) ExtractFromPatterns(content []byte, tag string, patterns []*regexp.Regexp) []string {
	var annotations []string
	seen := make(map[string]bool)
	for _, a := range extractDetailed(content, tag, patterns) {
		if !seen[a.Path] {
			seen[a.Path] = true
			annotations = append(annotations, a.Path)
		}
	}
	return annotations
}

// extractDetailed matches `@doc <path>` with an optional `#<scope>` suffix
// inside each comment, recording the line the annotation sits on. The scope is
// #-prefixed so trailing prose after a bare `@doc path` is never mistaken for a
// scope. Exact (path, scope, line) duplicates are collapsed.
func extractDetailed(content []byte, tag string, patterns []*regexp.Regexp) []DocAnnotation {
	tagPattern := regexp.MustCompile(regexp.QuoteMeta(tag) + `\s+(\S+)(?:\s+#(\S+))?`)

	var out []DocAnnotation
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		for _, loc := range pattern.FindAllIndex(content, -1) {
			comment := content[loc[0]:loc[1]]
			for _, tm := range tagPattern.FindAllSubmatchIndex(comment, -1) {
				path := string(comment[tm[2]:tm[3]])
				scope := ""
				if tm[4] >= 0 {
					scope = string(comment[tm[4]:tm[5]])
				}
				line := 1 + bytes.Count(content[:loc[0]+tm[0]], []byte("\n"))
				key := path + "\x00" + scope + "\x00" + strconv.Itoa(line)
				if seen[key] {
					continue
				}
				seen[key] = true
				out = append(out, DocAnnotation{Path: path, Scope: scope, Line: line})
			}
		}
	}
	return out
}
