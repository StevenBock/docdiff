package language

import (
	"testing"
)

func TestPowerShellStrategy(t *testing.T) {
	strategy := NewPowerShellStrategy()

	t.Run("name", func(t *testing.T) {
		if got := strategy.Name(); got != "powershell" {
			t.Errorf("Name() = %v, want powershell", got)
		}
	})

	t.Run("extensions", func(t *testing.T) {
		exts := strategy.Extensions()
		expected := map[string]bool{".ps1": true, ".psm1": true}
		for _, ext := range exts {
			if !expected[ext] {
				t.Errorf("Unexpected extension: %v", ext)
			}
		}
		if len(exts) != 2 {
			t.Errorf("Extensions() = %v, want [.ps1, .psm1]", exts)
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
Write-Host "hello"`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "block comment",
			content: `<#
Module for API handling
@doc docs/API.md
#>
function Get-Data {
    Write-Output "data"
}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "multiple hash comments",
			content: `# @doc docs/API.md
# @doc docs/GUIDE.md
param([string]$Name)`,
			tag:  "@doc",
			want: []string{"docs/API.md", "docs/GUIDE.md"},
		},
		{
			name: "inline comment",
			content: `if (Test-Path $file) {  # @doc docs/API.md
    Get-Content $file
}`,
			tag:  "@doc",
			want: []string{"docs/API.md"},
		},
		{
			name: "block comment with multiple annotations",
			content: `<#
.SYNOPSIS
Main module
@doc docs/README.md
@doc docs/INSTALL.md
#>
function Main {
    Write-Host "Starting"
}`,
			tag:  "@doc",
			want: []string{"docs/README.md", "docs/INSTALL.md"},
		},
		{
			name: "mixed comments",
			content: `# @doc docs/CONFIG.md
<#
Additional configuration
@doc docs/ADVANCED.md
#>
param([string]$Config)`,
			tag:  "@doc",
			want: []string{"docs/CONFIG.md", "docs/ADVANCED.md"},
		},
		{
			name:    "no annotations",
			content: `Write-Host "hello world"`,
			tag:     "@doc",
			want:    nil,
		},
		{
			name: "param block with annotation",
			content: `# @doc docs/PARAMS.md
param(
    [Parameter(Mandatory=$true)]
    [string]$Name
)`,
			tag:  "@doc",
			want: []string{"docs/PARAMS.md"},
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
