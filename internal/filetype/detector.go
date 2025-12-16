package filetype

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/StevenBock/docdiff/internal/language"
)

type Detector struct {
	registry *language.Registry
}

func NewDetector(registry *language.Registry) *Detector {
	return &Detector{registry: registry}
}

func (d *Detector) Detect(path string, content []byte) (language.Strategy, bool) {
	if strategy := d.detectShebang(content); strategy != nil {
		return strategy, true
	}

	if strategy := d.detectModeline(content); strategy != nil {
		return strategy, true
	}

	if strategy := d.detectExtension(path); strategy != nil {
		return strategy, true
	}

	return d.detectContent(content)
}

var shebangPattern = regexp.MustCompile(`^#!\s*(?:/usr/bin/env\s+)?(?:[^\s]+/)?([^\s/]+)`)

var shebangMap = map[string]string{
	"python":  "python",
	"python3": "python",
	"python2": "python",
	"ruby":    "ruby",
	"node":    "javascript",
	"nodejs":  "javascript",
	"ts-node": "javascript",
	"deno":    "javascript",
	"php":     "php",
}

func (d *Detector) detectShebang(content []byte) language.Strategy {
	if len(content) < 2 || content[0] != '#' || content[1] != '!' {
		return nil
	}

	firstLine := content
	if idx := bytes.IndexByte(content, '\n'); idx != -1 {
		firstLine = content[:idx]
	}

	matches := shebangPattern.FindSubmatch(firstLine)
	if len(matches) < 2 {
		return nil
	}

	interpreter := string(matches[1])
	if langName, ok := shebangMap[interpreter]; ok {
		if strategy, ok := d.registry.GetByName(langName); ok {
			return strategy
		}
	}

	return nil
}

var vimModelinePattern = regexp.MustCompile(`(?:vim?|ex):\s*(?:set\s+)?(?:.*\s)?(?:ft|filetype)=(\w+)`)
var emacsModelinePattern = regexp.MustCompile(`-\*-\s*(?:mode:\s*)?(\w+).*-\*-`)

var modelineMap = map[string]string{
	"python":     "python",
	"ruby":       "ruby",
	"javascript": "javascript",
	"js":         "javascript",
	"typescript": "javascript",
	"ts":         "javascript",
	"php":        "php",
	"go":         "go",
	"golang":     "go",
	"java":       "java",
}

func (d *Detector) detectModeline(content []byte) language.Strategy {
	searchArea := content
	if len(content) > 2000 {
		// Must copy to avoid corrupting original content slice
		searchArea = make([]byte, 2000)
		copy(searchArea[:1000], content[:1000])
		copy(searchArea[1000:], content[len(content)-1000:])
	}

	if matches := vimModelinePattern.FindSubmatch(searchArea); len(matches) > 1 {
		ft := strings.ToLower(string(matches[1]))
		if langName, ok := modelineMap[ft]; ok {
			if strategy, ok := d.registry.GetByName(langName); ok {
				return strategy
			}
		}
	}

	if matches := emacsModelinePattern.FindSubmatch(searchArea); len(matches) > 1 {
		mode := strings.ToLower(string(matches[1]))
		if langName, ok := modelineMap[mode]; ok {
			if strategy, ok := d.registry.GetByName(langName); ok {
				return strategy
			}
		}
	}

	return nil
}

func (d *Detector) detectExtension(path string) language.Strategy {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return nil
	}
	strategy, _ := d.registry.GetByExtension(ext)
	return strategy
}

type contentHeuristic struct {
	pattern  *regexp.Regexp
	language string
}

var contentHeuristics = []contentHeuristic{
	{regexp.MustCompile(`<\?php`), "php"},
	{regexp.MustCompile(`(?m)^import\s+java\.`), "java"},
	{regexp.MustCompile(`(?m)^package\s+[\w.]+;`), "java"},
	{regexp.MustCompile(`(?m)^import\s+.*\s+from\s+['"]`), "javascript"},
	{regexp.MustCompile(`(?m)^export\s+(?:default\s+)?(?:const|let|var|function|class)`), "javascript"},
	{regexp.MustCompile(`(?:const|let|var|function)\s+\w+`), "javascript"},
	{regexp.MustCompile(`(?m)^package\s+\w+`), "go"},
	{regexp.MustCompile(`(?m)^from\s+\w+\s+import`), "python"},
	{regexp.MustCompile(`(?m)^import\s+\w+`), "python"},
	{regexp.MustCompile(`(?m)^def\s+\w+.*:`), "python"},
	{regexp.MustCompile(`(?m)^class\s+\w+.*:`), "python"},
	{regexp.MustCompile(`(?m)^require\s+['"]`), "ruby"},
	{regexp.MustCompile(`(?m)^module\s+\w+`), "ruby"},
}

func (d *Detector) detectContent(content []byte) (language.Strategy, bool) {
	searchArea := content
	if len(content) > 5000 {
		searchArea = content[:5000]
	}

	for _, h := range contentHeuristics {
		if h.pattern.Match(searchArea) {
			if strategy, ok := d.registry.GetByName(h.language); ok {
				return strategy, true
			}
		}
	}

	return nil, false
}
