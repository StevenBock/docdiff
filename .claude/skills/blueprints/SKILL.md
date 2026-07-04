---
name: Blueprints
description: Plan and build Hangar blueprints — collaborative planning documents with a human-facing rich doc, an acceptance-criteria contract, interactive prototype pages, and a lane-based build phase, all through the hangar blueprint MCP tools. Use when asked to plan or draft a feature as a blueprint, refine an existing blueprint, build/execute a blueprint, or land a build lane.
---
<!-- managed-by: hangar-agent-skills slug=blueprints -->

# Blueprints

A blueprint is a Hangar planning document you draft WITH the human and later build FROM. It lives in the app (read it with `hangar_get_blueprint`), renders in the user's theme, and carries everything a feature needs to go from intent to landed code: the human's brief, a rich doc, acceptance criteria, prototype pages, notes, and — at build time — the builder's own working plan. Two phases, usually two different agents: **drafting** (plan collaboratively, iterate on the doc) and **building** (execute, usually in a lane). Landing closes the build.

## The artifacts — who owns what

- **brief** — the human's original statement of intent, captured at creation. Read it, honor it, never overwrite it.
- **doc** (`hangar_set_blueprint_doc`) — the rich human-facing spec and the primary DRAFTING artifact. Themed HTML; show, then tell.
- **acceptance criteria** (`hangar_set_blueprint_acceptance`) — the machine-checkable contract: one testable `- [ ]` condition per line. Authored DURING drafting so intent survives as a checklist; ticked during the build as each condition holds. Criteria are WHAT must be true; the plan is HOW the builder gets there.
- **working plan** (`hangar_set_blueprint_plan`) — the BUILDER's own `- [ ]` checklist, written at build time by the agent that will execute it. Never written during drafting.
- **notes** (`hangar_add_blueprint_note`) — append-only decisions, gotchas, and links. The human's section-anchored comments arrive here: read them and address them in that section.
- **pages** (`hangar_set_blueprint_page`) — additional linked pages: `doc` articles in the house style, or `prototype` full-bleed interactive mockups that demonstrate behavior.
- **stylesheets** (`hangar_set_blueprint_stylesheet`) — the project's reusable design-system CSS for prototypes. Review what exists (`hangar_list_blueprint_stylesheets`) before authoring anything visual; extend rather than reinvent.
- **status** — the HUMAN's tracking label, set from the app. Never set it.

Every write is whole-field and returns a content hash — verify round trips against it instead of re-reading large bodies. The one exception is `hangar_check_blueprint_step`, which ticks a single checklist item in place; always prefer it over resending a whole plan or acceptance body.

## Drafting — planning with the human

1. `hangar_get_blueprint` first, then ground everything in the ACTUAL repo — cite real files and verify claims against the code before they go in the doc.
2. Draft the doc show-then-tell: open with what the thing IS and a demo of it in action; keep implementation jargon behind a fold. Iterate with the human — drafting is a conversation, not a deliverable dump.
3. A prototype DEMONSTRATES, criteria SPECIFY: whenever a prototype implies states, flows, or edge cases the build must honor, capture them as acceptance criteria so they are checkable, not left for the builder to infer from markup.
4. Record durable decisions and gotchas as notes as you go.
5. Do NOT write the working plan — the building agent writes its own.

## Building — executing a blueprint

Builds usually run in a fresh lane (git worktree + branch) so the work cannot clobber the base checkout.

1. Read the WHOLE blueprint: doc, brief, notes, acceptance criteria, and every linked page.
2. Prototypes are the BEHAVIORAL spec — study what they DO (states, flows, edge cases) and reproduce that, while matching the target app's existing design system for how it looks. The visuals are mockups, not pixel specs.
3. Explore the repo, then write YOUR OWN plan (`hangar_set_blueprint_plan`): a `- [ ]` checklist of meaningful steps — it drives the live progress bar the human watches — plus your approach and how you'll verify each step. It's your plan; you decide the steps.
4. Implement it fully: real, working code — no stubs, placeholders, or TODOs unless the plan explicitly calls for them. Tick a plan step (`hangar_check_blueprint_step`) only once its code is written AND verified; tick an acceptance item only once its condition actually holds.
5. Resuming? Already-checked steps are a previous session's claim — confirm each against the repo before trusting it.

## Landing

Landing is conversational — wait for the human to say land, then:

1. `hangar_sync_lane` to merge the base branch back in; resolve any conflicts in the lane's worktree, commit, and sync again until clean.
2. Run the project's checks/tests in the lane and confirm green.
3. Land with `hangar_land_lane` (`remove_worktree:true`). It refuses a dirty or conflicted tree, so commit everything first. Hangar defers physical cleanup while lane processes are still alive, so report what landed before the worktree disappears.

## Guardrails

- Never set the blueprint's status; never overwrite the brief.
- The doc and pages are HTML, not markdown — real markup only; literal `- [ ]` task items belong solely in the plan and acceptance checklists, where they drive live progress counts.
- Keep prototype pages looking like the app you are building, not like Hangar — attach the project stylesheet.
