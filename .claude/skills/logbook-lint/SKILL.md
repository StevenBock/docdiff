---
name: Logbook Lint
description: 'Curate the health of the project Logbook as a whole: find contradictions between pages, merge near-duplicates, adopt or archive orphans, triage stale sagas, repair weak summaries and descriptions, and keep home an honest map. Use when asked to lint, clean up, or curate the logbook, or on a periodic schedule. Complements logbook-sync, which reconciles pages against commits — lint reconciles the wiki against itself.'
---
<!-- managed-by: hangar-agent-skills slug=logbook-lint -->

# Logbook Lint

You curate the Logbook as a whole. Sync keeps individual pages true to the commit stream; your job is the wiki-level health that no single write protects: pages that contradict each other, subjects split across near-duplicates, knowledge nobody can find, sagas nobody settled, and an index that no longer tells the truth. A wiki that visibly decays loses every future agent's trust — curation is what keeps it worth reading.

The bar for every action: it must leave the wiki easier to orient in. If a check finds nothing wrong, do nothing — churn to look busy is itself a defect.

## Survey cheaply first

1. `hangar_logbook_graph` — the whole shape: pages, links, unresolved targets, orphans.
2. `hangar_logbook_get_pages` with `bodies:"summaries"` across everything — titles, summaries, descriptions, statuses, timestamps without paying for bodies.
3. Fetch full bodies only for pages a check has made suspect.

## The checks — fan out subagents, one per check or per link-neighborhood

Subagents investigate and draft; they never write. You apply their verdicts as the single writer.

- **Contradictions.** Pages in the same link neighborhood making conflicting claims (two pages describing the same boundary differently, a saga saying an effort is live while another page calls it replaced). Verdict names both pages and which claim is right — checked against the repo, not majority vote.
- **Near-duplicates.** Two pages about one subject. Merge: the better page absorbs what the other adds, the loser is archived with a final entry — `Merged into [[winner]]` — and `home` plus inbound links are repointed. Never delete; archive with a pointer. The same check applies below page level: a concept explained in full on several pages gets one canonical home, and the other pages link to it instead of repeating it.
- **Orphans and unresolved targets.** Orphans (unreachable from `home`) get adopted — linked from `home` or a related page — or archived if they earned obscurity. Unresolved link targets are promises: write the page if the promise is still worth keeping, or remove the link where it is noise.
- **Stale sagas.** `active` pages long untouched: the effort moved (page must say so — flag for a sync run rather than inventing content), finished (`settled`), or died (`archived`, with a final entry saying why). A stale `active` saga misleads every launch digest.
- **Weak summaries and descriptions.** A summary that does not say what the page holds, a description that merely echoes the summary, a stub page pretending to be more. Rewrite honestly — including describing a stub as a stub. A stub with no growth in sight is better merged into a broader page (or its content folded into `home`) than left standing.
- **Home accuracy.** Every non-archived page mapped, categories still sensible, nothing dead still advertised. Home leads every agent's launch digest; its accuracy is the whole wiki's first impression.

## Write — single writer, minimal motion

Apply merges, status changes, link repairs, and summary/description rewrites with `hangar_logbook_upsert_page` (whole-field writes — resend unchanged fields; verify returned hashes). Never edit history: entries are append-only, and a wrong old entry is answered by a new entry, not a rewrite. Close with one entry recording the run: what was merged, archived, adopted, rewritten, and what was checked and found healthy — so the next lint knows where the last one stood.

## Guardrails

- Judgment, not mechanism: never bulk-apply a rule a human would wince at. When a verdict is genuinely uncertain, log it as a question in the closing entry instead of acting.
- Do not rewrite prose that is true for style points; do not touch `body_md` beyond what a check demands.
- Respect the field roles (summary one line, description a purpose paragraph) and the demand rules — lint removes and consolidates far more often than it creates.
- Contradiction verdicts must cite the repo (code, docs, or commits), never just the other page.
