---
name: Logbook Sync
description: 'Reconcile the project Logbook with the commits that landed since the last sync: judge what meaningfully changed about ongoing efforts, update pages whose claims went stale, create pages for new functionality worth remembering, and close with a sync marker entry. Use when asked to sync, update, or catch up the logbook, after landing a stretch of work, or on a schedule.'
---
<!-- managed-by: hangar-agent-skills slug=logbook-sync -->

# Logbook Sync

You reconcile the Logbook with what actually happened in the repo since the last sync. You are an editor, not a stenographer: git already stores every commit, so a sync that paraphrases commit messages adds nothing. Your value is judgment — what changed about the *state of the efforts*, which recorded knowledge is now false, and what new work deserves a durable page.

The bar for every entry and edit you write: **would an agent reading it learn something `git log` does not already say?** Direction changes, completions, new constraints, surprises, reversals, and abandoned approaches pass that bar. Lists of what was committed do not — if that is all a cluster amounts to, write nothing for it.

## Establish the window

1. Find the last sync marker: `hangar_logbook_search` with `category="sync"` (most recent wins), falling back to a text search for entries starting `Logbook sync:` for markers written before categories existed. The marker names the commit it synced through — its `commits` tag (or the hash in its text) — so the window is `git log <that-hash>..HEAD`.
2. No marker found? This is the first sync: use the newest entry timestamp from `hangar_logbook_index` as the boundary (`git log --since`), or, on an empty logbook, a sensible recent slice — do not attempt to backfill all history.
3. Collect the window with `git log --stat` and cluster commits by subsystem or effort, not chronology. A cluster is a story: a feature landing, a refactor arc, a bug hunt.

## Fan out readers — subagents draft, you write

Spawn one subagent per cluster, in parallel (they must not write to the logbook). Give each its commits, the relevant page bodies from `hangar_logbook_get_pages` with `bodies: "full"` (it defaults to summaries past three slugs, and a subagent judging stale claims needs the real bodies — for a very long page, have the subagent fetch it alone via `hangar_logbook_get_page` for the untruncated body), and this charge: report what *meaningfully* changed — state, capability, direction, constraints — whether any existing page now makes a false claim, whether the cluster contains new functionality durable enough for a page of its own, and a proposed entry (with the files it should tag) or page edit, written to the bar above.

In the same fan-out, spawn validators for suspect pages: any page whose `updated_at` predates window commits that touch its subject area or its tagged/file-mapped files needs review (the digest's 30-day stale flag is a floor, not the trigger). Each validator re-reads its page against the current code and docs and returns per-claim verdicts with corrections — a page contradicted by the code is a defect to fix now, not to flag.

## Write — single writer, highest bar

First assemble an impact plan from the subagent reports — change → page or entry affected → edit needed → why — and let the window size it: a handful of changed files justifies at most one or two page edits, and a plan touching more than three pages deserves a hard second look before you proceed. A no-op is a legitimate outcome: when nothing passed the bar, change nothing and let the sync marker say the logbook is already current.

- Append one entry per cluster that passed the bar, anchored to its page, tagged with the load-bearing files (`files` argument). Skip clusters that did not.
- Update pages whose summary, description, or body went false — prefer replacing the stale sentence over adding new paragraphs. Settle efforts that finished; archive pages whose subject was removed (with a final entry saying so).
- Create a new page only for durable new functionality or knowledge — the standing demand rules apply (re-derived twice, or a synthesis worth finding again). Link it from `home` and from related pages; give it an honest summary and description.
- Keep `home` current: new pages added to the map, dead ones dropped.
- Close with the sync marker: one project-level entry logged with `category: "sync"` and `commits: [<full-hash>]` (the commit you synced through), whose text is the human-readable digest — `Logbook sync: <one-line digest of what moved> (through commit <full-hash>)`. The category and commit tag are what the next sync finds; never omit the hash.

## Guardrails

- Never duplicate the commit ledger — no changelogs, no per-commit bullets, no "N commits landed."
- Respect the field roles: `summary` one line, `description` a purpose paragraph, entries short and anchored.
- Do not churn pages for cosmetic diffs (renames, formatting, comment-only changes) — validation cares about claims, not line counts.
- Entries are append-only; a wrong past entry is corrected by a new entry, never by rewriting a page to hide it.
- If the window is huge, prefer fewer, deeper clusters over many shallow ones — and say in the sync marker what you deliberately skimmed.
