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

**`Details` is sorted by line, and that is a contract.** Matching runs one comment pattern at a time, so a file mixing comment styles — a `.vue` with `//` in `<script setup>` and `<!-- -->` in `<template>` — comes out of the match loop grouped by style, with a line-2 annotation sitting after a line-900 one. `extractDetailed` sorts before returning. Consumers that ask positional questions (`lint`'s scope-to-eof asks which annotation is *last*) depend on this; `ownedRegion` deliberately does not, scanning for the nearest following line instead of trusting position, so region ownership stays correct even if the invariant is ever broken again.

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

- `check` - Show only docs affected by the current working tree / staged / `--files` set; exits non-zero when an affected doc needs updating (`--json`). Output is split into actionability sections — **Required** (the only one that gates the exit code), **Already updated**, **Ownership map changed** (annotation-only churn, non-gating), **Outstanding from HEAD** (docs the last commit put on review — committing empties the working tree, so without this the post-commit run goes quiet on exactly them; same set as `ack --required`, non-gating, JSON `outstanding_from_head`), and **Back-link hygiene** (`--no-backlinks` to hide it). For diff-backed modes it matches changed hunks against annotation scopes, so a change to one part of a central file only flags the docs that own that part; each scoped Required entry shows which lines hit the scope plus a short excerpt, so update-vs-ack is decidable from the output alone. The agent-focused command.
- `explain <doc>` - One-shot staleness reasoning for a single doc: linked files, review anchor, ack floor, effective baseline, newest linked commit, and whether the working tree contributes — instead of running several `changes` invocations.
- `report` - Show repo-wide stale/orphaned docs (supports `--json`, `--sarif`, `--ci`, `--no-backlinks`)
- `changes <doc>` - Show code changes since the doc's last commit (`--ai`, `--working-tree`, `--staged`, `--hide-annotations` to drop annotation-only diff hunks)
- `ack <doc>...` - Record a review floor for a doc whose code changed but text needed no edit (`--to <ref>`; writes `.docdiff-acks.json`). `--reason` records why the doc needed no edit alongside the floor; `.docdiff-acks.json` accepts both the original bare-sha form and `{commit, reason}`, and only writes the object form when a reason is given, so existing files are never rewritten. `--amend` folds the floor into the current HEAD commit so code and its ack live in one commit (per the commit-together rule). `ack --required` replays check's Required computation over HEAD's own diff (`ShowCommitHunks`, scope matching and annotation-only filtering included), intersects it with currently-stale docs, and acks that set in one command; composes with `--amend`.
- `lint` - Annotation-map hygiene, independent of staleness (`--json`). Problems (non-zero exit): scope-to-eof-unbounded (a scoped annotation running to EOF with a region above it that owns real code — fix with `@doc -` on the last line to mark it deliberate, or earlier to narrow; a file whose only annotation is scoped is never reported), mixed-ownership (same doc linked whole-file + scoped in one file), fan-out (≥ `lint.fan_out_threshold` distinct whole-file links on one file, default 3), scope-drift (a doc using section markers — HTML comments of the form `@scope <name>` — whose markers don't match code annotation scopes, both directions; marker extraction in `scopeMarkers`, retained as `Result.DocScopeMarkers`; docs without markers are never checked, and section-granular staleness is deliberately NOT built). unlinked (a doc names neither the file, its directory, nor its bare filename, yet the file links to it — nothing routes a task from doc to code). Advisory (never gates): scope-covers-file (a scoped annotation whose whole file is its region, because nothing above owns code — `regionOwnsCodeAbove` in `scope.go` decides which of the two rules applies), plus dir-only and basename-only, the weaker back-link tiers. Doc reference sets come from the scanner (`Result.DocRefs`), which reads every markdown doc under `docs_directory` PLUS every linked doc elsewhere (CLAUDE.md, README.md, ...); the weaker tiers come from `Result.DocWeakRefs`, where a *dir* mention means any ancestor of any path the doc names — naming a sibling file routes a reader to the directory just as well as naming the directory does.

**Doc reference extraction** (`internal/docparse`) has two deliberately asymmetric readings of the same doc. `ParseIncludingCode` (→ `Result.DocRefs`, back-link coverage) counts paths inside fenced code blocks: `npx vitest run src/Foo.test.ts` really does route an agent to the file. `Parse` (→ `Result.UndocumentedRefs`) skips fences: a command example is not a claim that the doc owns that file. Both expand shell-style brace globs (`dir/{a,b}.go`), and both treat a path as a token via `isPathBoundary` rather than a regex context group — a context group consumes its delimiter, so `a/x.go a/y.go` sharing one space hid every second path.

**Matcher changes are monotonic, and that predicts their blast radius.** Widening what the parser recognizes only ever *adds* matches — no existing finding disappears or flips. Every consumer of the reference sets moves in a direction that follows from this, so reason about a matcher change by asking which side of the relation each consumer sits on rather than by measuring one and generalizing:

- **removes** findings where the rule is "the doc doesn't name this file" — `lint`'s unlinked/dir-only/basename-only tiers;
- **adds** findings where the rule is "the doc names a file that doesn't point back" — `report`'s undocumented-refs and `check`'s back-link hygiene.

They are the two directions of one relation. When the fence fix resolved 106 unlinked findings it necessarily also added back-link findings; that is not a regression. The practical consequence: a widening can only ever introduce *new true positives* to a gating config (`ci.fail_on_undocumented_refs`), never change the meaning of an existing one — which is a far smaller risk than "it changes back-link behavior for every repo".

**A structural predicate cannot decide scope ownership — settled by evidence, recorded so it is not retried.** The rule above was going to *silence* the "nothing above owns code" case outright. A predicate was written, pre-committed to a standard, measured over Hangar's 69 findings (47 silence / 22 keep), then checked against independent review of the same files. Three of the 47 were real defects: scopes written at the top of a file and never bounded, which the finding existed to catch and the predicate would have hidden (`lsp-transport.ts` claimed all 316 lines for `#frontend-self-telemetry`; `events.rs` all 86 for `#dev-self-telemetry`; `AccountSyncSettings.test.ts` a Mobile Remote test another doc claims).

The predicate infers "nothing above owns code" ⇒ "the author meant whole-file ownership". That inference is invalid: it is equally consistent with "the author wrote a scope and never bounded it", and those two are structurally identical. Only reading what the scope *names* against what the code *does* separates them. **It was a 94%-accurate prior dressed up as a decision procedure** — which is the sentence to weigh the next plausible structural predicate against, because the next one will fail the same way.

So the observation was kept and the silencing was dropped: the weak case became `scope-covers-file` (advisory, still visible) rather than nothing. Two properties worth preserving if this is ever revisited — a finding that should have been advisory is noise, while one that should have gated and didn't is a defect nobody sees, so unknown-boundary gates; and the three defects were only findable *because* the rule still fired, which is the general argument against shipping a silencer ahead of the review that would validate it.

## Configuration

Reads `.docdiff.yaml` or `.docdiff.json` from project root. Key settings:
- `annotation_tag` - Customizable (default `@doc`)
- `include/exclude` - Glob patterns for file scanning. A pattern without a `/` matches the basename at any depth (gitignore-like).
- `respect_gitignore` - Skip files git ignores via `git check-ignore` (default `true`)
- `ci.fail_on_stale` - Exit code behavior
- `lint.fan_out_threshold` - Whole-file links on one file before `lint` flags fan-out (default `3`)
- `check.scope` - Pin check's default changed-set source per repo (`staged` or `working-tree`); `--staged`/`--files` flags still win. Scope is never inferred from tree state (check-before-staging would narrow to an empty set). Human output notes files the chosen scope leaves out (staged mode) or silently includes (working-tree mode with a dirty index).
- `docs.<path>.path_prefix` - What that doc's relative paths hang off (e.g. `src-tauri/src/`), for docs that don't write repo-root-relative paths. Resolution order: literal path, then the declared prefix, then **each parent of the prefix** (`prefixChain` in `internal/docparse`). The walk-up is why this is a single string and not a list — docs that declare a root normally carve out a nested exception ("relative to `src-tauri/src/` unless they begin with `crates/`"), and one declaration covers both. A list is only warranted if a doc turns up with two roots that aren't nested; don't add one speculatively.
- `docs.<path>.gate: structural` - Inverts flagging polarity for that doc: annotation-map changes and linked-file add/delete/rename gate (`check`) and stale (`report`); body edits do not. Pure annotation moves within a file (equal per-doc multisets of scopes over annotation-only hunks) never gate; an annotation changed inside a code hunk gates conservatively. Shared logic in `internal/commands/structural.go` (`annotationDeltasByFile`, `structuralTriggers`); applied in `check`'s structural pass and `computeStaleDocs` (whole-repo diff + `--name-status` per baseline, cached). Diff-backed modes only — `--files` mode keeps normal behavior for these docs.

A `.docdiffignore` file (one glob per line, `#` comments) adds extra excludes on top of `exclude:` — use it for committed files git won't ignore (vendored license text, local notes).
