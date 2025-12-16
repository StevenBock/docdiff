package language

import (
	"testing"
)

func TestGoStrategy(t *testing.T) {
	strategy := NewGoStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "go" {
			t.Errorf("Name() = %v, want go", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		if len(exts) != 1 || exts[0] != ".go" {
			t.Errorf("Extensions() = %v, want [.go]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "single line comment",
			content: `package main

// @doc docs/API.md
func main() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "block comment",
			content: `package main

/* @doc docs/API.md */
func main() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "multiline block comment",
			content: `package main

/*
@doc docs/API.md
@doc docs/GUIDE.md
*/
func main() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/GUIDE.md"},
		},
		{
			name: "godoc style comment",
			content: `package main

// Handler processes requests.
// @doc docs/API.md
func Handler() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name:    "no annotations",
			content: `package main`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "multiple files same doc",
			content: `package main

// @doc docs/API.md
func Handler1() {}

// @doc docs/API.md
func Handler2() {}`,
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
