package language

import (
	"testing"
)

func TestPHPStrategy(t *testing.T) {
	strategy := NewPHPStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "php" {
			t.Errorf("Name() = %v, want php", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		if len(exts) != 1 || exts[0] != ".php" {
			t.Errorf("Extensions() = %v, want [.php]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "docblock annotation",
			content: `<?php
/**
 * Service for handling API requests
 * @doc docs/API.md
 */
class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "multiple docblock annotations",
			content: `<?php
/**
 * @doc docs/API.md
 * @doc docs/SERVICES.md
 */
class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/SERVICES.md"},
		},
		{
			name: "single line comment",
			content: `<?php
// @doc docs/API.md
class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "hash comment",
			content: `<?php
# @doc docs/API.md
class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name:    "no annotations",
			content: `<?php class ApiService {}`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "annotation in string (matches - known limitation)",
			content: `<?php
$str = "// @doc docs/FAKE.md";
// @doc docs/REAL.md`,
			tag:  "@doc",
			want: []string{"docs/FAKE.md\";", "docs/REAL.md"},
		},
		{
			name: "custom tag",
			content: `<?php
/**
 * @docs docs/API.md
 */
class ApiService {}`,
			tag:  "@docs",
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
