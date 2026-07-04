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

func TestExtractDetailed_ScopeAndLine(t *testing.T) {
	src := "package main\n" + // line 1
		"// @doc docs/A.md #alpha\n" + // line 2, scoped
		"var A = 1\n" + // line 3
		"// @doc docs/B.md trailing prose here\n" + // line 4, bare (prose is NOT a scope)
		"var B = 2\n" // line 5
	got := NewGoStrategy().ExtractDetailed([]byte(src), "@doc")
	if len(got) != 2 {
		t.Fatalf("got %d annotations, want 2: %+v", len(got), got)
	}
	if got[0].Path != "docs/A.md" || got[0].Scope != "alpha" || got[0].Line != 2 {
		t.Errorf("first = %+v, want {docs/A.md alpha 2}", got[0])
	}
	// Trailing non-# prose must not be captured as a scope.
	if got[1].Path != "docs/B.md" || got[1].Scope != "" || got[1].Line != 4 {
		t.Errorf("second = %+v, want {docs/B.md \"\" 4}", got[1])
	}
}
