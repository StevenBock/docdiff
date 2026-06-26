package language

import "testing"

func TestRustStrategy(t *testing.T) {
	strategy := NewRustStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "rust" {
			t.Errorf("Name() = %v, want rust", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		if len(exts) != 1 || exts[0] != ".rs" {
			t.Errorf("Extensions() = %v, want [.rs]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "line comment",
			content: `// @doc docs/RUST.md
pub fn run() {}`,
			tag:  "@doc",
			want: []string{"docs/RUST.md"},
		},
		{
			name: "rustdoc line comment",
			content: `/// Handles project lifecycle state.
/// @doc docs/RUST.md
pub fn run() {}`,
			tag:  "@doc",
			want: []string{"docs/RUST.md"},
		},
		{
			name: "block comment",
			content: `/* @doc docs/RUST.md */
pub fn run() {}`,
			tag:  "@doc",
			want: []string{"docs/RUST.md"},
		},
		{
			name: "rustdoc block comment",
			content: `/**
 * @doc docs/RUST.md
 * @doc docs/BACKEND.md
 */
pub fn run() {}`,
			tag:  "@doc",
			want: []string{"docs/RUST.md", "docs/BACKEND.md"},
		},
		{
			name: "inner module doc comment",
			content: `//! Process orchestration.
//! @doc docs/RUST.md

pub mod lifecycle;`,
			tag:  "@doc",
			want: []string{"docs/RUST.md"},
		},
		{
			name: "deduplicates repeated docs",
			content: `// @doc docs/RUST.md
/// @doc docs/RUST.md`,
			tag:  "@doc",
			want: []string{"docs/RUST.md"},
		},
		{
			name:    "no annotations",
			content: `pub fn run() {}`,
			tag:     "@doc",
			want:    nil,
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
