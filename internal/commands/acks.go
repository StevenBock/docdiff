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

// ackEntry is one recorded review: the floor commit, and why the doc needed no
// edit. An ack asserts "I read this and it is still true", and six months later
// a bare sha cannot be told apart from a rubber stamp.
type ackEntry struct {
	Commit string `json:"commit"`
	Reason string `json:"reason,omitempty"`
}

// UnmarshalJSON accepts both the original bare-sha form and the object form, so
// a repo full of `{"docs/A.md": "abc123"}` keeps parsing after an upgrade.
func (a *ackEntry) UnmarshalJSON(data []byte) error {
	var sha string
	if err := json.Unmarshal(data, &sha); err == nil {
		a.Commit, a.Reason = sha, ""
		return nil
	}
	type plain ackEntry // shed the method set, or this recurses
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*a = ackEntry(p)
	return nil
}

// MarshalJSON writes the bare-sha form when there is no reason, so adopting
// this version doesn't rewrite every existing entry into a noisier shape.
func (a ackEntry) MarshalJSON() ([]byte, error) {
	if a.Reason == "" {
		return json.Marshal(a.Commit)
	}
	type plain ackEntry
	return json.Marshal(plain(a))
}

func parseAckEntries(data []byte) (map[string]ackEntry, error) {
	entries := map[string]ackEntry{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// loadAckEntries reads the full ack records. A missing file is not an error.
func loadAckEntries(rootDir string) (map[string]ackEntry, error) {
	data, err := os.ReadFile(acksPath(rootDir))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]ackEntry{}, nil
		}
		return nil, err
	}
	return parseAckEntries(data)
}

// loadAcks reads just the floor commits — what staleness computation needs.
func loadAcks(rootDir string) (map[string]string, error) {
	entries, err := loadAckEntries(rootDir)
	if err != nil {
		return nil, err
	}
	acks := make(map[string]string, len(entries))
	for doc, e := range entries {
		acks[doc] = e.Commit
	}
	return acks, nil
}

func saveAcks(rootDir string, acks map[string]ackEntry) error {
	data, err := json.MarshalIndent(acks, "", "    ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(acksPath(rootDir), data, 0644)
}
