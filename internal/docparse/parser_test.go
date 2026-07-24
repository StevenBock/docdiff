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

// Fixtures are verbatim lines from the Hangar repo, where these cases produced
// ~10% of 1203 vestigial-link findings on files the docs plainly do name.
func TestParser_FencedCodeIsAReferenceForBacklinksOnly(t *testing.T) {
	exts := []string{".ts", ".vue", ".rs"}
	// docs/frontend/SQUAWKS-UI.md:44
	content := []byte("# Squawks\n\nRun the tests:\n\n```bash\nnpx vitest run src/components/SquawkEditor.test.ts\n```\n")
	p := New([]string{"src/components/SquawkEditor.test.ts"}, exts)

	if refs := p.Parse(content); len(refs) != 0 {
		t.Errorf("Parse must skip fenced code (a command example claims no ownership), got %v", refs)
	}

	refs := p.ParseIncludingCode(content)
	if len(refs) != 1 || refs[0].Path != "src/components/SquawkEditor.test.ts" {
		t.Errorf("ParseIncludingCode must see the fenced path (it routes an agent to the file), got %v", refs)
	}
	if refs[0].Line != 6 {
		t.Errorf("line number should survive the fence, got %d want 6", refs[0].Line)
	}
}

func TestParser_BraceGlobExpansion(t *testing.T) {
	// docs/tauri/RUST-TAURI-LOG-INGESTION.md:36
	const dir = "src-tauri/crates/ingestion-engine/src/parser/"
	names := []string{"json", "logfmt", "regex", "grok", "dissect", "kv", "csv"}
	known := make([]string, 0, len(names))
	for _, n := range names {
		known = append(known, dir+n+".rs")
	}
	content := []byte("| Change an existing parser's output | `" + dir + "{json,logfmt,regex,grok,dissect,kv,csv}.rs` |\n")

	got := make(map[string]bool)
	for _, ref := range New(known, []string{".rs"}).Parse(content) {
		got[ref.Path] = true
	}
	for _, want := range known {
		if !got[want] {
			t.Errorf("brace glob should expand to %s, got %v", want, got)
		}
	}
}

func TestExpandBraces(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a/{x,y}.go", "a/x.go a/y.go"},
		{"a/{x, y}.go", "a/x.go a/y.go"},                         // spaces after commas
		{"plain/path.go", ""},                                    // no braces: nothing to do
		{"{a,b}", "a b"},                                         // no prefix or suffix
		{"pre/{a}.go", "pre/a.go"},                               // single alternative
		{"a/{x,y}.go b/{p,q}.rs", "a/x.go a/y.go b/p.rs b/q.rs"}, // two groups
	}
	for _, tt := range tests {
		if got := expandBraces(tt.in); got != tt.want {
			t.Errorf("expandBraces(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParser_AdjacentPathsSeparatedByOneSpace(t *testing.T) {
	// docs/frontend/TERMINAL-UI.md:91 — two paths sharing a single space.
	known := []string{"src/lib/terminal-color-resolver.test.ts", "src/components/Terminal.test.ts"}
	content := []byte(known[0] + " " + known[1] + "\n")

	refs := New(known, []string{".ts"}).Parse(content)
	if len(refs) != 2 {
		t.Fatalf("both paths should be seen; a shared delimiter must not hide the second, got %v", refs)
	}
}

func TestParser_BoundariesStillReject(t *testing.T) {
	// Removing the context groups must not loosen what counts as a path.
	p := New([]string{"file.go"}, []string{".go"})
	for _, line := range []string{"file.gox", "prefixfile.go", "file.go.bak", "https://x.dev/file.go"} {
		if refs := p.Parse([]byte(line)); len(refs) != 0 {
			t.Errorf("%q must not read as a reference to file.go, got %v", line, refs)
		}
	}
	// Pre-existing limitation, unchanged by the boundary refactor: '.' counts as
	// path continuation, so a filename ending a sentence is not a match. Left
	// as-is deliberately — widening it belongs in its own change, not a
	// bug-fix commit that is supposed to be behaviour-neutral here.
	if refs := p.Parse([]byte("see file.go.")); len(refs) != 0 {
		t.Errorf("trailing-period behaviour changed unintentionally, got %v", refs)
	}
}

func TestParser_PathPrefixResolvesDocRelativePaths(t *testing.T) {
	// docs/tauri/RUST-TAURI-PROCESS-LIFECYCLE.md:13 states "Rust paths are
	// relative to `src-tauri/src/` unless they begin with `crates/`", so both
	// forms appear in the same doc and both must resolve.
	known := []string{
		"src-tauri/src/commands/mod.rs",
		"src-tauri/crates/hangar-core/src/lib.rs",
		"README.md",
	}
	p := New(known, []string{".rs", ".md"})
	content := []byte("Edit `commands/mod.rs`; core entry is `crates/hangar-core/src/lib.rs`. Root: README.md\n")

	got := map[string]bool{}
	for _, ref := range p.ParseWith(content, Options{PathPrefix: "src-tauri/src/"}) {
		got[ref.Path] = true
	}
	for _, want := range known {
		if !got[want] {
			t.Errorf("prefix should resolve %s (literal paths must keep working too), got %v", want, got)
		}
	}

	// Without the prefix declared, only the repo-root-relative path resolves;
	// both doc-relative forms fall through, which is the status quo this opts
	// out of.
	if refs := p.Parse(content); len(refs) != 1 || refs[0].Path != "README.md" {
		t.Errorf("undeclared prefix must leave doc-relative paths unresolved, got %v", refs)
	}
}

func TestParser_WeakMentions(t *testing.T) {
	p := New([]string{"src/views/git/GitLogToolbar.vue"}, []string{".vue"})
	w := p.WeakMentions([]byte("Toolbar lives under `src/views/git/`; see also `GitCommitLog.vue`.\n"), "")

	if !w.Dirs["src/views/git"] {
		t.Errorf("trailing-slash directory should register, got %v", w.Dirs)
	}
	if !w.Basenames["GitCommitLog.vue"] {
		t.Errorf("bare filename should register, got %v", w.Basenames)
	}

	// Naming one file is not naming its folder: a sibling's full path must not
	// register the directory, or dir-only would swallow real gaps.
	w2 := p.WeakMentions([]byte("See src/views/git/GitHistoryView.vue for the log.\n"), "")
	if w2.Dirs["src/views/git"] {
		t.Errorf("a sibling file path must not count as naming the directory, got %v", w2.Dirs)
	}
}
