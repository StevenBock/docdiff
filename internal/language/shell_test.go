package language

import (
	"testing"
)

func TestShellStrategy(t *testing.T) {
	strategy := NewShellStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "shell" {
			t.Errorf("Name() = %v, want shell", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		expected := map[string]bool{".sh": true, ".bash": true}
		for _, ext := range exts {
			if !expected[ext] {
				t.Errorf("Unexpected extension: %v", ext)
			}
		}
		if len(exts) != 2 {
			t.Errorf("Extensions() = %v, want [.sh, .bash]", exts)
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
			content: `#!/bin/bash
# @doc docs/INSTALL.md
echo "hello"`,
			tag:  "@doc",
			want: []string{"docs/INSTALL.md"},
		},
		{
			name: "multiple hash comments",
			content: `# @doc docs/INSTALL.md
# @doc docs/README.md
set -e`,
			tag:  "@doc",
			want: []string{"docs/INSTALL.md", "docs/README.md"},
		},
		{
			name: "inline comment",
			content: `if [ -f /etc/passwd ]; then  # @doc docs/API.md
  cat /etc/passwd
fi`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "shebang with annotation",
			content: `#!/usr/bin/env bash
# @doc docs/DEPLOY.md
function deploy() {
  echo "Deploying..."
}`,
			tag:  "@doc",
			want: []string{"docs/DEPLOY.md"},
		},
		{
			name:    "no annotations",
			content: `#!/bin/bash
echo "hello world"`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "case statement with annotation",
			content: `# @doc docs/CLI.md
case $1 in
  start) echo "Starting" ;;
  stop) echo "Stopping" ;;
esac`,
			tag:  "@doc",
			want: []string{"docs/CLI.md"},
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
