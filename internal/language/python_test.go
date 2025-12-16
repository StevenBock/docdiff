package language

import (
	"testing"
)

func TestPythonStrategy(t *testing.T) {
	strategy := NewPythonStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "python" {
			t.Errorf("Name() = %v, want python", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		expected := map[string]bool{".py": true, ".pyw": true}
		for _, ext := range exts {
			if !expected[ext] {
				t.Errorf("Unexpected extension: %v", ext)
			}
		}
		if len(exts) != 2 {
			t.Errorf("Extensions() = %v, want [.py, .pyw]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "hash comment",
			content: `# @doc docs/API.md
def main():
    pass`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "docstring double quotes",
			content: `"""
Module for API handling
@doc docs/API.md
"""
def main():
    pass`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "docstring single quotes",
			content: `'''
Module for API handling
@doc docs/API.md
'''
def main():
    pass`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "multiple hash comments",
			content: `# @doc docs/API.md
# @doc docs/GUIDE.md
def main():
    pass`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/GUIDE.md"},
		},
		{
			name: "function docstring",
			content: `def main():
    """
    Main function
    @doc docs/API.md
    """
    pass`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name:    "no annotations",
			content: `def main(): pass`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "inline comment",
			content: `def main():  # @doc docs/API.md
    pass`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.ExtractAnnotations([]byte(tt.content), tt.tag)

			if len(got) != len(tt.want) {
				t.Errorf("ExtractAnnotations() got %v, want %v", got, tt.want)
				return
			}

			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("ExtractAnnotations()[%d] = %v, want %v", i, v, tt.want[i])
				}
			}
		})
	}
}
