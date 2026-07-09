---
name: docdiff
description: Keep documentation fresh with docdiff — see which docs your code changes affect, update or ack them, and commit code and docs together. Use when modifying source files with @doc annotations, before committing, or when asked about stale documentation.
---
<!-- managed-by: hangar-agent-skills slug=docdiff -->

# docdiff — Keeping Docs Fresh

docdiff links source files to docs via `@doc` comments and flags a doc as stale when its linked code changes. The model is git-native: a doc is "reviewed" as of its own last commit — there is no metadata file and nothing to sync. Run `docdiff onboard` for the tool's own workflow reference.

## Core loop

1. `docdiff check --no-backlinks` — lists ONLY the docs your uncommitted changes affect; exit code is non-zero while any still needs updating. Each flagged doc shows which annotation pulled it in (`via file:line scoped @doc #x`) and a `(broad: N linked files)` marker on high-fanout docs — broad docs are often false positives worth confirming before editing.
2. `docdiff changes <doc> --working-tree --hide-annotations` — the code diff since the doc's last review. `docdiff explain <doc>` — why it's stale (baseline, ack floor, verdict).
3. If the change affects what the doc claims, update the doc and commit code + doc TOGETHER — the shared commit marks it fresh.
4. If a flagged doc genuinely needs no edit: after committing, run ONE variadic `docdiff ack <doc1> <doc2> ... --amend` covering all such docs, then verify with a final `docdiff check`. Never ack a doc you haven't read.

## Habits

- Working tree carrying unrelated changes (parallel agents, shared checkouts)? Stage your paths and gate on `docdiff check --staged --no-backlinks` — plain `check` counts the whole tree.
- `--files <paths...>` scopes a check to an explicit file set; `--json` gives machine-readable output.
- `docdiff report` is the repo-wide stale/orphan backlog — reach for it only when you want the full picture, not for routine changes.
- Invoke the CLI directly — docdiff is a standalone binary, not an npm package (`npx docdiff` will not resolve).

## Annotations

- `// @doc docs/X.md` — whole-file ownership; use only when the entire file belongs to that doc.
- `// @doc docs/X.md #some-scope` — owns the region from its line to the next `@doc` annotation. Prefer scoped annotations on high-fanout central files (registries, event maps, app wiring) so one edit doesn't flag a dozen docs.
- New source files: add annotations linking to the owning doc; `docdiff suggest` emits paste-ready proposals grouped by nearby documented modules. Comment styles: `//`, `#`, `/* */`, `/** */`, `<!-- -->`.
- New docs: commit the doc together with its `@doc` annotations — the doc's own first commit becomes its review anchor.
