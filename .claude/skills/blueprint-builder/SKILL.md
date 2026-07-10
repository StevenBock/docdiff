---
name: Blueprint Building
description: Execute a drafted Hangar blueprint and land it — read the doc, acceptance criteria, and prototype pages, write your own working plan, implement real code (usually in a lane), tick criteria as they hold, and land the build lane, all through the hangar blueprint MCP tools. Use when asked to build, execute, or land a blueprint. To plan, draft, or refine a blueprint first, use the Blueprint Planning skill instead.
---
<!-- managed-by: hangar-agent-skills slug=blueprint-builder -->

# Blueprint Building

A blueprint is a Hangar planning document drafted WITH the human and built FROM. It lives in the app — read the whole thing with `hangar_get_blueprint`. This skill is the BUILD phase: turn a drafted blueprint into real, landed code, usually in a fresh lane (git worktree + branch) so the work cannot clobber the base checkout. Drafting is done by the time you start; if you are instead being asked to plan or draft, use the Blueprint Planning skill.

## The artifacts — what you read vs write

You read and honor — never change:

- **brief** — the human's original statement of intent. Honor it, never overwrite it.
- **doc** — the rich human-facing spec.
- **acceptance criteria** — the machine-checkable contract: `- [ ]` conditions that must end up true. Tick each only once it actually holds.
- **notes** — decisions, gotchas, and the human's section-anchored comments; read them and address them.
- **pages** — linked `doc` articles and `prototype` mockups. The prototypes are the BEHAVIORAL spec.

You write:

- **working plan** (`hangar_set_blueprint_plan`) — YOUR OWN `- [ ]` checklist, written at build time after exploring the repo. It drives the live progress bar the human watches. It's your plan; you decide the steps.

You never set the blueprint's **status** — that's the human's, from the app.

Every write is whole-field and returns a content hash — verify round trips against it instead of re-reading large bodies. The exception is `hangar_check_blueprint_step`, which ticks a single checklist item (a plan step or an acceptance condition) in place; always prefer it over resending a whole plan or acceptance body.

## Building — executing a blueprint

1. Read the WHOLE blueprint: doc, brief, notes, acceptance criteria, and every linked page.
2. Prototypes are the BEHAVIORAL spec — study what they DO (states, flows, edge cases) and reproduce that, while matching the target app's existing design system for how it looks. The visuals are mockups, not pixel specs.
3. Explore the repo, then write YOUR OWN plan (`hangar_set_blueprint_plan`): a `- [ ]` checklist of meaningful steps — it drives the live progress bar — plus your approach and how you'll verify each step. It's your plan; you decide the steps.
4. Implement it fully: real, working code — no stubs, placeholders, or TODOs unless the plan explicitly calls for them. Tick a plan step (`hangar_check_blueprint_step`) only once its code is written AND verified; tick an acceptance item only once its condition actually holds.
5. Resuming? Already-checked steps are a previous session's claim — confirm each against the repo before trusting it.

## Task-cards pipeline

If the blueprint's `pipeline` is `taskcards`, do NOT write a working plan — the plan already exists as task cards. Replace steps 3–4 with this card loop:

0. Deck review first: read ALL cards before executing any. The deck was written with less code context than you have now. Recommend and proceed: fix missized open cards yourself by rewriting title/body (never a done card's), note what you changed, and stop for the human only on load-bearing ambiguity. Redecomposing? Cancel with the reason, refile, and re-point the cancelled card's dependents to the replacement.
1. `hangar_list_task_cards` with status `ready` — cards whose blockers are all done. Your build seed names the configured agent you were spawned as: only take cards designated to you or undesignated. When the only remaining ready cards are designated to a different agent, stop and tell the human which agent the graph is waiting on. The human can always override a designation conversationally.
2. Take one: mark it `in_progress`, implement it, verify it, commit the card's work (explicit paths), then mark it `done` with a `resolution_note_md`. The note records what you built, files touched, how you verified it, anything surprising, and the commit hash. Per-card commits make a half-done deck resumable; uncommitted work dies with your process. Done or cancelled without a note is rejected. A card waiting on a human decision stays `in_progress` with a note saying what it needs — raise it in conversation and take another ready card meanwhile.
3. Work you discover that no card covers: if an acceptance criterion needs it, file it as a new card (`discovered_from` = the card you were on); if it is nice-to-have, add a blueprint note and let the human decide. Every open card is mandatory before the build ends, so filing polish inflates the build.
4. Repeat until no open or in_progress cards remain. A parked waiting-on-human card is not done. Tick acceptance criteria (`hangar_check_blueprint_step`) as each condition actually holds, same as any build.

When `ready` comes back empty but the deck is not done, do not spin: list all cards and say what is blocked on what. An `in_progress` card is someone else's; leave it unless the human hands it to you. When taking one over, re-verify its partial work against the repo first because resolution notes cite per-card commits. Finished cards that unblock another agent's designated cards? Message that agent (`hangar_message_agent`) if it is alive; otherwise tell the human.

## Landing

Landing is conversational — wait for the human to say land, then:

1. `hangar_sync_lane` to merge the base branch back in; resolve any conflicts in the lane's worktree, commit, and sync again until clean.
2. Run the project's checks/tests in the lane and confirm green.
3. Land with `hangar_land_lane` (`remove_worktree:true`). It refuses a dirty or conflicted tree, so commit everything first. Hangar defers physical cleanup while lane processes are still alive, so report what landed before the worktree disappears.

## Guardrails

- Never set the blueprint's status; never overwrite the brief.
- The doc and pages are HTML, not markdown; literal `- [ ]` task items belong solely in the plan and acceptance checklists, where they drive live progress counts.
- Keep any prototype pages you touch looking like the app you are building, not like Hangar — attach the project stylesheet.
- If you touch a prototype that mocks the app's surrounding shell (sidebar, window chrome), keep it on the design system's canonical shell template (for Hangar itself: the `shell-template` page on the design-system blueprint); never invent navigation or chrome.
- New user-facing surface (route, panel, view family)? Register it per the target project's documentation conventions before landing — module doc, doc map/index entries, code annotations. Projects that gate doc coverage will fail their checks on an unmapped surface.
