package scanner

type Annotation struct {
	FilePath string
	DocPaths []string
	Language string
}

type Result struct {
	Annotations map[string]*Annotation
	FilesByDoc  map[string][]string
	AllFiles    []string
	Errors      []error
}

func NewResult() *Result {
	return &Result{
		Annotations: make(map[string]*Annotation),
		FilesByDoc:  make(map[string][]string),
		AllFiles:    make([]string, 0),
		Errors:      make([]error, 0),
	}
}

func (r *Result) AddAnnotation(filePath string, docPaths []string, language string) {
	r.Annotations[filePath] = &Annotation{
		FilePath: filePath,
		DocPaths: docPaths,
		Language: language,
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
