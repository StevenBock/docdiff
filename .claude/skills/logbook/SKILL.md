---
name: Logbook
description: 'Use and maintain the project Logbook — the agent-owned wiki that carries working state between sessions: read it before resuming ongoing work, log what moved before you finish, and keep the home map of content current. Use when starting work on a long-running effort, when asked "where did we leave off", after finishing a work session, or when a synthesis is worth keeping.'
---
<!-- managed-by: hangar-agent-skills slug=logbook -->

# Logbook

The Logbook is this project's memory between sessions, and you own it. It is a per-project wiki of durable **pages** plus an append-only **log**, maintained entirely through the `hangar_logbook_*` MCP tools and read by every agent that launches here — a digest of it arrived in your first message. A fresh session should resume a long-running effort from a page, not from re-exploring the codebase or making the human re-explain.

It holds what the code cannot say — current state of ongoing efforts, what was tried and abandoned and why, syntheses worth finding again, and project facts that conversation keeps re-establishing — and code orientation is welcome too: write those pages knowing the code moves, and read them trusting the timestamps. Where the project keeps repo docs that actually match the code (`docs/`, README, CLAUDE.md), link to them rather than copying — but deference is earned by accuracy, not by a docs folder existing. When a repo doc has drifted from the code, record the correction here instead of propagating its claim.

## The shape

- **Pages** are slug-addressed markdown, whole-field writes. Kinds: `saga` (a long-running effort: current state, attempts, open issues), `synthesis` (an audit, review, or investigation result), `reference` (stable facts), `map` (a navigation/lookup hub of pointers and relationships across pages or subsystems, not one subsystem's internals), `playbook` (a prescriptive procedure/runbook whose phases or steps are executed in order). Statuses: `active` · `settled` · `archived` — only active pages reach the launch digest. Each page carries a one-line `summary` (the index/digest row) plus a short `description` — a couple of sentences on what the page is for and when to read it — so a listing explains itself without opening every page.
- **`home`** is the map of content — a curated, categorized index of `[[slug]]` links, auto-created on first write. Keep it a map, not content. It leads every agent's launch digest, so its quality sets how well the next session orients.
- **Entries** are the append-only log: short, timestamped paragraphs, optionally anchored to a page. They are never edited or deleted.
- **`[[slug]]`** links (or `[[slug|label]]`) are first-class: extracted into the knowledge graph on every write. A dangling link is not an error — it marks a page worth writing. Link another project's page with `[[project-name/slug]]` — the project name is matched against your project list, and the link resolves live (so a project rename can dangle it).

## Reading — orient before you dig

1. The launch digest gives you home's summary, the active pages, and recent log entries. Start there.
2. `hangar_logbook_index` re-fetches that view; `hangar_logbook_graph` returns the whole knowledge graph — pages, links, unresolved targets, and orphans — when you need to see what exists, what's missing, or what's unconnected.
3. Reach for the surgical path, not the firehose: `hangar_logbook_get_page` for one page in full (its backlinks tell you what context points here); `hangar_logbook_search` (by query, file, module, or category) to find the entries and pages that match your intent; `hangar_logbook_get_pages` with `bodies: "summaries"` for a quick multi-page survey (metadata only — it defaults to summaries past three slugs). Pull full bodies only when you genuinely need them — ask for three slugs or fewer, or pass `bodies: "full"`.

## Writing back — close the loop

Before finishing a work session on anything long-running: `hangar_logbook_log` what moved, in a paragraph. Tag the files your session touched via the log tool's `files` argument so future agents working on those files find your entry. Tag the modules ([[slug]] namespace) and commits your entry concerns, and mark decisions and gotchas with `category` so they stay findable by intent. If the page's summary is no longer true, update the page (`hangar_logbook_upsert_page`). If you settled or killed an effort, set its status accordingly — a stale `active` saga misleads every future session.

Create a **new** page only when knowledge was re-derived or re-explained more than once, or a synthesis is worth finding again. Otherwise prefer a log entry — pages are for knowledge with a future, the log is for what happened. When you do create one, link it from `home` (and from related pages), or it becomes an orphan nobody finds.

## Guardrails

- Trust repo docs as far as they match the code, no further. Accurate, maintained docs are worth linking instead of copying — they live in git and stay consistent with each branch. A stale or junk docs folder earns no deference: verify against the code, log the drift, and let the Logbook carry the truth. Treat a code page here as exactly as fresh as its updated_at.
- Anchor code references by symbol and file (`resolveThing` in `src/things.ts`), never by line number — symbols survive edits, line numbers rot on the next commit.
- Whole-field page writes return a content hash — verify round trips against it instead of re-reading large bodies.
- Read/write ratio is tracked and shown in the UI. A logbook that is only written to is a failed logbook — reading it is part of the job.
- Pages untouched for 30 days get flagged possibly-stale in the digest. Either the effort moved and the page should say so, or it ended and the status should.
