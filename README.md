# docdiff

A language-agnostic CLI tool that detects when documentation becomes stale relative to code changes.

## Overview

docdiff tracks the relationship between source code and documentation files using annotations. When code changes but its associated documentation hasn't been updated, docdiff flags it as stale.

**How it works:**
1. Add `@doc docs/FILE.md` annotations to your source code comments
2. Run `docdiff init` to create a metadata file tracking commit hashes
3. Run `docdiff report` to see which docs are stale after code changes
4. Use `docdiff changes` to see what changed (with AI-friendly output)
5. Run `docdiff sync` after updating documentation

## Installation

```bash
go install github.com/StevenBock/docdiff/cmd/docdiff@latest
```

Or build from source:

```bash
git clone https://github.com/StevenBock/docdiff.git
cd docdiff
go build -o docdiff ./cmd/docdiff
```

## Quick Start

1. Add annotations to your source files:

```go
// @doc docs/API.md
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

```python
# @doc docs/ALGORITHMS.md
def quicksort(arr):
    # ...
```

```php
/**
 * @doc docs/AUTHENTICATION.md
 */
class AuthController {
    // ...
}
```

2. Initialize tracking:

```bash
docdiff init
```

3. Check for stale documentation:

```bash
docdiff report
```

## Commands

### `docdiff init`

Initialize documentation tracking by creating a metadata file.

```bash
docdiff init [--force]
```

| Flag | Description |
|------|-------------|
| `--force` | Overwrite existing metadata file |

### `docdiff report`

Show documentation coverage and staleness report.

```bash
docdiff report [flags]
```

| Flag | Description |
|------|-------------|
| `--stale` | Only show stale docs |
| `--orphaned` | Only show orphaned files (no `@doc` annotation) |
| `--json` | Output as JSON |
| `--sarif` | Output as SARIF (for CI integration) |
| `--ci` | Enable CI mode (exit 1 on stale docs) |

### `docdiff changes`

Show code changes since a doc was last updated.

```bash
docdiff changes <doc> [flags]
```

| Flag | Description |
|------|-------------|
| `--commits` | Show commit list only |
| `--summary` | Output summary format |
| `--ai` | Output format optimized for AI documentation updates |

### `docdiff sync`

Update metadata after reviewing and updating documentation.

```bash
docdiff sync [doc]
```

Without arguments, syncs all docs to current HEAD. With a doc path, syncs only that doc.

## Configuration

Create `.docdiff.yaml` in your project root:

```yaml
annotation_tag: "@doc"
docs_directory: docs
metadata_file: docs/.doc-versions.json

include:
  - "src/**"
  - "app/**"
  - "lib/**"

exclude:
  - "vendor/**"
  - "node_modules/**"
  - "**/*_test.go"
  - "**/*.test.js"

languages:
  php:
    enabled: true
  go:
    enabled: true
  java:
    enabled: true
  python:
    enabled: true
  javascript:
    enabled: true
  ruby:
    enabled: true

ci:
  fail_on_stale: true
  fail_on_orphaned: false
```

Also supports `.docdiff.json`.

## Supported Languages

| Language | Extensions | Comment Styles |
|----------|------------|----------------|
| Go | `.go` | `//`, `/* */` |
| Java | `.java` | `//`, `/* */`, `/** */` |
| JavaScript | `.js`, `.jsx`, `.ts`, `.tsx`, `.mjs`, `.cjs` | `//`, `/* */` |
| PHP | `.php` | `//`, `#`, `/* */`, `/** */` |
| Python | `.py` | `#`, `"""`, `'''` |
| Ruby | `.rb`, `.rake` | `#`, `=begin/=end` |

### Intelligent File Type Detection

docdiff uses a priority cascade to detect file types:

1. **Shebang** - `#!/usr/bin/env python`, `#!/usr/bin/node`
2. **Editor modelines** - `# vim: ft=ruby`, `# -*- mode: python -*-`
3. **File extension** - Standard extension mapping
4. **Content heuristics** - `<?php`, `package main`, etc.

This means extensionless scripts are handled correctly.

## AI-Friendly Output

The `--ai` flag produces structured output perfect for feeding to AI assistants:

```bash
docdiff changes docs/API.md --ai | claude
```

Output includes:
- Current documentation content
- List of tracked source files
- All commits since last sync
- Full diffs grouped by commit
- Instructions for the AI

## CI Integration

### GitHub Actions

```yaml
- name: Check documentation freshness
  run: docdiff report --ci
```

### Exit Codes

- `0` - Success, no issues
- `1` - Stale docs found (CI mode) or error

### SARIF Output

For GitHub Code Scanning:

```yaml
- name: Check docs
  run: docdiff report --sarif > docdiff.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: docdiff.sarif
```

## Adding New Languages

Implement the `Strategy` interface:

```go
type Strategy interface {
    Name() string
    Extensions() []string
    CommentPatterns() []*regexp.Regexp
    ExtractAnnotations(content []byte, tag string) []string
}
```

Then register in `internal/language/registry.go`:

```go
func DefaultRegistry() *Registry {
    r := NewRegistry()
    r.Register(NewGoStrategy())
    r.Register(NewYourLanguageStrategy()) // Add here
    return r
}
```

## Example Workflow

```bash
# Initial setup
docdiff init

# After making code changes, check what's stale
docdiff report

# See what changed for a specific doc
docdiff changes docs/API.md

# Or get AI-friendly output
docdiff changes docs/API.md --ai | pbcopy
# Paste into your AI assistant

# After updating the documentation
docdiff sync docs/API.md

# Verify everything is current
docdiff report
```

## License

MIT
