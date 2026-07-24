package docparse

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"
)

type FileReference struct {
	Path string
	Line int
}

type Parser struct {
	knownFiles map[string]bool
	extensions []string
	pattern    *regexp.Regexp
}

func New(knownFiles []string, extensions []string) *Parser {
	fileSet := make(map[string]bool, len(knownFiles))
	for _, f := range knownFiles {
		fileSet[f] = true
	}

	extPattern := buildExtensionPattern(extensions)

	return &Parser{
		knownFiles: fileSet,
		extensions: extensions,
		pattern:    extPattern,
	}
}

func buildExtensionPattern(extensions []string) *regexp.Regexp {
	if len(extensions) == 0 {
		return nil
	}

	escapedExts := make([]string, 0, len(extensions))
	for _, ext := range extensions {
		ext = strings.TrimPrefix(ext, ".")
		escapedExts = append(escapedExts, regexp.QuoteMeta(ext))
	}

	extGroup := strings.Join(escapedExts, "|")

	// The surrounding non-path characters are checked separately (see
	// isPathBoundary) rather than matched here: a context group consumes the
	// delimiter, so in "a/x.go a/y.go" the single shared space would be eaten
	// by the first match and every second path would go unseen.
	patternStr := `((?:\./|\.\./)?)` +
		`((?:[a-zA-Z0-9_.-]+[/\\])*` +
		`[a-zA-Z0-9_.-]+\.` +
		`(?:` + extGroup + `))`

	return regexp.MustCompile(patternStr)
}

func isPathChar(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' ||
		b == '_' || b == '.' || b == '/' || b == '\\' || b == '-'
}

// isPathBoundary reports whether line[start:end] is a whole token rather than a
// fragment of a longer path (so "file.gox" does not read as "file.go").
func isPathBoundary(line string, start, end int) bool {
	if start > 0 && isPathChar(line[start-1]) {
		return false
	}
	if end < len(line) && isPathChar(line[end]) {
		return false
	}
	return true
}

// Parse returns the files a doc references in its prose, skipping fenced code
// blocks. This is the "does the doc claim to own this file" reading, so a path
// inside a command example does not count.
func (p *Parser) Parse(content []byte) []FileReference {
	return p.parse(content, false)
}

// ParseIncludingCode also reads fenced code blocks. This is the "can an agent
// get from this doc to that file" reading: `npx vitest run src/Foo.test.ts` in a
// fenced block is a real route to the file even though it claims no ownership.
func (p *Parser) ParseIncludingCode(content []byte) []FileReference {
	return p.parse(content, true)
}

func (p *Parser) parse(content []byte, includeFenced bool) []FileReference {
	if p.pattern == nil {
		return nil
	}

	var refs []FileReference
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	inCodeBlock := false

	collect := func(line string, lineNum int) {
		for _, m := range p.pattern.FindAllStringSubmatchIndex(line, -1) {
			if len(m) < 6 {
				continue
			}

			if !isPathBoundary(line, m[0], m[1]) {
				continue
			}

			fullPath := normalizePath(line[m[2]:m[3]] + line[m[4]:m[5]])

			if isURL(line, m[0]) {
				continue
			}

			if !p.knownFiles[fullPath] {
				continue
			}

			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			refs = append(refs, FileReference{
				Path: fullPath,
				Line: lineNum,
			})
		}
	}

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock && !includeFenced {
			continue
		}

		collect(line, lineNum)
		// Docs list sibling files compactly as `dir/{a,b,c}.go`; the path
		// pattern stops at the brace, so without expanding every listed file
		// reads as unmentioned.
		if expanded := expandBraces(line); expanded != "" {
			collect(expanded, lineNum)
		}
	}

	return refs
}

var bracePattern = regexp.MustCompile(`([A-Za-z0-9_./\\-]*)\{([^{}]+)\}([A-Za-z0-9_./\\-]*)`)

// maxBraceAlternatives caps one group's expansion; a doc listing more sibling
// files than this in a single glob is not the case this is for.
const maxBraceAlternatives = 64

// expandBraces rewrites every `dir/{a,b}.go` group in a line into the plain
// paths it stands for, joined by spaces, so the ordinary path matcher sees
// them. Returns "" when the line has no expandable group.
func expandBraces(line string) string {
	if !strings.Contains(line, "{") {
		return ""
	}
	var out []string
	for _, m := range bracePattern.FindAllStringSubmatch(line, -1) {
		prefix, alts, suffix := m[1], strings.Split(m[2], ","), m[3]
		if len(alts) > maxBraceAlternatives {
			continue
		}
		for _, alt := range alts {
			if alt = strings.TrimSpace(alt); alt != "" {
				out = append(out, prefix+alt+suffix)
			}
		}
	}
	return strings.Join(out, " ")
}

func normalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	return path
}

// isURL reports whether the path starting at idx is really the tail of a URL.
// idx is the match position, not a re-search: the path has been normalized by
// then, so searching for it in the line can miss or find the wrong occurrence.
func isURL(line string, idx int) bool {
	if idx <= 0 {
		return false
	}
	return strings.HasSuffix(line[:idx], "://")
}
