package language

import (
	"testing"
)

func TestJavaStrategy(t *testing.T) {
	strategy := NewJavaStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "java" {
			t.Errorf("Name() = %v, want java", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		if len(exts) != 1 || exts[0] != ".java" {
			t.Errorf("Extensions() = %v, want [.java]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "javadoc annotation",
			content: `package com.example;

/**
 * Service for handling API requests
 * @doc docs/API.md
 */
public class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "multiple javadoc annotations",
			content: `package com.example;

/**
 * @doc docs/API.md
 * @doc docs/SERVICES.md
 */
public class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/SERVICES.md"},
		},
		{
			name: "single line comment",
			content: `package com.example;

// @doc docs/API.md
public class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "block comment",
			content: `package com.example;

/* @doc docs/API.md */
public class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name:    "no annotations",
			content: `package com.example; public class ApiService {}`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "mixed comment styles",
			content: `package com.example;

/**
 * @doc docs/JAVADOC.md
 */
// @doc docs/INLINE.md
/* @doc docs/BLOCK.md */
public class ApiService {}`,
			tag:  "@doc",
			want: []string{"docs/JAVADOC.md", "docs/INLINE.md", "docs/BLOCK.md"},
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
