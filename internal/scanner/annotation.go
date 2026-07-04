package scanner

import "github.com/StevenBock/docdiff/internal/language"

type Annotation struct {
	FilePath string
	DocPaths []string
	Language string
	Details  []language.DocAnnotation // per-annotation path/scope/line for hunk-level ownership
}

type UndocumentedRef struct {
	DocPath    string `json:"doc_path"`
	SourceFile string `json:"source_file"`
	Line       int    `json:"line"`
}

type Result struct {
	Annotations      map[string]*Annotation
	FilesByDoc       map[string][]string
	AllFiles         []string
	Errors           []error
	UndocumentedRefs []UndocumentedRef
}

func NewResult() *Result {
	return &Result{
		Annotations:      make(map[string]*Annotation),
		FilesByDoc:       make(map[string][]string),
		AllFiles:         make([]string, 0),
		Errors:           make([]error, 0),
		UndocumentedRefs: make([]UndocumentedRef, 0),
	}
}

func (r *Result) AddAnnotation(filePath string, details []language.DocAnnotation, lang string) {
	docPaths := make([]string, 0, len(details))
	seen := make(map[string]bool)
	for _, d := range details {
		if !seen[d.Path] {
			seen[d.Path] = true
			docPaths = append(docPaths, d.Path)
		}
	}

	r.Annotations[filePath] = &Annotation{
		FilePath: filePath,
		DocPaths: docPaths,
		Language: lang,
		Details:  details,
	}

	for _, doc := range docPaths {
		r.FilesByDoc[doc] = append(r.FilesByDoc[doc], filePath)
	}
}

func (r *Result) AddFile(filePath string) {
	r.AllFiles = append(r.AllFiles, filePath)
}

func (r *Result) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

func (r *Result) OrphanedFiles() []string {
	orphaned := make([]string, 0)
	for _, f := range r.AllFiles {
		if _, ok := r.Annotations[f]; !ok {
			orphaned = append(orphaned, f)
		}
	}
	return orphaned
}

func (r *Result) AddUndocumentedRef(docPath, sourceFile string, line int) {
	r.UndocumentedRefs = append(r.UndocumentedRefs, UndocumentedRef{
		DocPath:    docPath,
		SourceFile: sourceFile,
		Line:       line,
	})
}
