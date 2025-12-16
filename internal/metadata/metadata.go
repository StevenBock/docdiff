package metadata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type DocVersions map[string]string

type Manager struct {
	path string
}

func New(path string) *Manager {
	return &Manager{path: path}
}

func (m *Manager) Exists() bool {
	_, err := os.Stat(m.path)
	return err == nil
}

func (m *Manager) Load() (DocVersions, error) {
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, err
	}

	var versions DocVersions
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil, err
	}

	return versions, nil
}

func (m *Manager) Save(versions DocVersions) error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(versions, "", "    ")
	if err != nil {
		return err
	}

	data = append(data, '\n')

	return os.WriteFile(m.path, data, 0644)
}

func (m *Manager) Path() string {
	return m.path
}

func (v DocVersions) SortedDocs() []string {
	docs := make([]string, 0, len(v))
	for doc := range v {
		docs = append(docs, doc)
	}
	sort.Strings(docs)
	return docs
}
