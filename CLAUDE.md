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

**Annotation scopes.** `@doc <path>` links a whole file to a doc. `@doc <path> #<scope>` (a `#`-prefixed suffix, e.g. `@doc docs/CORE.md #settings.general`) narrows ownership: the scoped annotation owns the code region from its line down to the next annotation, so a change elsewhere in a central/mirror file doesn't flag it. `@doc -` is a region terminator: it closes the current scoped region without assigning the code below it to any doc (`language.Terminator`; kept in `Details` as a boundary, excluded from `FilesByDoc`). `check` matches changed diff hunks against these regions (diff-backed modes only — `--files` and untracked/new files fall back to whole-file). Extraction lives in `extractDetailed`/`ExtractDetailed` on `BaseStrategy` (inherited by every language for free); the region + hunk-overlap logic is in `internal/commands/scope.go`.

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

Diff hunks whose every changed line contains the annotation tag are
**annotation-only** (ownership-map churn, not code change). Classification
happens per hunk in `parseHunks` (`internal/git/git.go`, `LineRange.AnnotationOnly`)
and per file in `annotationOnlyDiffFiles` (`internal/commands/difffilter.go`).
`check` moves docs whose only trigger was annotation churn into a non-gating
"Ownership map changed" section (JSON: `ownership_changed`), and
`computeStaleDocs` drops fully-annotation-only files before declaring a doc
stale, so `report`/`graph` stay consistent with `check`.

For the rare "code changed but the doc needs no edit" case, `ack` records a floor
commit per doc in `.docdiff-acks.json` (repo root, committed); `effectiveBaseline`
takes the newer of the doc's last commit and its floor. A missing/garbage-collected
floor falls back to the doc's own commit, so it never hides real changes. To keep
the ack in the same commit as the code it reviews, `ack --amend` folds
`.docdiff-acks.json` into HEAD. Because a commit cannot contain its own hash in
the ack file, amended-away floors are re-anchored from the ack entry's committed
history when staleness is computed.

### Core Flow

1. **Scanner** (`internal/scanner`) walks the codebase, uses Detector to identify languages, extracts annotations via Strategy
2. **Git** (`internal/git`) wraps git commands; `LastCommit(path)` is the review anchor, plus range diffs between commits
3. **Report** (`internal/report`) formats output (human, JSON, SARIF)
4. **Commands** (`internal/commands`) wires everything together via Cobra CLI

### Commands

- `check` - Show only docs affected by the current working tree / staged / `--files` set; exits non-zero when an affected doc needs updating (`--json`). Output is split into actionability sections — **Required** (the only one that gates the exit code), **Already updated**, **Ownership map changed** (annotation-only churn, non-gating), and **Back-link hygiene** (`--no-backlinks` to hide it). For diff-backed modes it matches changed hunks against annotation scopes, so a change to one part of a central file only flags the docs that own that part; each scoped Required entry shows which lines hit the scope plus a short excerpt, so update-vs-ack is decidable from the output alone. The agent-focused command.
- `explain <doc>` - One-shot staleness reasoning for a single doc: linked files, review anchor, ack floor, effective baseline, newest linked commit, and whether the working tree contributes — instead of running several `changes` invocations.
- `report` - Show repo-wide stale/orphaned docs (supports `--json`, `--sarif`, `--ci`, `--no-backlinks`)
- `changes <doc>` - Show code changes since the doc's last commit (`--ai`, `--working-tree`, `--staged`, `--hide-annotations` to drop annotation-only diff hunks)
- `ack <doc>...` - Record a review floor for a doc whose code changed but text needed no edit (`--to <ref>`; writes `.docdiff-acks.json`). `--amend` folds the floor into the current HEAD commit so code and its ack live in one commit (per the commit-together rule). `ack --required` replays check's Required computation over HEAD's own diff (`ShowCommitHunks`, scope matching and annotation-only filtering included), intersects it with currently-stale docs, and acks that set in one command; composes with `--amend`.
- `lint` - Annotation-map hygiene, independent of staleness (`--json`). Problems (non-zero exit): scope-to-eof (a file's only annotation is scoped, so its region runs to EOF — fix with `@doc -`), mixed-ownership (same doc linked whole-file + scoped in one file), fan-out (≥ `lint.fan_out_threshold` distinct whole-file links on one file, default 3). Advisory (never gates): vestigial-link (a doc read as content never mentions a file linking to it). Doc reference sets come from the scanner (`Result.DocRefs`), which reads every markdown doc under `docs_directory` PLUS every linked doc elsewhere (CLAUDE.md, README.md, ...).
- `suggest` - Group orphaned files by likely owning doc (directory-vote heuristic) and emit `@doc` annotation lines in batches (`--json`)
- `graph` - Output doc-to-file relationship graph (DOT or `--mermaid`); stale links highlighted

## Configuration

Reads `.docdiff.yaml` or `.docdiff.json` from project root. Key settings:
- `annotation_tag` - Customizable (default `@doc`)
- `include/exclude` - Glob patterns for file scanning. A pattern without a `/` matches the basename at any depth (gitignore-like).
- `respect_gitignore` - Skip files git ignores via `git check-ignore` (default `true`)
- `ci.fail_on_stale` - Exit code behavior
- `lint.fan_out_threshold` - Whole-file links on one file before `lint` flags fan-out (default `3`)
- `docs.<path>.gate: structural` - Inverts flagging polarity for that doc: annotation-map changes and linked-file add/delete/rename gate (`check`) and stale (`report`); body edits do not. Pure annotation moves within a file (equal per-doc multisets of scopes over annotation-only hunks) never gate; an annotation changed inside a code hunk gates conservatively. Shared logic in `internal/commands/structural.go` (`annotationDeltasByFile`, `structuralTriggers`); applied in `check`'s structural pass and `computeStaleDocs` (whole-repo diff + `--name-status` per baseline, cached). Diff-backed modes only — `--files` mode keeps normal behavior for these docs.

A `.docdiffignore` file (one glob per line, `#` comments) adds extra excludes on top of `exclude:` — use it for committed files git won't ignore (vendored license text, local notes).
