---
name: Blueprint Planning
description: Plan and draft a Hangar blueprint WITH the human — a collaborative planning document with a rich human-facing doc, a machine-checkable acceptance-criteria contract, and interactive prototype pages, authored through the hangar blueprint MCP tools. Use when asked to plan, draft, or refine a feature as a blueprint. To BUILD or execute a drafted blueprint, use the Blueprint Building skill instead.
---
<!-- managed-by: hangar-agent-skills slug=blueprints -->

# Blueprint Planning

A blueprint is a Hangar planning document you draft WITH the human and later build FROM. It lives in the app (materialize its complete snapshot with `hangar_download_blueprint`, then read the returned JSON file), renders in the user's theme, and carries everything a feature needs to go from intent to landed code. This skill is the DRAFTING phase: plan collaboratively, iterate on the doc, and leave a spec a separate agent can build. When the human is ready to build, a building agent takes over under the Blueprint Building skill — you do NOT write the working plan or the code.

## The artifacts — who owns what

You author, during drafting:

- **doc** (`hangar_set_blueprint_doc`) — the rich human-facing spec and the primary drafting artifact. Themed HTML; show, then tell.
- **acceptance criteria** (`hangar_set_blueprint_acceptance`) — the machine-checkable contract: one testable `- [ ]` condition per line. Author it DURING drafting so intent survives as a checklist the builder ticks. Criteria are WHAT must be true; the plan is HOW — and that plan is the builder's, not yours.
- **notes** (`hangar_add_blueprint_note`) — append-only decisions, gotchas, and links. The human's section-anchored comments arrive here: read them and address them in that section.
- **pages** (`hangar_set_blueprint_page`) — additional linked pages: `doc` articles in the house style, or `prototype` full-bleed interactive mockups that demonstrate behavior.
- **stylesheets** (`hangar_set_blueprint_stylesheet`) — the project's reusable design-system CSS for prototypes. Review what exists (`hangar_list_blueprint_stylesheets`) before authoring anything visual; extend rather than reinvent. If the project has no stylesheet (or no shell template) yet, follow the Livery skill to build the design language first — do not improvise per-blueprint styling.

You never touch:

- **brief** — the human's original statement of intent, captured at creation. Read it, honor it, never overwrite it.
- **working plan** (`hangar_set_blueprint_plan`) — the building agent writes its own at build time. Leave it alone.
- **status** — the human's tracking label, set from the app.

Edit at the scale of the change. Use `hangar_patch_blueprint_body` for small doc/page/stylesheet edits; reserve whole-field setters for restructures. During wave planning, update only the active acceptance `##` section with `hangar_set_blueprint_acceptance(scope: ...)`. `hangar_check_blueprint_step` remains the one-item tick path; address acceptance items by exact `section` + phase-local `ordinal`. Verify returned content hashes instead of re-reading large bodies. Downloaded Blueprint files are immutable point-in-time snapshots, so download again after mutations before relying on the file.

## Drafting — planning with the human

1. `hangar_download_blueprint` first, then read the returned JSON file and ground everything in the ACTUAL repo — cite real files and verify claims against the code before they go in the doc. Use `hangar_get_blueprint` only for a compact refresh or when an inline response is specifically useful.
2. Draft the doc show-then-tell: open with what the thing IS and a demo of it in action; keep implementation jargon behind a fold. Iterate with the human — drafting is a conversation, not a deliverable dump.
3. A prototype DEMONSTRATES, criteria SPECIFY: whenever a prototype implies states, flows, or edge cases the build must honor, capture them as acceptance criteria so they are checkable, not left for the builder to infer from markup.
4. Record durable decisions and gotchas as notes as you go.
5. Do NOT write the working plan — the building agent writes its own.

## Task-cards pipeline

If the blueprint's `pipeline` is `taskcards`, drafting starts the same way: doc, prototypes, and acceptance criteria first. End drafting by decomposing the build into task cards WITH the human. Cards come last, after the doc and acceptance criteria are agreed; don't decompose a moving target.

Propose the decomposition in conversation first: discrete, independently verifiable cards. Give each card `criteria` references (exact `##` section + 1-based checklist ordinal) for what it delivers. After writing the deck, call `hangar_list_task_cards(include: ["criteriaCoverage"])` and check `criteriaCoverage`: no uncovered or dangling references in the active wave. Coverage unavailable means the cards carry no references — fix the deck before finishing.

Ask the human which configured agents should work which cards. Designation is their routing preference, never your assumption; leaving cards undesignated is fine. A mistyped agent name errors back with the roster. Once agreed, write the deck with `hangar_create_task_cards`: `title`, `body_md` (what to do and how to verify), `blocked_by` deps, `phase_id` where phased, `criteria`, and `designated_agent` where chosen.

Sanity-check the response's ready list: the first ready cards must be foundations, not polish. If polish is ready first, your `blocked_by` direction is inverted; fix it now, while it is cheap. Keep deps minimal and real: A truly cannot start before B. Over-linking serializes the build. Cards are the how; acceptance criteria remain the what, so do not duplicate them. You still never write the working plan, in any pipeline.

## Rolling-wave phases — plan one wave at a time

Big, multi-phase features drown in a single flat deck: cards get compressed, criteria certify "capability exists" instead of "works + is reachable," and felt-experience feedback all lands at the very end. The fix is rolling-wave planning over first-class **phases** (`hangar_create_blueprint_phase`; a phase is `future` | `planning` | `building` | `reviewed`).

- **The doc holds the whole arc from day one; you deep-plan only the ACTIVE phase.** Write the full vision, phase map, and per-phase acceptance sections (a `##` heading per phase in `acceptance_md`) up front so nothing is lost. But author fat, self-sufficient cards + hardened criteria + a discovery-journey prototype ONLY for the phase being built now. Future phases stay doc-level intent until their turn — planning them early anchors the build to assumptions that the earlier phases will invalidate.
- **Card self-sufficiency bar.** A complex card must be executable by a FRESH agent holding only the blueprint doc + that one card, without inventing a correctness policy. It names: the concrete outcome; the owning seams/files; 2–3 load-bearing invariants; failure/lifecycle behavior (not just the happy path); a verification matrix (what to check, how); and explicit exclusions. Short cards are fine when they hide no decisions — length is not the bar, hidden decisions are.
- **Criteria certify correctness-under-lifecycle AND reachability, not mere presence.** "Capability X exists" is a weak criterion. A user-facing criterion must name the entry point (palette command, keybinding, visible control) so "works" also means "reachable." A default-off or gated capability needs a discovery/enable-affordance criterion — otherwise it ships invisible. Capture edge/failure behavior as criteria too.
- **Assign the phase's cards a `phase_id`**; leave later phases' cards uncreated. Advancing a phase is the human's call at a phase boundary — you never set phase status during drafting.
- **Keep status out of authored HTML.** Doc badges name scope only; never hand-write live phase status there. The phase strip is the single source of status truth. When planning the next wave, refresh the previous phase's prose to its terminal outcome (built & reviewed), without duplicating the live status pill.
- **The NEXT phase is planned by a FRESH agent** — not the phase builder (it holds tacit knowledge and will under-specify) and not you, the original planner (you are anchored to stale pre-build assumptions). So the builder's end-of-phase deliverable includes externalizing lessons into the Logbook + doc; plan the next wave from those durable artifacts, not from a warm chat history. The human is the continuous through-line across phases.

## Guardrails

- Never set the blueprint's status; never overwrite the brief; never write the working plan.
- The doc and pages are HTML, not markdown — real markup only; literal `- [ ]` task items belong solely in the acceptance checklist, where they drive live progress counts.
- Keep prototype pages looking like the app you are building, not like Hangar — attach the project stylesheet.
- Prototype the FEATURE, not the app around it: default to a full-bleed panel/page mock with no surrounding shell. If the demo genuinely needs the app's shell (sidebar, window chrome), copy the design system's canonical shell template verbatim (for Hangar itself: the `shell-template` page on the design-system blueprint) — never invent navigation, sidebars, or chrome from imagination. You have components in the stylesheet but no compositional knowledge of the real app; an invented shell reads as instantly wrong to the human who lives in it.
