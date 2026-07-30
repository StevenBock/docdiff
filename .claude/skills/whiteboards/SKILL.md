---
name: Whiteboard Design
description: 'The visual grammar for drawing a Hangar whiteboard: layer frames as strata, one or two named hero paths, shape and tone semantics, density limits, a derived legend, and the diagnostics-then-render check loop. Read BEFORE drawing or revising a board with the hangar whiteboard MCP tools, so a board reads as one system instead of a rediscovered style.'
---
<!-- managed-by: hangar-agent-skills slug=whiteboards -->

# Whiteboard Design

A whiteboard is a Hangar diagram YOU draw and a human only reads — the renderer has no editing affordances, so every correction comes back to you as an ask. Draw for reading. This skill is the board design language: use it before your first `hangar_add_shape` and again before you call a board done.

The corrective principle, in one line: **a board is a document with one story, not a dump of everything that exists.**

## Shape and tone semantics — fixed meanings

Pick kinds by MEANING, not by variety:

- `ellipse` — an external actor (human, other process, remote service).
- `cylinder` — a datastore (SQLite, Parquet, a file on disk).
- `box` — a code module or component. The default.
- `frame` — a layer or subsystem; a navigable container, not decoration.
- `sticky` — a note or invariant. `code` — a real snippet. `table` — a matrix.
- `legend` — the board's key. Its rows are derived from the scene, not typed (see below).
- `entity_link` — use it whenever the node IS a live Hangar object (`refKind`: process / lane / blueprint / squawk / file). It resolves current status and navigates on activation, so it beats a hand-labeled box that goes stale.

Tone is semantic too. Assign **one tone per edge family** and ship at most ~4 families on one board — e.g. `primary` = hero call path, `accent` = the second hero path, `info` = events, `muted` + `dash: "dotted"` = structural, containment, and persistence edges. Never tone-code for prettiness.

Every non-trivial board carries a **`legend` shape** — one shape, no entries to write. The renderer derives its rows from the board itself (every tone and dash family in use, every named path, the conventional shape kinds present), so a family you delete leaves the legend on its own and it can never go stale. Give the rows words with optional captions: `content = {tones: {"primary": "UI call path"}, dashes: {"dotted": "structural"}, kinds: {"cylinder": "datastore"}, paths: {"request": "cold-start request"}}`. An uncaptioned family still lists its token name, so the gap shows. When the board argues a contract, add a short sticky of the key invariants too. A reader should never have to infer your code.

## Layer frames as strata

Architecture boards read as horizontal layer frames, top to bottom: UI → boundary/IPC → runtime → core → OS and external edges. One frame per stratum, named for the stratum. Because a frame is a navigable container (the human drills into it), a frame must be a real subsystem, so nesting has to mean something. Balance them: split a frame that gets crowded instead of shrinking what is inside it.

## Hero paths

- Choose **at most 1–2 primary flows** per board. Everything else is context around them.
- Give each hero its own tone and make it a **named path**: put `style.path = {"name": "request", "step": 10}` on every connector in the flow. The renderer numbers the path 1, 2, 3… by step ORDER on the label chips, and the human can WALK it step by step in the viewer. Never hand-number a label (`1 · invoke`) — that goes stale the moment you insert a step. Leave gaps in the steps (10, 20, 30) so inserting one needs no renumbering; two connectors sharing a step is flagged back to you by `hangar_get_whiteboard`.
- **Animate ONLY the hero paths** (`style.animation.kind: "flow"`). A board where everything moves says nothing moves. A flow connector touching an entity_link can carry `bind: "status"` so it plays only while that object is alive.
- Keep secondary edges out of the hero corridor; reroute or restructure rather than crossing it.

## Density limits

- A node above ~6 edges is a star, not a hub: introduce a **bus** node (a thin box the spokes meet) and route through it.
- Short labels on chips. Prose belongs in `detail` — agent-authored markdown shown in the popover — never as long on-canvas text.
- `content.subtitle` is one mono line: a file path, a type name, a port. Not a sentence.
- If a frame needs a paragraph to explain itself, it is two frames.

## Layout and the check loop

1. **Omit `x` and `y`.** Auto-layout ranks the board left→right along the arrows and is deterministic. Pin coordinates only for a deliberate skeleton (typically the layer frames), and pin BOTH axes when you do — one axis alone is not a pin.
2. Read `hangar_get_whiteboard` back: it returns shape overlaps, connector crossings, counts, and the bounding box for the pinned shapes. Fix what it flags and re-check.
3. **Finish with `hangar_render_whiteboard` and actually look at the PNG** before declaring the board done. Clean diagnostics are not legibility; only the render shows crowding, label collisions, and a hero path you cannot follow. Render a single frame when the whole board is too dense to judge.

## Guardrails

- Never a raw color. `tone` and `accent` are keywords that map to the user's theme; a hardcoded color is wrong under every other theme.
- Boards are documents, not apps: motion auto-plays and loops, and is never an interactive control.
- Do not hand-write a status word (running, failed) where an `entity_link` would show it live.
- Revising an existing board: read it with `hangar_get_whiteboard` first and extend its existing grammar — a second visual language on one board is worse than a mediocre first one.
- Delete before you add. A board that answers one question well beats a complete inventory nobody reads.
