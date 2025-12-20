package language

import (
	"testing"
)

func TestVueStrategy(t *testing.T) {
	strategy := NewVueStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "vue" {
			t.Errorf("Name() = %v, want vue", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		if len(exts) != 1 || exts[0] != ".vue" {
			t.Errorf("Extensions() = %v, want [.vue]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "html comment in template",
			content: `<template>
  <!-- @doc docs/COMPONENT.md -->
  <div>Hello</div>
</template>`,
			tag:  "@doc",
			want: []string{"docs/COMPONENT.md"},
		},
		{
			name: "single line comment in script",
			content: `<script>
// @doc docs/API.md
export default {
  name: 'MyComponent'
}
</script>`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "block comment in script",
			content: `<script>
/* @doc docs/API.md */
export default {}
</script>`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "jsdoc in script",
			content: `<script>
/**
 * @doc docs/API.md
 */
export default {}
</script>`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "script setup with TypeScript",
			content: `<script setup lang="ts">
// @doc docs/COMPOSABLES.md
import { ref } from 'vue'
const count = ref(0)
</script>`,
			tag:  "@doc",
			want: []string{"docs/COMPOSABLES.md"},
		},
		{
			name: "css comment in style",
			content: `<style>
/* @doc docs/STYLES.md */
.container { color: red; }
</style>`,
			tag:  "@doc",
			want: []string{"docs/STYLES.md"},
		},
		{
			name: "multiple annotations across sections",
			content: `<template>
  <!-- @doc docs/TEMPLATE.md -->
  <div>Hello</div>
</template>

<script>
// @doc docs/SCRIPT.md
export default {}
</script>

<style>
/* @doc docs/STYLES.md */
</style>`,
			tag:  "@doc",
			want: []string{"docs/TEMPLATE.md", "docs/SCRIPT.md", "docs/STYLES.md"},
		},
		{
			name:    "no annotations",
			content: `<template><div>Hello</div></template>`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "multiline html comment",
			content: `<template>
  <!--
    @doc docs/COMPONENT.md
  -->
  <div>Hello</div>
</template>`,
			tag:  "@doc",
			want: []string{"docs/COMPONENT.md"},
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
