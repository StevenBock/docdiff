package report

import (
	"encoding/json"
	"fmt"
	"sort"
)

type SARIFFormatter struct {
	Version string
}

type sarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Version        string      `json:"version"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	ShortDescription sarifDescription `json:"shortDescription"`
	FullDescription  sarifDescription `json:"fullDescription"`
	DefaultConfig    sarifConfig      `json:"defaultConfiguration"`
}

type sarifDescription struct {
	Text string `json:"text"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string           `json:"ruleId"`
	Message   sarifDescription `json:"message"`
	Locations []sarifLocation  `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

func (s *SARIFFormatter) Format(report *Report) ([]byte, error) {
	version := s.Version
	if version == "" {
		version = "0.1.0"
	}

	sarif := sarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:           "docdiff",
					InformationURI: "https://github.com/StevenBock/docdiff",
					Version:        version,
					Rules: []sarifRule{
						{
							ID:               "stale-doc",
							Name:             "Stale Documentation",
							ShortDescription: sarifDescription{Text: "Documentation may be out of date"},
							FullDescription:  sarifDescription{Text: "The source code referenced by this documentation has changed since the doc was last updated."},
							DefaultConfig:    sarifConfig{Level: "warning"},
						},
					},
				},
			},
			Results: make([]sarifResult, 0),
		}},
	}

	docPaths := make([]string, 0, len(report.StaleDocs))
	for docPath := range report.StaleDocs {
		docPaths = append(docPaths, docPath)
	}
	sort.Strings(docPaths)

	for _, docPath := range docPaths {
		stale := report.StaleDocs[docPath]
		result := sarifResult{
			RuleID: "stale-doc",
			Message: sarifDescription{
				Text: fmt.Sprintf("Documentation '%s' may be out of date. %d files changed since last update (%s).",
					docPath, stale.FilesChanged, stale.LastCommitInfo),
			},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{
						URI: docPath,
					},
				},
			}},
		}
		sarif.Runs[0].Results = append(sarif.Runs[0].Results, result)
	}

	return json.MarshalIndent(sarif, "", "  ")
}
