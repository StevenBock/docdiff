package scanner

// @doc CLAUDE.md

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/StevenBock/docdiff/internal/config"
	"github.com/StevenBock/docdiff/internal/docparse"
	"github.com/StevenBock/docdiff/internal/filetype"
	"github.com/StevenBock/docdiff/internal/git"
	"github.com/StevenBock/docdiff/internal/language"
)

type Scanner struct {
	config   *config.Config
	detector *filetype.Detector
	registry *language.Registry
}

func New(cfg *config.Config, registry *language.Registry) *Scanner {
	return &Scanner{
		config:   cfg,
		detector: filetype.NewDetector(registry),
		registry: registry,
	}
}

type candidate struct {
	path    string
	relPath string
}

func (s *Scanner) Scan(rootDir string) (*Result, error) {
	result := NewResult()

	excludes := append([]string{}, s.config.Exclude...)
	excludes = append(excludes, loadDocdiffIgnore(rootDir)...)
	gitignore := newGitignorePruner(rootDir, s.config)

	var candidates []candidate
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == rootDir {
				return err
			}
			result.AddError(err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(rootDir, path, excludes, gitignore) {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}

		relPath = filepath.ToSlash(relPath)

		if isExcluded(relPath, excludes) {
			return nil
		}

		if len(s.config.Include) > 0 && !s.isIncluded(relPath) {
			return nil
		}

		candidates = append(candidates, candidate{path: path, relPath: relPath})
		return nil
	})

	if err != nil {
		return result, err
	}

	candidates = s.filterGitignored(rootDir, candidates)

	for _, c := range candidates {
		content, err := os.ReadFile(c.path)
		if err != nil {
			result.AddError(err)
			continue
		}

		strategy, ok := s.detector.Detect(c.path, content)
		if !ok {
			continue
		}

		result.AddFile(c.relPath)

		details := strategy.ExtractDetailed(content, s.config.AnnotationTag)
		if len(details) > 0 {
			result.AddAnnotation(c.relPath, details, strategy.Name())
		}
	}

	if err := s.scanDocsForRefs(rootDir, result); err != nil {
		log.Printf("Warning: failed to scan docs for references: %v", err)
	}

	return result, nil
}

func shouldSkipDir(rootDir, path string, excludes []string, gitignore *gitignorePruner) bool {
	if path == rootDir {
		return false
	}

	base := filepath.Base(path)
	if base == ".git" || base == "node_modules" || base == "vendor" {
		return true
	}

	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return false
	}
	relPath = filepath.ToSlash(relPath)

	return isExcludedDir(relPath, excludes) || gitignore.IgnoredDir(relPath)
}

type gitignorePruner struct {
	enabled bool
	g       *git.Git
}

func newGitignorePruner(rootDir string, cfg *config.Config) *gitignorePruner {
	p := &gitignorePruner{}
	if !cfg.GitignoreRespected() {
		return p
	}
	g := git.New(rootDir)
	if !g.IsRepo() {
		return p
	}
	p.enabled = true
	p.g = g
	return p
}

func (p *gitignorePruner) IgnoredDir(relPath string) bool {
	if p == nil || !p.enabled {
		return false
	}
	dirPath := strings.TrimSuffix(relPath, "/") + "/"
	ignored, err := p.g.CheckIgnore([]string{dirPath, relPath})
	if err != nil {
		log.Printf("Warning: git check-ignore failed for %s: %v", relPath, err)
		p.enabled = false
		return false
	}
	return ignored[dirPath] || ignored[relPath]
}

// filterGitignored drops candidates that git ignores. Best-effort: outside a
// repo or when respect_gitignore is off, candidates pass through unchanged.
func (s *Scanner) filterGitignored(rootDir string, candidates []candidate) []candidate {
	if !s.config.GitignoreRespected() || len(candidates) == 0 {
		return candidates
	}
	g := git.New(rootDir)
	if !g.IsRepo() {
		return candidates
	}
	rels := make([]string, len(candidates))
	for i, c := range candidates {
		rels[i] = c.relPath
	}
	ignored, err := g.CheckIgnore(rels)
	if err != nil {
		log.Printf("Warning: git check-ignore failed, scanning all files: %v", err)
		return candidates
	}
	kept := candidates[:0]
	for _, c := range candidates {
		if !ignored[c.relPath] {
			kept = append(kept, c)
		}
	}
	return kept
}

// loadDocdiffIgnore reads .docdiffignore as additional exclude glob patterns,
// one per line (blank lines and # comments ignored). Same glob syntax as the
// config `exclude:` list.
func loadDocdiffIgnore(rootDir string) []string {
	data, err := os.ReadFile(filepath.Join(rootDir, ".docdiffignore"))
	if err != nil {
		return nil
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// isExcluded matches relPath against glob patterns. A pattern without a slash
// also matches against the basename at any depth (gitignore-like convenience),
// so `*.lock` or `LICENSE.txt` exclude matching files anywhere.
func isExcluded(relPath string, patterns []string) bool {
	base := filepath.Base(relPath)
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			log.Printf("Warning: invalid exclude pattern %q: %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
		if !strings.Contains(pattern, "/") {
			if m, _ := doublestar.Match(pattern, base); m {
				return true
			}
		}
	}
	return false
}

// isExcludedDir reports whether relPath names a directory that can be pruned
// before walking its children. Patterns ending in /** exclude the directory
// itself too, not just files already enumerated below it.
func isExcludedDir(relPath string, patterns []string) bool {
	relPath = strings.TrimSuffix(relPath, "/")
	if relPath == "" || relPath == "." {
		return false
	}

	base := filepath.Base(relPath)
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		if matchGlob(pattern, relPath) || matchGlob(pattern, relPath+"/") {
			return true
		}

		if strings.HasSuffix(pattern, "/**") {
			parent := strings.TrimSuffix(pattern, "/**")
			if matchGlob(parent, relPath) {
				return true
			}
		}

		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			if matchGlob(dirPattern, relPath) {
				return true
			}
		}

		if !strings.Contains(pattern, "/") && matchGlob(pattern, base) {
			return true
		}
	}
	return false
}

func matchGlob(pattern, path string) bool {
	matched, err := doublestar.Match(pattern, path)
	if err != nil {
		log.Printf("Warning: invalid exclude pattern %q: %v", pattern, err)
		return false
	}
	return matched
}

func (s *Scanner) isIncluded(relPath string) bool {
	for _, pattern := range s.config.Include {
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			log.Printf("Warning: invalid include pattern %q: %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

func (s *Scanner) scanDocsForRefs(rootDir string, result *Result) error {
	docsDir := s.config.DocsPath(rootDir)

	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		return nil
	}

	extensions := s.registry.AllExtensions()
	parser := docparse.New(result.AllFiles, extensions)

	filesWithDocToThis := make(map[string]map[string]bool)
	for doc, files := range result.FilesByDoc {
		for _, f := range files {
			if filesWithDocToThis[doc] == nil {
				filesWithDocToThis[doc] = make(map[string]bool)
			}
			filesWithDocToThis[doc][f] = true
		}
	}

	return filepath.WalkDir(docsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relDocPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}
		relDocPath = filepath.ToSlash(relDocPath)

		refs := parser.Parse(content)
		linkedFiles := filesWithDocToThis[relDocPath]

		for _, ref := range refs {
			if linkedFiles != nil && linkedFiles[ref.Path] {
				continue
			}
			result.AddUndocumentedRef(relDocPath, ref.Path, ref.Line)
		}

		return nil
	})
}
