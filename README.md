# docdiff

A language-agnostic CLI tool that detects when documentation becomes stale relative to code changes.

## Overview

docdiff tracks the relationship between source code and documentation files using annotations. When code changes but its associated documentation hasn't been updated, docdiff flags it as stale.

**How it works:**
1. Add `@doc docs/FILE.md` annotations to your source code comments
2. Run `docdiff report` (or `docdiff check` for just your changes) to see which docs are stale
3. Use `docdiff changes` to see what changed (with AI-friendly output)
4. Commit the code and its doc together — a doc is "reviewed" as of its own last commit, so the shared commit marks it fresh. No metadata file, no separate sync step.

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

2. Check for stale documentation:

```bash
docdiff report
```

No setup step is needed — docdiff derives each doc's "last reviewed" point from
its own last commit in git history.

## Commands

### `docdiff check`

Show only the docs affected by your current (uncommitted) changes, ignoring
unrelated stale docs elsewhere. Exits non-zero while an affected doc still needs
updating — the command for focused agent work.

```bash
docdiff check [flags]
```

| Flag | Description |
|------|-------------|
| `--staged` | Only consider staged (index) changes |
| `--files` | Check an explicit list of files instead of git changes |
| `--json` | Output as JSON |
| `--no-backlinks` | Hide missing back-link hygiene suggestions |

Changes that only touch `@doc` annotation lines (moving, adding, or retargeting
links) are ownership-map churn, not code changes: they never gate the exit code.
They're listed in a separate "Ownership map changed" info section (JSON:
`ownership_changed`), and `report` skips them the same way when computing
staleness.

Committing empties the working tree, so `check` alone would go quiet on the docs
your last commit just put on review. Those are listed as **Outstanding from HEAD**
(JSON: `outstanding_from_head`) with the command that clears them — the same set
`ack --required` computes. It never gates the exit code, since it reports on a
commit already made; remaining stale docs unrelated to HEAD stay a hidden count.

### `docdiff report`

Show documentation coverage and staleness report.

Scoped annotations narrow staleness here too, not just in `check`: a doc is only
stale when a commit since its baseline touched lines inside a region that doc
owns. A central file carrying many scoped annotations therefore flags only the
docs whose own regions changed. Files with no hunk data (added, deleted,
renamed) and whole-file links keep the conservative whole-file behaviour.

```bash
docdiff report [flags]
```

| Flag | Description |
|------|-------------|
| `--stale` | Only show stale docs |
| `--orphaned` | Only show orphaned files (no `@doc` annotation) |
| `--undocumented` | Only show docs that reference files without back-links |
| `--json` | Output as JSON |
| `--sarif` | Output as SARIF (for CI integration) |
| `--ci` | Enable CI mode (exit 1 on stale docs) |
| `--no-backlinks` | Hide missing back-link suggestions |

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
| `--working-tree` | Diff against the working tree (include uncommitted changes) |
| `--staged` | Diff against the index (staged changes only) |
| `--hide-annotations` | Hide diff hunks whose only changes are `@doc` annotation lines |

### `docdiff ack`

Mark a doc reviewed when its linked code changed but the doc needed **no** edit.
Normally you mark a doc reviewed by editing it in the same commit as its code;
when there's nothing to edit, `ack` records a floor commit (default HEAD) in
`.docdiff-acks.json` instead. Staleness is measured from the newer of the doc's
own last commit and this floor, so the doc stops reporting stale until its code
changes again. The floor is an existing commit, so there's no chicken-and-egg —
commit the code, run `ack`, and commit `.docdiff-acks.json`. Use `--amend` to
fold that ack into the current HEAD commit.

Use `--reason` to record *why* the doc needed no edit. An ack asserts "I read
this and it is still true"; without a reason a bulk ack is indistinguishable
from a rubber stamp later. `.docdiff-acks.json` accepts both the original
`{doc: sha}` form and `{doc: {commit, reason}}`, and only writes the object form
when a reason is given — existing files are never rewritten.

```bash
docdiff ack <doc>... [--to <ref>] [--amend] [--reason "..."]
docdiff ack --required [--amend] [--reason "..."]
```

| Flag | Description |
|------|-------------|
| `--to <ref>` | Floor commit to ack at (HEAD, branch, or sha); default HEAD |
| `--amend` | Fold `.docdiff-acks.json` into the current HEAD commit |
| `--required` | Ack exactly the docs `check` would have flagged as Required for HEAD's own diff (and that are still stale) — "I reviewed everything my last commit affected" in one command |
| `--reason` | Why the doc needed no edit; recorded next to the floor commit and shown by `docdiff explain` |

### `docdiff suggest`

Group orphaned files (no `@doc`) by their likely owning doc and emit ready-to-paste annotation lines in batches. The owner is inferred by directory: the nearest ancestor directory with annotated files votes for its most common doc.

```bash
docdiff suggest [--json]
```

### `docdiff graph`

Output a graph showing relationships between documentation and source files.

```bash
docdiff graph [flags]
```

| Flag | Description |
|------|-------------|
| `--mermaid` | Output in Mermaid format instead of DOT (GraphViz) |

Stale documentation relationships are highlighted in red.

### `docdiff lint`

Check the health of the annotation map itself, independent of staleness.
Problems exit non-zero: `scope-to-eof-unbounded` (a scoped annotation running to
end of file with a real region above it — see below), a file linking the same
doc both whole-file and scoped (whole-file wins, the scopes are dead), and
fan-out (one file whole-file-linking ≥ N docs; `lint.fan_out_threshold`,
default 3), and scope drift — a doc using section markers (HTML comments of
the form `@scope <name>`) whose markers no longer match the code annotation
scopes targeting it (both directions; docs without markers are never
checked).

Scope-to-EOF is split by whether a boundary actually exists above the annotation.
**`scope-to-eof-unbounded`** gates: something above owns real code, so how far
the scope reaches is a live question. **`scope-covers-file`** is advisory: nothing
above it owns any code, so the scope narrows nothing. That is usually deliberate
— but a scope written at the top and never bounded looks *identical* from the
file's shape, and only reading what the scope names against what the code does
tells them apart. So it stays visible instead of being silenced. (A file whose
**only** annotation is scoped is not reported at all; it owns the whole file
either way.)

Both have the same two remedies, and both clear the finding: put `@doc -` on the
**last line** to mark the region as deliberately ending at EOF, or earlier if the
scope shouldn't own everything below. Terminating is worth preferring even when
the scope legitimately does own the tail — code appended later then lands
*outside* the region and surfaces as orphaned, rather than being silently
absorbed into whatever scope happened to reach the end of the file. That is a
correctness property, not tidiness.

Back-link coverage is tiered by how well a doc actually routes a reader to the
files that link to it. Only **unlinked** gates — the doc names neither the file,
its directory, nor its bare filename, so nothing gets a reader from the doc to
the code. **dir-only** (names the directory, or a sibling file in it) and
**basename-only** (names the filename but not its path) are weaker coverage, not
broken links, and stay advisory: gating on those is what gets the whole check
suppressed.

Docs whose paths are relative to a subdirectory rather than the repo root can
say so:

```yaml
docs:
  docs/tauri/RUST-TAURI-PROCESS-LIFECYCLE.md:
    path_prefix: src-tauri/src/
```

Resolution order is: the literal path first, then the declared prefix, then
**each parent of the prefix** in turn.

That last step is deliberate, and it is why `path_prefix` is a single string
rather than a list. Real docs that declare a root usually carve out an exception
for a sibling subtree. The doc above opens with "Rust paths are relative to
`src-tauri/src/` unless they begin with `crates/`" — two roots in one sentence.
Walking up covers both from one declaration: `commands/mod.rs` resolves under
`src-tauri/src/`, and `crates/hangar-core/src/lib.rs` falls through to
`src-tauri/`. A list would make every such doc restate a prefix it already
implied, and would need maintaining whenever the exception moved.

If you find a doc whose conventions genuinely aren't nested — two unrelated
roots, neither an ancestor of the other — that's the case a list would solve,
and the case to reopen this on.

```bash
docdiff lint [--json]
```

`--json` carries a `revision` field: the commit this binary was built from, read
from Go's VCS build stamps (suffixed `+dirty` when built over uncommitted
changes, empty when built with `-buildvcs=false`). It is not a hand-maintained
version constant — one nobody remembers to bump reads as authoritative while
being wrong. Every finding also carries its `rule`, so a change in the
`problems` count is attributable per rule without needing a version at all.

### `docdiff stats`

Per-doc ack-vs-edit ratios over the whole history: how often a doc's linked
code was acked (reviewed, no edit) versus how often the doc itself was edited.
A doc that is mostly acked tracks code it doesn't describe — a candidate for a
`gate: structural` policy or deletion. Read-only.

```bash
docdiff stats [--json]
```

### `docdiff onboard`

Print comprehensive docdiff usage instructions for AI agents.

```bash
docdiff onboard
```

Outputs usage instructions and a ready-to-paste snippet for agent instruction files (CLAUDE.md, .github/copilot-instructions.md, .cursorrules, .windsurfrules, AGENTS.md). It also includes the bundled docdiff coding-agent skill content and tells agents to ask before installing it; `onboard` itself never writes skill files. Works without any project setup — no config file or git repository needed.

## Configuration

Create `.docdiff.yaml` in your project root:

```yaml
annotation_tag: "@doc"
docs_directory: docs

include:
  - "src/**"
  - "app/**"
  - "lib/**"

# Skip files git ignores (via `git check-ignore`). Default: true.
respect_gitignore: true

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

lint:
  fan_out_threshold: 3

# Pin check's default scope for this repo ("staged" or "working-tree").
# check also prints an info note when the other scope has changes it
# isn't looking at; it never switches scope on its own.
check:
  scope: working-tree

# Per-doc policies. `gate: structural` inverts what flags the doc: annotation
# map changes and linked-file add/delete/rename gate; body edits do NOT. For
# docs that inventory files (module maps, ownership indexes) rather than
# describe behavior. Pure annotation moves within a file never gate.
docs:
  docs/MODULES.md:
    gate: structural
```

Also supports `.docdiff.json`.

For excludes that aren't in `.gitignore` (committed vendored license text, local notes), add a `.docdiffignore` file — one glob per line, `#` for comments. A pattern with no `/` matches the basename at any depth (gitignore-like).

## Supported Languages

| Language | Extensions | Comment Styles |
|----------|------------|----------------|
| Go | `.go` | `//`, `/* */` |
| Rust | `.rs` | `//`, `/* */`, `///`, `//!`, `/** */` |
| Java | `.java` | `//`, `/* */`, `/** */` |
| JavaScript | `.js`, `.jsx`, `.ts`, `.tsx`, `.mjs`, `.cjs` | `//`, `/* */` |
| PHP | `.php` | `//`, `#`, `/* */`, `/** */` |
| PowerShell | `.ps1`, `.psm1`, `.psd1` | `#`, `<# #>` |
| Python | `.py` | `#`, `"""`, `'''` |
| Ruby | `.rb`, `.rake` | `#`, `=begin/=end` |
| Shell | `.sh`, `.bash`, `.zsh`, `.ksh` | `#` |
| Vue | `.vue` | `//`, `/* */`, `<!-- -->` |

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
- All commits since the doc's last commit
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
# After making code changes, check what's stale
docdiff report          # whole repo
docdiff check           # just the docs your current changes touch

# See what changed for a specific doc
docdiff changes docs/API.md

# Or get AI-friendly output
docdiff changes docs/API.md --ai | pbcopy
# Paste into your AI assistant

# Update the doc, then commit it TOGETHER with your code — the shared commit
# marks the doc reviewed. No sync step, no second commit.
git add src/ docs/API.md && git commit -m "Update handler and its docs"

# If the doc needed NO change, ack it instead of editing it
docdiff ack docs/API.md && git add .docdiff-acks.json && git commit -m "Ack API.md"

# Verify everything is current
docdiff report
```

## License

MIT
