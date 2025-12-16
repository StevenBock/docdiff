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

### Core Flow

1. **Scanner** (`internal/scanner`) walks the codebase, uses Detector to identify languages, extracts annotations via Strategy
2. **Metadata** (`internal/metadata`) manages `docs/.doc-versions.json` storing commit hashes per doc
3. **Git** (`internal/git`) wraps git commands to detect changes between commits
4. **Report** (`internal/report`) formats output (human, JSON, SARIF)
5. **Commands** (`internal/commands`) wires everything together via Cobra CLI

### Commands

- `init` - Create metadata file with current HEAD hashes
- `report` - Show stale/orphaned docs (supports `--json`, `--sarif`, `--ci`)
- `changes <doc>` - Show code changes since doc updated (`--ai` for LLM-friendly output)
- `sync [doc]` - Update metadata after doc review

## Configuration

Reads `.docdiff.yaml` or `.docdiff.json` from project root. Key settings:
- `annotation_tag` - Customizable (default `@doc`)
- `include/exclude` - Glob patterns for file scanning
- `ci.fail_on_stale` - Exit code behavior
