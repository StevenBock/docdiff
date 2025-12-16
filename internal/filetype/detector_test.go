package filetype

import (
	"testing"

	"github.com/StevenBock/docdiff/internal/language"
)

func TestDetector_DetectShebang(t *testing.T) {
	registry := language.DefaultRegistry()
	detector := NewDetector(registry)

	tests := []struct {
		name     string
		content  string
		wantLang string
		wantOk   bool
	}{
		{
			name:     "python shebang",
			content:  "#!/usr/bin/env python\nprint('hello')",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "python3 shebang",
			content:  "#!/usr/bin/env python3\nprint('hello')",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "ruby shebang",
			content:  "#!/usr/bin/env ruby\nputs 'hello'",
			wantLang: "ruby",
			wantOk:   true,
		},
		{
			name:     "node shebang",
			content:  "#!/usr/bin/env node\nconsole.log('hello')",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "ts-node shebang",
			content:  "#!/usr/bin/env ts-node\nconsole.log('hello')",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "php shebang",
			content:  "#!/usr/bin/env php\n<?php echo 'hello';",
			wantLang: "php",
			wantOk:   true,
		},
		{
			name:     "direct path shebang",
			content:  "#!/usr/bin/python3\nprint('hello')",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "no shebang",
			content:  "print('hello')",
			wantLang: "",
			wantOk:   false,
		},
		{
			name:     "bash shebang (not supported)",
			content:  "#!/bin/bash\necho hello",
			wantLang: "",
			wantOk:   false,
		},
		{
			name:     "empty content",
			content:  "",
			wantLang: "",
			wantOk:   false,
		},
		{
			name:     "shebang with spaces",
			content:  "#! /usr/bin/env python\nprint('hello')",
			wantLang: "python",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, ok := detector.Detect("script", []byte(tt.content))

			if ok != tt.wantOk {
				t.Errorf("Detect() ok = %v, want %v", ok, tt.wantOk)
				return
			}

			if tt.wantOk {
				if strategy == nil {
					t.Error("Detect() returned nil strategy")
					return
				}
				if strategy.Name() != tt.wantLang {
					t.Errorf("Detect() lang = %s, want %s", strategy.Name(), tt.wantLang)
				}
			}
		})
	}
}

func TestDetector_DetectModeline(t *testing.T) {
	registry := language.DefaultRegistry()
	detector := NewDetector(registry)

	tests := []struct {
		name     string
		content  string
		wantLang string
		wantOk   bool
	}{
		{
			name:     "vim filetype",
			content:  "# vim: set ft=python :\nsome content",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "vim filetype full",
			content:  "# vim: set filetype=ruby :\nsome content",
			wantLang: "ruby",
			wantOk:   true,
		},
		{
			name:     "emacs mode",
			content:  "# -*- mode: python -*-\nsome content",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "emacs mode short",
			content:  "# -*- python -*-\nsome content",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "vim at end of file",
			content:  "some content\n# vim: ft=go",
			wantLang: "go",
			wantOk:   true,
		},
		{
			name:     "typescript via vim",
			content:  "// vim: ft=typescript\nconst x = 1;",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "no modeline",
			content:  "just some content",
			wantLang: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, ok := detector.Detect("file", []byte(tt.content))

			if ok != tt.wantOk {
				t.Errorf("Detect() ok = %v, want %v", ok, tt.wantOk)
				return
			}

			if tt.wantOk {
				if strategy == nil {
					t.Error("Detect() returned nil strategy")
					return
				}
				if strategy.Name() != tt.wantLang {
					t.Errorf("Detect() lang = %s, want %s", strategy.Name(), tt.wantLang)
				}
			}
		})
	}
}

func TestDetector_DetectExtension(t *testing.T) {
	registry := language.DefaultRegistry()
	detector := NewDetector(registry)

	tests := []struct {
		name     string
		path     string
		wantLang string
		wantOk   bool
	}{
		{
			name:     "php file",
			path:     "/path/to/file.php",
			wantLang: "php",
			wantOk:   true,
		},
		{
			name:     "go file",
			path:     "/path/to/file.go",
			wantLang: "go",
			wantOk:   true,
		},
		{
			name:     "java file",
			path:     "/path/to/File.java",
			wantLang: "java",
			wantOk:   true,
		},
		{
			name:     "python file",
			path:     "/path/to/file.py",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "javascript file",
			path:     "/path/to/file.js",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "typescript file",
			path:     "/path/to/file.ts",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "tsx file",
			path:     "/path/to/Component.tsx",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "ruby file",
			path:     "/path/to/file.rb",
			wantLang: "ruby",
			wantOk:   true,
		},
		{
			name:     "rake file",
			path:     "/path/to/Rakefile.rake",
			wantLang: "ruby",
			wantOk:   true,
		},
		{
			name:     "unknown extension",
			path:     "/path/to/file.xyz",
			wantLang: "",
			wantOk:   false,
		},
		{
			name:     "no extension",
			path:     "/path/to/Makefile",
			wantLang: "",
			wantOk:   false,
		},
		{
			name:     "uppercase extension (case-insensitive)",
			path:     "/path/to/file.PHP",
			wantLang: "php",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, ok := detector.Detect(tt.path, []byte{})

			if ok != tt.wantOk {
				t.Errorf("Detect() ok = %v, want %v", ok, tt.wantOk)
				return
			}

			if tt.wantOk {
				if strategy == nil {
					t.Error("Detect() returned nil strategy")
					return
				}
				if strategy.Name() != tt.wantLang {
					t.Errorf("Detect() lang = %s, want %s", strategy.Name(), tt.wantLang)
				}
			}
		})
	}
}

func TestDetector_DetectContent(t *testing.T) {
	registry := language.DefaultRegistry()
	detector := NewDetector(registry)

	tests := []struct {
		name     string
		content  string
		wantLang string
		wantOk   bool
	}{
		{
			name:     "php opening tag",
			content:  "<?php\necho 'hello';",
			wantLang: "php",
			wantOk:   true,
		},
		{
			name:     "go package",
			content:  "package main\n\nfunc main() {}",
			wantLang: "go",
			wantOk:   true,
		},
		{
			name:     "java package",
			content:  "package com.example;\n\npublic class Main {}",
			wantLang: "java",
			wantOk:   true,
		},
		{
			name:     "java import",
			content:  "import java.util.List;\n\npublic class Main {}",
			wantLang: "java",
			wantOk:   true,
		},
		{
			name:     "python import",
			content:  "from os import path\n\ndef main(): pass",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "python def",
			content:  "def main():\n    pass",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "python class",
			content:  "class MyClass:\n    pass",
			wantLang: "python",
			wantOk:   true,
		},
		{
			name:     "ruby require",
			content:  "require 'json'\n\nputs 'hello'",
			wantLang: "ruby",
			wantOk:   true,
		},
		{
			name:     "ruby module",
			content:  "module MyModule\nend",
			wantLang: "ruby",
			wantOk:   true,
		},
		{
			name:     "javascript const",
			content:  "const x = 1;\nfunction foo() {}",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "javascript export",
			content:  "export default function() {}",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "javascript import from",
			content:  "import React from 'react';\n",
			wantLang: "javascript",
			wantOk:   true,
		},
		{
			name:     "no heuristic match",
			content:  "just some random text",
			wantLang: "",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, ok := detector.Detect("file_without_extension", []byte(tt.content))

			if ok != tt.wantOk {
				t.Errorf("Detect() ok = %v, want %v", ok, tt.wantOk)
				return
			}

			if tt.wantOk {
				if strategy == nil {
					t.Error("Detect() returned nil strategy")
					return
				}
				if strategy.Name() != tt.wantLang {
					t.Errorf("Detect() lang = %s, want %s", strategy.Name(), tt.wantLang)
				}
			}
		})
	}
}

func TestDetector_PriorityCascade(t *testing.T) {
	registry := language.DefaultRegistry()
	detector := NewDetector(registry)

	t.Run("shebang overrides extension", func(t *testing.T) {
		content := "#!/usr/bin/env python\nsome code"
		strategy, ok := detector.Detect("script.rb", []byte(content))

		if !ok {
			t.Fatal("Detect() returned false")
		}
		if strategy.Name() != "python" {
			t.Errorf("Shebang should override extension: got %s, want python", strategy.Name())
		}
	})

	t.Run("modeline overrides extension", func(t *testing.T) {
		content := "# vim: ft=ruby\nsome code"
		strategy, ok := detector.Detect("script.py", []byte(content))

		if !ok {
			t.Fatal("Detect() returned false")
		}
		if strategy.Name() != "ruby" {
			t.Errorf("Modeline should override extension: got %s, want ruby", strategy.Name())
		}
	})

	t.Run("extension used when no shebang or modeline", func(t *testing.T) {
		content := "some code without hints"
		strategy, ok := detector.Detect("file.go", []byte(content))

		if !ok {
			t.Fatal("Detect() returned false")
		}
		if strategy.Name() != "go" {
			t.Errorf("Extension should be used: got %s, want go", strategy.Name())
		}
	})

	t.Run("content heuristics as last resort", func(t *testing.T) {
		content := "<?php echo 'hello';"
		strategy, ok := detector.Detect("script_no_ext", []byte(content))

		if !ok {
			t.Fatal("Detect() returned false")
		}
		if strategy.Name() != "php" {
			t.Errorf("Content heuristics should be used: got %s, want php", strategy.Name())
		}
	})
}
