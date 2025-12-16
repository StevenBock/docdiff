package language

import (
	"testing"
)

func TestJavaScriptStrategy(t *testing.T) {
	strategy := NewJavaScriptStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "javascript" {
			t.Errorf("Name() = %v, want javascript", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		expected := map[string]bool{
			".js": true, ".jsx": true, ".ts": true,
			".tsx": true, ".mjs": true, ".cjs": true,
		}
		for _, ext := range exts {
			if !expected[ext] {
				t.Errorf("Unexpected extension: %v", ext)
			}
		}
		if len(exts) != 6 {
			t.Errorf("Extensions() has %d items, want 6", len(exts))
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
			content: `// @doc docs/API.md
function main() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "block comment",
			content: `/* @doc docs/API.md */
function main() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "jsdoc style",
			content: `/**
 * Main function
 * @doc docs/API.md
 */
function main() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "typescript",
			content: `// @doc docs/API.md
interface ApiResponse {
  data: string;
}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "jsx component",
			content: `// @doc docs/COMPONENTS.md
export function Button() {
  return <button>Click</button>;
}`,
			tag:  "@doc",
			want: []string{"docs/COMPONENTS.md"},
		},
		{
			name: "es module",
			content: `// @doc docs/API.md
export const handler = () => {};`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name:    "no annotations",
			content: `function main() {}`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "multiple annotations",
			content: `// @doc docs/API.md
// @doc docs/HANDLERS.md
export function handler() {}`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/HANDLERS.md"},
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
