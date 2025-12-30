package graph

import (
	"sort"
	"strings"
)

type NodeType int

const (
	NodeTypeDoc NodeType = iota
	NodeTypeSource
)

type Node struct {
	ID      string
	Path    string
	Label   string
	Type    NodeType
	IsStale bool
}

type Edge struct {
	From    string
	To      string
	IsStale bool
}

type Graph struct {
	Nodes map[string]*Node
	Edges []*Edge
}

type GraphFormatter interface {
	Format(g *Graph) ([]byte, error)
}

func New() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make([]*Edge, 0),
	}
}

func (g *Graph) AddDocNode(path string, isStale bool) {
	id := SanitizeID(path)
	g.Nodes[id] = &Node{
		ID:      id,
		Path:    path,
		Label:   path,
		Type:    NodeTypeDoc,
		IsStale: isStale,
	}
}

func (g *Graph) AddSourceNode(path string) {
	id := SanitizeID(path)
	if _, exists := g.Nodes[id]; exists {
		return
	}
	g.Nodes[id] = &Node{
		ID:    id,
		Path:  path,
		Label: path,
		Type:  NodeTypeSource,
	}
}

func (g *Graph) AddEdge(docPath, sourcePath string, isStale bool) {
	g.Edges = append(g.Edges, &Edge{
		From:    SanitizeID(docPath),
		To:      SanitizeID(sourcePath),
		IsStale: isStale,
	})
}

func Build(filesByDoc map[string][]string, staleDocs map[string]bool) *Graph {
	g := New()

	docs := sortedKeys(filesByDoc)
	for _, doc := range docs {
		files := filesByDoc[doc]
		isStale := staleDocs[doc]
		g.AddDocNode(doc, isStale)

		sortedFiles := make([]string, len(files))
		copy(sortedFiles, files)
		sort.Strings(sortedFiles)

		for _, file := range sortedFiles {
			g.AddSourceNode(file)
			g.AddEdge(doc, file, isStale)
		}
	}

	return g
}

func SanitizeID(path string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		".", "_",
		"-", "_",
		" ", "_",
	)
	return replacer.Replace(path)
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
