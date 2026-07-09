---
name: Fleet Drafting — blueprint fleet protocol
description: Run a fleet of feature ideas from concept to fully-ruled blueprints (and built lanes) in one orchestrator session using delegated drafter and builder agents. Use when asked to "run the fleet protocol", "draft a fleet of blueprints", "blueprint fleet", a "killer features session", or to brainstorm/draft several features at once as blueprints.
---
<!-- managed-by: hangar-agent-skills slug=fleet-drafting -->

# Fleet Drafting — blueprint fleet protocol

You are the **orchestrator**. Your job is to take a batch of feature ideas from concept to **fully-ruled blueprints** — and, for the ready ones, to built code in lanes — in one session, by delegating to fresh Opus sub-agents and driving every open question to a recorded ruling *before* any builder starts.

The one principle everything serves: **no decision gets made twice, and nothing is left for a builder to guess.** A blueprint is not done when it reads well; it is done when a fresh sweep for open questions comes back empty.

Run it in five phases. Announce your phase with `hangar_set_activity` as you go.

## Phase 1 — Draft (one drafter per feature)

Spawn **one Opus drafter agent per feature** (`hangar_spawn_agent`, or the Agent tool with an Opus model). Give each a **concept brief**, not a one-liner. A good brief has:

- **Vision** — what the feature is and the user value, in a few sentences.
- **Synergy points** — how it should lean on / feed sibling features in this same fleet.
- **Starting design positions** — your current opinions, framed as *positions to react to*, not a spec. You want the drafter to push back.
- **Grounding docs** — the exact module docs / logbook pages / files to read first (name them).
- **Constraints** — hard limits (gate posture, retention bounds, threading, security).
- **The blueprint protocol** — tell the drafter to use the `blueprints` skill and author a real blueprint (rich human doc, machine-checkable acceptance criteria, prototype pages) via the `hangar_*_blueprint*` MCP tools.

**Make the grounding non-negotiable:** the drafter must ground every claim in the actual repo and is **expected to correct your premises from source**. Your brief is a hypothesis. A drafter that discovers the premise is wrong (a coarse enum you wanted to tap, a capability that's actually inverted, a bridge that's debug-only) is doing its job — tell it explicitly that correcting you is a success, not a deviation. Expect whole execution models to pivot mid-draft.

## Phase 2 — Question sweep (enumerate, fix nothing)

When a drafter reports its draft done, send it back in to **re-read its own blueprint and enumerate every open question**: TODOs, alternatives named without a winner, tradeoffs punted to the builder, ambiguous acceptance criteria. A **numbered list. No fixes yet.**

The separation is the point. If drafting and resolving happen in one pass, a drafter quietly resolves questions in its own head and never surfaces them — and those are exactly the decisions a builder later re-makes differently. **If it isn't on the list, it isn't ruled.**

## Phase 3 — Rulings (you answer everything, once)

Answer **every** question with **one line of reasoning each**, and send the batch back. Instruct the drafter to:

1. Record a **"Rulings" note verbatim** on the blueprint (`hangar_add_blueprint_note`) — the recorded ruling is what the builder will read, so it must survive verbatim.
2. Apply each ruling through the doc, acceptance criteria, and prototype pages.
3. **Sweep again** (phase 2) and report the new list.

**Loop until a sweep returns empty.** "Fully ruled" is that observable state, not a vibe. Rule decisively — a one-line rationale is enough; you are removing ambiguity, not writing an essay. When a drafter proposes a genuinely load-bearing, non-obvious fork, surface it to the human; otherwise decide and move on.

## Phase 4 — Build (fresh builders, in lanes, rulings BINDING)

For each blueprint the human green-lights, spawn a **fresh builder agent** (the `blueprint-builder` skill) working in **its own lane**. Tell each builder:

- The **Rulings notes are BINDING**, not advisory. Do not re-decide settled questions.
- Tick acceptance criteria **only when verifiably true**. Leave live-app and **visual-fidelity criteria for the human** — do not self-approve those.
- Run the full gate suite **in-lane**: Rust `cargo test -- --test-threads=1` (from `src-tauri/`), `npm run check`, `npm run test`, docdiff clean.
- **Do NOT land.** You (the orchestrator) sequence landings.

## Phase 5 — Landing coordination (you sequence the merges)

Parallel lane builds collide on shared **monotonic** resources. Land **one lane at a time, syncing with `main` between each**, and fix collisions by hand:

- **Migration renumbering** — two lanes both grabbed the next `V<N>`. After the first lands, renumber the second (e.g. `V57 → V58`) via the registry in `src-tauri/src/db/mod.rs` and its `include_str!`.
- **MCP tool-count assertions** — features that add tools bump a hard-coded tool-count test. Lands stack, so re-base the next lane's assertion on the landed total (e.g. `126 → 127` after one new tool).
- **Re-run every gate after each sync** before landing that lane.

Write the landing runbook (order, per-lane migration/tool-count deltas, commits) into the logbook before you start landing, so it survives a session boundary.

## Standing rules (these made it work)

- **Prototypes never invent app chrome.** Drafters have the design ingredients but will compose a fictional sidebar from generic-dashboard priors unless stopped. Default to **full-bleed, feature-only** prototypes; when app context is genuinely needed, copy the canonical `shell-template` prototype **verbatim** and only drop panel content into it. For a non-Hangar project, run the **Livery** skill first to build that project's blueprint design language (survey real UI source → stylesheet + shell template + gallery → human fidelity gate).
- **Verify adversarial sub-agent reports against ground truth before acting.** A sub-agent can report confidently on the wrong thing (e.g. an app-driver observing the wrong instance). Check a claim against source or a second observation before you act on it.
- **The human supplies taste, screenshots, and lived context.** Visual-fidelity gates are **human-only**. You do not screenshot-and-approve your own fleet.

## Close out

Log what moved (`hangar_logbook_log`): the fleet drafted, ids + status, and the landing runbook. The durable rationale for this whole method lives on the `fleet-drafting-protocol` logbook page — read it if you need the *why* behind a phase.
