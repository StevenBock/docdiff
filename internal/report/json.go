package report

import "encoding/json"

type JSONFormatter struct{}

type jsonOutput struct {
	Metadata      map[string]string      `json:"metadata"`
	StaleDocs     map[string]*StaleDoc   `json:"stale_docs"`
	FilesByDoc    map[string][]string    `json:"files_by_doc"`
	OrphanedFiles []string               `json:"orphaned_files"`
	Summary       Summary                `json:"summary"`
}

func (j *JSONFormatter) Format(report *Report) ([]byte, error) {
	output := jsonOutput{
		Metadata:      report.Metadata,
		StaleDocs:     report.StaleDocs,
		FilesByDoc:    report.FilesByDoc,
		OrphanedFiles: report.OrphanedFiles,
		Summary:       report.Summary,
	}

	return json.MarshalIndent(output, "", "  ")
}
