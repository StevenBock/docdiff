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

	patternStr := `(?:^|[^a-zA-Z0-9_./\\-])` +
		`((?:\./|\.\./)?)` +
		`((?:[a-zA-Z0-9_.-]+[/\\])*` +
		`[a-zA-Z0-9_.-]+\.` +
		`(?:` + extGroup + `))` +
		`(?:[^a-zA-Z0-9_./\\-]|$)`

	return regexp.MustCompile(patternStr)
}

func (p *Parser) Parse(content []byte) []FileReference {
	if p.pattern == nil {
		return nil
	}

	var refs []FileReference
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNum := 0
	inCodeBlock := false

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		matches := p.pattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			prefix := match[1]
			path := match[2]

			fullPath := prefix + path
			fullPath = normalizePath(fullPath)

			if isURL(line, fullPath) {
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

	return refs
}

func normalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "./")
	return path
}

func isURL(line, path string) bool {
	idx := strings.Index(line, path)
	if idx <= 0 {
		return false
	}

	prefix := line[:idx]
	return strings.HasSuffix(prefix, "://") ||
		strings.HasSuffix(prefix, "http://") ||
		strings.HasSuffix(prefix, "https://")
}
