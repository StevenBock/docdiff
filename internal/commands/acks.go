package commands

// @doc CLAUDE.md

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// acksFile records docs reviewed at a commit where their linked code changed
// but the doc itself needed no edit. It maps doc path -> floor commit sha:
// staleness is measured from the newer of the doc's own last commit and this
// floor. It lives at the repo root and is meant to be committed and shared.
const acksFile = ".docdiff-acks.json"

func acksPath(rootDir string) string {
	return filepath.Join(rootDir, acksFile)
}

// loadAcks reads the ack floors. A missing file is not an error (empty map).
func loadAcks(rootDir string) (map[string]string, error) {
	data, err := os.ReadFile(acksPath(rootDir))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	acks := map[string]string{}
	if err := json.Unmarshal(data, &acks); err != nil {
		return nil, err
	}
	return acks, nil
}

func saveAcks(rootDir string, acks map[string]string) error {
	data, err := json.MarshalIndent(acks, "", "    ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(acksPath(rootDir), data, 0644)
}
