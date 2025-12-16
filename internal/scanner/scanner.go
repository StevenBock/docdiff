package scanner

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/StevenBock/docdiff/internal/config"
	"github.com/StevenBock/docdiff/internal/filetype"
	"github.com/StevenBock/docdiff/internal/language"
)

type Scanner struct {
	config   *config.Config
	detector *filetype.Detector
}

func New(cfg *config.Config, registry *language.Registry) *Scanner {
	return &Scanner{
		config:   cfg,
		detector: filetype.NewDetector(registry),
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

	return result, err
}

func (s *Scanner) isExcluded(relPath string) bool {
	for _, pattern := range s.config.Exclude {
		if matched, _ := doublestar.Match(pattern, relPath); matched {
			return true
		}
	}
	return false
}

func (s *Scanner) isIncluded(relPath string) bool {
	for _, pattern := range s.config.Include {
		if matched, _ := doublestar.Match(pattern, relPath); matched {
			return true
		}
	}
	return false
}
