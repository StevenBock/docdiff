package scanner

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

func (s *Scanner) Scan(rootDir string) (*Result, error) {
	result := NewResult()

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}

		relPath = filepath.ToSlash(relPath)

		if s.isExcluded(relPath) {
			return nil
		}

		if len(s.config.Include) > 0 && !s.isIncluded(relPath) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			result.AddError(err)
			return nil
		}

		strategy, ok := s.detector.Detect(path, content)
		if !ok {
			return nil
		}

		result.AddFile(relPath)

		docs := strategy.ExtractAnnotations(content, s.config.AnnotationTag)
		if len(docs) > 0 {
			result.AddAnnotation(relPath, docs, strategy.Name())
		}

		return nil
	})

	if err != nil {
		return result, err
	}

	if err := s.scanDocsForRefs(rootDir, result); err != nil {
		log.Printf("Warning: failed to scan docs for references: %v", err)
	}

	return result, nil
}

func (s *Scanner) isExcluded(relPath string) bool {
	for _, pattern := range s.config.Exclude {
		matched, err := doublestar.Match(pattern, relPath)
		if err != nil {
			log.Printf("Warning: invalid exclude pattern %q: %v", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
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
