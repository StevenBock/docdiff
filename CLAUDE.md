# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

docdiff is a language-agnostic CLI tool that detects stale documentation by tracking `@doc` annotations in source code comments. When code files change but their linked documentation hasn't been updated, docdiff flags them as stale.

## Build & Test Commands

```bash
# Build
go build -o docdiff ./cmd/docdiff

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/language/...

# Run a single test
go test ./internal/language -run TestPHPStrategy

# Run tests with verbose output
go test -v ./...
```

## Architecture

### Strategy Pattern for Language Support

The `internal/language` package uses a strategy pattern for extensibility:

- `Strategy` interface defines how to extract `@doc` annotations from source files
- `BaseStrategy` provides shared implementation for comment pattern matching
- Each language (Go, Java, PHP, Python, JavaScript, Ruby) implements `Strategy`
- `Registry` maps file extensions to strategies and allows runtime registration

**To add a new language:** Implement `Strategy` interface in a new file, then register it in `DefaultRegistry()` in `registry.go`.

### File Type Detection

`internal/filetype/Detector` uses a priority cascade to identify file types:
1. Shebang (`#!/usr/bin/env python`)
2. Editor modelines (`vim: ft=ruby`, `-*- mode: python -*-`)
3. File extension
4. Content heuristics (`<?php`, `package main`, etc.)

### Staleness model

There is **no metadata file**. A doc's "last reviewed" anchor is its own last
commit (`git log -1 -- <doc>`). A doc is stale when a linked source file has a
commit newer than the doc's last commit. Editing code and its doc in the **same
commit** makes them share that anchor, so nothing is stale — one commit per unit
of work, no separate sync step. `computeStaleDocs` (`internal/commands/staleness.go`)
implements this and is shared by `report`, `check`, and `graph`.

For the rare "code changed but the doc needs no edit" case, `ack` records a floor
commit per doc in `.docdiff-acks.json` (repo root, committed); `effectiveBaseline`
takes the newer of the doc's last commit and its floor. A missing/garbage-collected
floor falls back to the doc's own commit, so it never hides real changes.

### Core Flow

1. **Scanner** (`internal/scanner`) walks the codebase, uses Detector to identify languages, extracts annotations via Strategy
2. **Git** (`internal/git`) wraps git commands; `LastCommit(path)` is the review anchor, plus range diffs between commits
3. **Report** (`internal/report`) formats output (human, JSON, SARIF)
4. **Commands** (`internal/commands`) wires everything together via Cobra CLI

### Commands

- `check` - Show only docs affected by the current working tree / staged / `--files` set; exits non-zero when an affected doc needs updating (`--json`). Also surfaces missing back-links (undocumented refs) for affected docs. The agent-focused command.
- `report` - Show repo-wide stale/orphaned docs (supports `--json`, `--sarif`, `--ci`)
- `changes <doc>` - Show code changes since the doc's last commit (`--ai`, `--working-tree`, `--staged`, `--hide-annotations` to drop annotation-only diff hunks)
- `ack <doc>...` - Record a review floor for a doc whose code changed but text needed no edit (`--to <ref>`; writes `.docdiff-acks.json`)
- `suggest` - Group orphaned files by likely owning doc (directory-vote heuristic) and emit `@doc` annotation lines in batches (`--json`)
- `graph` - Output doc-to-file relationship graph (DOT or `--mermaid`); stale links highlighted

## Configuration

Reads `.docdiff.yaml` or `.docdiff.json` from project root. Key settings:
- `annotation_tag` - Customizable (default `@doc`)
- `include/exclude` - Glob patterns for file scanning. A pattern without a `/` matches the basename at any depth (gitignore-like).
- `respect_gitignore` - Skip files git ignores via `git check-ignore` (default `true`)
- `ci.fail_on_stale` - Exit code behavior

A `.docdiffignore` file (one glob per line, `#` comments) adds extra excludes on top of `exclude:` — use it for committed files git won't ignore (vendored license text, local notes).
