package docparse

import (
	"testing"
)

func TestParser_Parse(t *testing.T) {
	extensions := []string{".go", ".py", ".js", ".ts", ".java", ".php", ".rb", ".vue"}

	tests := []struct {
		name       string
		content    string
		knownFiles []string
		want       []FileReference
	}{
		{
			name:       "basic path extraction",
			content:    "See src/handler.go for implementation",
			knownFiles: []string{"src/handler.go"},
			want:       []FileReference{{Path: "src/handler.go", Line: 1}},
		},
		{
			name:       "relative path with ./",
			content:    "Check ./internal/utils.py",
			knownFiles: []string{"internal/utils.py"},
			want:       []FileReference{{Path: "internal/utils.py", Line: 1}},
		},
		{
			name:       "multiple paths on same line",
			content:    "See src/a.go and src/b.go",
			knownFiles: []string{"src/a.go", "src/b.go"},
			want: []FileReference{
				{Path: "src/a.go", Line: 1},
				{Path: "src/b.go", Line: 1},
			},
		},
		{
			name:       "paths on different lines",
			content:    "Line 1: src/a.go\nLine 2: src/b.go",
			knownFiles: []string{"src/a.go", "src/b.go"},
			want: []FileReference{
				{Path: "src/a.go", Line: 1},
				{Path: "src/b.go", Line: 2},
			},
		},
		{
			name:       "skip unknown files",
			content:    "See src/handler.go and src/unknown.go",
			knownFiles: []string{"src/handler.go"},
			want:       []FileReference{{Path: "src/handler.go", Line: 1}},
		},
		{
			name: "skip code blocks",
			content: `Check this file:

` + "```go" + `
src/example.go
` + "```" + `

But also src/real.go`,
			knownFiles: []string{"src/example.go", "src/real.go"},
			want:       []FileReference{{Path: "src/real.go", Line: 7}},
		},
		{
			name:       "skip URLs",
			content:    "See https://example.com/foo.go for docs",
			knownFiles: []string{"foo.go"},
			want:       nil,
		},
		{
			name:       "inline code backticks",
			content:    "The file `src/handler.go` handles requests",
			knownFiles: []string{"src/handler.go"},
			want:       []FileReference{{Path: "src/handler.go", Line: 1}},
		},
		{
			name:       "markdown link",
			content:    "[handler](src/handler.go)",
			knownFiles: []string{"src/handler.go"},
			want:       []FileReference{{Path: "src/handler.go", Line: 1}},
		},
		{
			name:       "nested directory path",
			content:    "See internal/api/v1/handler.go",
			knownFiles: []string{"internal/api/v1/handler.go"},
			want:       []FileReference{{Path: "internal/api/v1/handler.go", Line: 1}},
		},
		{
			name:       "dedupe same file mentioned twice",
			content:    "See handler.go and handler.go again",
			knownFiles: []string{"handler.go"},
			want:       []FileReference{{Path: "handler.go", Line: 1}},
		},
		{
			name:       "various extensions",
			content:    "Python: app.py, JS: app.js, Java: App.java",
			knownFiles: []string{"app.py", "app.js", "App.java"},
			want: []FileReference{
				{Path: "app.py", Line: 1},
				{Path: "app.js", Line: 1},
				{Path: "App.java", Line: 1},
			},
		},
		{
			name:       "empty content",
			content:    "",
			knownFiles: []string{"src/handler.go"},
			want:       nil,
		},
		{
			name:       "no matches",
			content:    "No file references here",
			knownFiles: []string{"src/handler.go"},
			want:       nil,
		},
		{
			name:       "file with hyphen and underscore",
			content:    "See my-file_name.go",
			knownFiles: []string{"my-file_name.go"},
			want:       []FileReference{{Path: "my-file_name.go", Line: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New(tt.knownFiles, extensions)
			got := parser.Parse([]byte(tt.content))

			if len(got) != len(tt.want) {
				t.Errorf("Parse() got %d refs, want %d", len(got), len(tt.want))
				t.Errorf("got: %+v", got)
				t.Errorf("want: %+v", tt.want)
				return
			}

			for i, ref := range got {
				if ref.Path != tt.want[i].Path {
					t.Errorf("ref[%d].Path = %q, want %q", i, ref.Path, tt.want[i].Path)
				}
				if ref.Line != tt.want[i].Line {
					t.Errorf("ref[%d].Line = %d, want %d", i, ref.Line, tt.want[i].Line)
				}
			}
		})
	}
}

func TestParser_NoExtensions(t *testing.T) {
	parser := New([]string{"file.go"}, nil)
	refs := parser.Parse([]byte("See file.go"))
	if refs != nil {
		t.Errorf("Parse() with no extensions should return nil, got %v", refs)
	}
}

func TestParser_EmptyKnownFiles(t *testing.T) {
	parser := New(nil, []string{".go"})
	refs := parser.Parse([]byte("See file.go"))
	if len(refs) != 0 {
		t.Errorf("Parse() with empty known files should return empty, got %v", refs)
	}
}

func TestBuildExtensionPattern(t *testing.T) {
	pattern := buildExtensionPattern([]string{".go", ".py"})
	if pattern == nil {
		t.Fatal("buildExtensionPattern returned nil")
	}

	testCases := []struct {
		input string
		match bool
	}{
		{"file.go", true},
		{"path/to/file.py", true},
		{"file.js", false},
	}

	for _, tc := range testCases {
		matched := pattern.MatchString(tc.input)
		if matched != tc.match {
			t.Errorf("pattern.MatchString(%q) = %v, want %v", tc.input, matched, tc.match)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"./src/file.go", "src/file.go"},
		{"src\\file.go", "src/file.go"},
		{".\\src\\file.go", "src/file.go"},
		{"src/file.go", "src/file.go"},
	}

	for _, tt := range tests {
		got := normalizePath(tt.input)
		if got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
