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
	// DocRefs maps each doc that was READ as content to the set of source
	// files it mentions. A doc present with an empty set genuinely mentions
	// nothing; a doc absent from the map was never read (missing/non-markdown).
	DocRefs map[string]map[string]bool
	// DocScopeMarkers maps each doc read as content to its section markers
	// (`<!-- @scope name -->`), in order of appearance, deduplicated. Markers
	// are opt-in doc-side mirrors of code annotation scopes.
	DocScopeMarkers map[string][]string
}

func NewResult() *Result {
	return &Result{
		Annotations:      make(map[string]*Annotation),
		FilesByDoc:       make(map[string][]string),
		AllFiles:         make([]string, 0),
		Errors:           make([]error, 0),
		UndocumentedRefs: make([]UndocumentedRef, 0),
		DocRefs:          make(map[string]map[string]bool),
		DocScopeMarkers:  make(map[string][]string),
	}
}

func (r *Result) AddAnnotation(filePath string, details []language.DocAnnotation, lang string) {
	docPaths := make([]string, 0, len(details))
	seen := make(map[string]bool)
	for _, d := range details {
		if d.Path == language.Terminator {
			continue // region boundary, not a doc link
		}
		if !seen[d.Path] {
			seen[d.Path] = true
			docPaths = append(docPaths, d.Path)
		}
	}
	if len(docPaths) == 0 {
		return // only terminators: the file links no docs (stays orphaned)
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
