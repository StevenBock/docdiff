package graph

import (
	"strings"
	"testing"
)

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"docs/API.md", "docs_API_md"},
		{"src/api.go", "src_api_go"},
		{"path/to/file-name.txt", "path_to_file_name_txt"},
		{"simple", "simple"},
		{"with spaces.go", "with_spaces_go"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeID(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	g := New()
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.Nodes == nil {
		t.Error("Nodes map is nil")
	}
	if g.Edges == nil {
		t.Error("Edges slice is nil")
	}
}

func TestAddDocNode(t *testing.T) {
	g := New()
	g.AddDocNode("docs/API.md", false)

	node, exists := g.Nodes["docs_API_md"]
	if !exists {
		t.Fatal("Doc node not added")
	}
	if node.Type != NodeTypeDoc {
		t.Error("Wrong node type")
	}
	if node.Path != "docs/API.md" {
		t.Errorf("Wrong path: got %q", node.Path)
	}
	if node.IsStale {
		t.Error("Node should not be stale")
	}
}

func TestAddDocNodeStale(t *testing.T) {
	g := New()
	g.AddDocNode("docs/STALE.md", true)

	node := g.Nodes["docs_STALE_md"]
	if !node.IsStale {
		t.Error("Node should be stale")
	}
}

func TestAddSourceNode(t *testing.T) {
	g := New()
	g.AddSourceNode("src/api.go")

	node, exists := g.Nodes["src_api_go"]
	if !exists {
		t.Fatal("Source node not added")
	}
	if node.Type != NodeTypeSource {
		t.Error("Wrong node type")
	}
}

func TestAddSourceNodeNoDuplicate(t *testing.T) {
	g := New()
	g.AddSourceNode("src/api.go")
	g.AddSourceNode("src/api.go")

	if len(g.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(g.Nodes))
	}
}

func TestAddEdge(t *testing.T) {
	g := New()
	g.AddEdge("docs/API.md", "src/api.go", false)

	if len(g.Edges) != 1 {
		t.Fatalf("Expected 1 edge, got %d", len(g.Edges))
	}
	edge := g.Edges[0]
	if edge.From != "docs_API_md" {
		t.Errorf("Wrong from: %q", edge.From)
	}
	if edge.To != "src_api_go" {
		t.Errorf("Wrong to: %q", edge.To)
	}
	if edge.IsStale {
		t.Error("Edge should not be stale")
	}
}

func TestBuild(t *testing.T) {
	filesByDoc := map[string][]string{
		"docs/API.md":   {"src/api.go", "src/handler.go"},
		"docs/GUIDE.md": {"src/main.go"},
	}
	staleDocs := map[string]bool{
		"docs/API.md": true,
	}

	g := Build(filesByDoc, staleDocs)

	if len(g.Nodes) != 5 {
		t.Errorf("Expected 5 nodes, got %d", len(g.Nodes))
	}

	apiDoc := g.Nodes["docs_API_md"]
	if apiDoc == nil {
		t.Fatal("API doc node missing")
	}
	if !apiDoc.IsStale {
		t.Error("API doc should be stale")
	}

	guideDoc := g.Nodes["docs_GUIDE_md"]
	if guideDoc == nil {
		t.Fatal("GUIDE doc node missing")
	}
	if guideDoc.IsStale {
		t.Error("GUIDE doc should not be stale")
	}

	if len(g.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(g.Edges))
	}
}

func TestBuildEmpty(t *testing.T) {
	g := Build(map[string][]string{}, map[string]bool{})

	if len(g.Nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("Expected 0 edges, got %d", len(g.Edges))
	}
}

func TestDOTFormatter(t *testing.T) {
	filesByDoc := map[string][]string{
		"docs/API.md": {"src/api.go"},
	}
	staleDocs := map[string]bool{
		"docs/API.md": true,
	}

	g := Build(filesByDoc, staleDocs)
	formatter := &DOTFormatter{}
	output, err := formatter.Format(g)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	result := string(output)

	if !strings.Contains(result, "digraph docdiff") {
		t.Error("Missing digraph declaration")
	}
	if !strings.Contains(result, "rankdir=LR") {
		t.Error("Missing rankdir")
	}
	if !strings.Contains(result, "shape=note") {
		t.Error("Missing doc node shape")
	}
	if !strings.Contains(result, "shape=box") {
		t.Error("Missing source node shape")
	}
	if !strings.Contains(result, "fillcolor=\"#ffcccc\"") {
		t.Error("Missing stale highlighting")
	}
	if !strings.Contains(result, "style=dashed") {
		t.Error("Missing stale edge styling")
	}
}

func TestMermaidFormatter(t *testing.T) {
	filesByDoc := map[string][]string{
		"docs/API.md": {"src/api.go"},
	}
	staleDocs := map[string]bool{
		"docs/API.md": true,
	}

	g := Build(filesByDoc, staleDocs)
	formatter := &MermaidFormatter{}
	output, err := formatter.Format(g)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	result := string(output)

	if !strings.Contains(result, "graph LR") {
		t.Error("Missing graph declaration")
	}
	if !strings.Contains(result, "[[") {
		t.Error("Missing doc node syntax")
	}
	if !strings.Contains(result, "(\"") {
		t.Error("Missing source node syntax")
	}
	if !strings.Contains(result, "-.->") {
		t.Error("Missing stale edge syntax")
	}
	if !strings.Contains(result, "style") {
		t.Error("Missing stale node styling")
	}
}

func TestDOTFormatterEmpty(t *testing.T) {
	g := New()
	formatter := &DOTFormatter{}
	output, err := formatter.Format(g)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	result := string(output)
	if !strings.Contains(result, "digraph docdiff") {
		t.Error("Missing digraph declaration for empty graph")
	}
}

func TestMermaidFormatterEmpty(t *testing.T) {
	g := New()
	formatter := &MermaidFormatter{}
	output, err := formatter.Format(g)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	result := string(output)
	if !strings.Contains(result, "graph LR") {
		t.Error("Missing graph declaration for empty graph")
	}
}
