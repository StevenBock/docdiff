package docparse

import (
	"bufio"
	"bytes"
	"path"
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

// Options tunes one reading of a doc.
type Options struct {
	// IncludeFenced reads fenced code blocks too.
	IncludeFenced bool
	// PathPrefix is tried when a path matches no known file, for docs whose
	// paths are relative to a subdirectory rather than the repo root.
	PathPrefix string
}

// Parse returns the files a doc references in its prose, skipping fenced code
// blocks. This is the "does the doc claim to own this file" reading, so a path
// inside a command example does not count.
func (p *Parser) Parse(content []byte) []FileReference {
	return p.ParseWith(content, Options{})
}

// ParseIncludingCode also reads fenced code blocks. This is the "can an agent
// get from this doc to that file" reading: `npx vitest run src/Foo.test.ts` in a
// fenced block is a real route to the file even though it claims no ownership.
func (p *Parser) ParseIncludingCode(content []byte) []FileReference {
	return p.ParseWith(content, Options{IncludeFenced: true})
}

// resolve maps a path as written in a doc onto a known file, falling back to
// the doc's declared prefix. The literal path wins, so a doc with a prefix can
// still name repo-root-relative paths.
func (p *Parser) resolve(raw, prefix string) (string, bool) {
	if p.knownFiles[raw] {
		return raw, true
	}
	// Walk up from the declared prefix. Docs that root their paths at one
	// directory usually keep an exception for a sibling subtree — Hangar's
	// process-lifecycle doc opens with "Rust paths are relative to
	// src-tauri/src/ unless they begin with crates/", two roots in one sentence.
	// Walking up covers both from one declaration, which is why PathPrefix is a
	// string and not a list: a list would make every such doc restate a prefix
	// it already implied. Only reach for a list if a real doc turns up with two
	// roots that are NOT nested.
	for _, cand := range prefixChain(prefix) {
		if p.knownFiles[cand+raw] {
			return cand + raw, true
		}
	}
	return "", false
}

// prefixChain is the declared prefix followed by each of its parents.
func prefixChain(prefix string) []string {
	var out []string
	for prefix != "" {
		out = append(out, prefix)
		trimmed := strings.TrimSuffix(prefix, "/")
		i := strings.LastIndex(trimmed, "/")
		if i < 0 {
			break
		}
		prefix = trimmed[:i+1]
	}
	return out
}

func (p *Parser) ParseWith(content []byte, opts Options) []FileReference {
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

			if isURL(line, m[0]) {
				continue
			}

			fullPath, ok := p.resolve(normalizePath(line[m[2]:m[3]]+line[m[4]:m[5]]), opts.PathPrefix)
			if !ok || seen[fullPath] {
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

		if inCodeBlock && !opts.IncludeFenced {
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

// Weak is what a doc gives short of naming a file outright: the bare filenames
// and the directory paths it mentions. Back-link tiering uses it to separate
// "names nothing at all" from "points at roughly the right place". Ownership
// never does — a directory is not a claim on the files under it.
type Weak struct {
	Basenames map[string]bool
	Dirs      map[string]bool
}

// dirTokenPattern matches multi-segment slash paths with or without a trailing
// slash, so `src/views/git/` and `src/views/git` both count. A sibling file's
// full path matches whole, so it does NOT also register as its own directory —
// naming one file is not naming the folder.
var dirTokenPattern = regexp.MustCompile(`[A-Za-z0-9_.\-]+(?:/[A-Za-z0-9_.\-]+)+/?`)

// WeakMentions reads the whole doc, fenced blocks included: a directory in a
// command example still points an agent at roughly the right place.
func (p *Parser) WeakMentions(content []byte, prefix string) Weak {
	w := Weak{Basenames: map[string]bool{}, Dirs: map[string]bool{}}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			continue
		}
		for _, variant := range []string{line, expandBraces(line)} {
			if variant == "" {
				continue
			}
			if p.pattern != nil {
				for _, m := range p.pattern.FindAllStringSubmatchIndex(variant, -1) {
					if len(m) >= 6 && isPathBoundary(variant, m[0], m[1]) {
						w.Basenames[path.Base(normalizePath(variant[m[2]:m[3]]+variant[m[4]:m[5]]))] = true
					}
				}
			}
			for _, d := range dirTokenPattern.FindAllString(variant, -1) {
				// Every ancestor of every path token, not just bare directory
				// mentions: a doc that names src/views/git/GitLog.vue routes a
				// reader to src/views/git as surely as writing the directory
				// does. This tier only decides advisory-vs-gating, so erring
				// toward "the doc points somewhere useful" is the safe side.
				for d = strings.TrimSuffix(normalizePath(d), "/"); d != "" && d != "." && d != "/"; d = path.Dir(d) {
					w.Dirs[d] = true
					for _, cand := range prefixChain(prefix) {
						w.Dirs[cand+d] = true
					}
				}
			}
		}
	}
	return w
}
