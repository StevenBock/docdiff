package language

import (
	"testing"
)

func TestBaseStrategy_ExtractFromPatterns(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		tag      string
		patterns []string
		want     []string
	}{
		{
			name:     "single annotation",
			content:  "// @doc docs/API.md",
			tag:      "@doc",
			patterns: []string{`//[^\n]*`},
			want:     []string{"docs/API.md"},
		},
		{
			name:     "multiple annotations same line",
			content:  "// @doc docs/API.md @doc docs/GUIDE.md",
			tag:      "@doc",
			patterns: []string{`//[^\n]*`},
			want:     []string{"docs/API.md", "docs/GUIDE.md"},
		},
		{
			name:     "multiple lines",
			content:  "// @doc docs/API.md\n// @doc docs/GUIDE.md",
			tag:      "@doc",
			patterns: []string{`//[^\n]*`},
			want:     []string{"docs/API.md", "docs/GUIDE.md"},
		},
		{
			name:     "no annotations",
			content:  "// This is a comment",
			tag:      "@doc",
			patterns: []string{`//[^\n]*`},
			want:     nil,
		},
		{
			name:     "custom tag",
			content:  "// @track-doc docs/API.md",
			tag:      "@track-doc",
			patterns: []string{`//[^\n]*`},
			want:     []string{"docs/API.md"},
		},
		{
			name:     "deduplicates",
			content:  "// @doc docs/API.md\n// @doc docs/API.md",
			tag:      "@doc",
			patterns: []string{`//[^\n]*`},
			want:     []string{"docs/API.md"},
		},
		{
			name:     "block comment",
			content:  "/* @doc docs/API.md */",
			tag:      "@doc",
			patterns: []string{`(?s)/\*.*?\*/`},
			want:     []string{"docs/API.md"},
		},
		{
			name:     "multiline block comment",
			content:  "/**\n * @doc docs/API.md\n * @doc docs/GUIDE.md\n */",
			tag:      "@doc",
			patterns: []string{`(?s)/\*\*.*?\*/`},
			want:     []string{"docs/API.md", "docs/GUIDE.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewGoStrategy()
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
