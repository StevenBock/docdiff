package language

import (
	"testing"
)

func TestRubyStrategy(t *testing.T) {
	strategy := NewRubyStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "ruby" {
			t.Errorf("Name() = %v, want ruby", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		expected := map[string]bool{".rb": true, ".rake": true}
		for _, ext := range exts {
			if !expected[ext] {
				t.Errorf("Unexpected extension: %v", ext)
			}
		}
		if len(exts) != 2 {
			t.Errorf("Extensions() = %v, want [.rb, .rake]", exts)
		}
	})

	tests := []struct {
		name    string
		content string
		tag     string
		want    []string
	}{
		{
			name: "hash comment",
			content: `# @doc docs/API.md
class ApiService
end`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "multiple hash comments",
			content: `# @doc docs/API.md
# @doc docs/SERVICES.md
class ApiService
end`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/SERVICES.md"},
		},
		{
			name: "block comment",
			content: `=begin
@doc docs/API.md
=end
class ApiService
end`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "inline comment",
			content: `class ApiService  # @doc docs/API.md
end`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name:    "no annotations",
			content: `class ApiService; end`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "rake file",
			content: `# @doc docs/TASKS.md
task :default do
  puts "Hello"
end`,
			tag:  "@doc",
			want: []string{"docs/TASKS.md"},
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
