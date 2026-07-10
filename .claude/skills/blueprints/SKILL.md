---
name: Blueprint Planning
description: Plan and draft a Hangar blueprint WITH the human — a collaborative planning document with a rich human-facing doc, a machine-checkable acceptance-criteria contract, and interactive prototype pages, authored through the hangar blueprint MCP tools. Use when asked to plan, draft, or refine a feature as a blueprint. To BUILD or execute a drafted blueprint, use the Blueprint Building skill instead.
---
<!-- managed-by: hangar-agent-skills slug=blueprints -->

# Blueprint Planning

A blueprint is a Hangar planning document you draft WITH the human and later build FROM. It lives in the app (read it with `hangar_get_blueprint`), renders in the user's theme, and carries everything a feature needs to go from intent to landed code. This skill is the DRAFTING phase: plan collaboratively, iterate on the doc, and leave a spec a separate agent can build. When the human is ready to build, a building agent takes over under the Blueprint Building skill — you do NOT write the working plan or the code.

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

Every write is whole-field and returns a content hash — verify round trips against it instead of re-reading large bodies. `hangar_check_blueprint_step` ticks a single acceptance condition in place; prefer it over resending a whole acceptance body.

## Drafting — planning with the human

1. `hangar_get_blueprint` first, then ground everything in the ACTUAL repo — cite real files and verify claims against the code before they go in the doc.
2. Draft the doc show-then-tell: open with what the thing IS and a demo of it in action; keep implementation jargon behind a fold. Iterate with the human — drafting is a conversation, not a deliverable dump.
3. A prototype DEMONSTRATES, criteria SPECIFY: whenever a prototype implies states, flows, or edge cases the build must honor, capture them as acceptance criteria so they are checkable, not left for the builder to infer from markup.
4. Record durable decisions and gotchas as notes as you go.
5. Do NOT write the working plan — the building agent writes its own.

## Task-cards pipeline

If the blueprint's `pipeline` is `taskcards`, drafting starts the same way: doc, prototypes, and acceptance criteria first. End drafting by decomposing the build into task cards WITH the human. Cards come last, after the doc and acceptance criteria are agreed; don't decompose a moving target.

Propose the decomposition in conversation first: discrete, independently verifiable cards, each sized so a fresh agent could execute it in one verification cycle with no context beyond the blueprint doc and the card itself. A deck over roughly 30 cards is a decomposition smell. Before writing the deck, check coverage: every acceptance criterion must be delivered by at least one card. A criterion with no card is a plan gap you just caught early.

Ask the human which configured agents should work which cards. Designation is their routing preference, never your assumption; leaving cards undesignated is fine. A mistyped agent name errors back with the roster. Once agreed, write the deck with `hangar_create_task_cards`: `title`, `body_md` (what to do and how to verify), `blocked_by` deps, and `designated_agent` where chosen.

Sanity-check the response's ready list: the first ready cards must be foundations, not polish. If polish is ready first, your `blocked_by` direction is inverted; fix it now, while it is cheap. Keep deps minimal and real: A truly cannot start before B. Over-linking serializes the build. Cards are the how; acceptance criteria remain the what, so do not duplicate them. You still never write the working plan, in any pipeline.

## Guardrails

- Never set the blueprint's status; never overwrite the brief; never write the working plan.
- The doc and pages are HTML, not markdown — real markup only; literal `- [ ]` task items belong solely in the acceptance checklist, where they drive live progress counts.
- Keep prototype pages looking like the app you are building, not like Hangar — attach the project stylesheet.
- Prototype the FEATURE, not the app around it: default to a full-bleed panel/page mock with no surrounding shell. If the demo genuinely needs the app's shell (sidebar, window chrome), copy the design system's canonical shell template verbatim (for Hangar itself: the `shell-template` page on the design-system blueprint) — never invent navigation, sidebars, or chrome from imagination. You have components in the stylesheet but no compositional knowledge of the real app; an invented shell reads as instantly wrong to the human who lives in it.
