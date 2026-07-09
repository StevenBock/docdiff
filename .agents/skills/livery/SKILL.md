---
name: Livery — Blueprint Design Language
description: 'Build (or refresh) a project''s blueprint design language: survey the target application''s real UI, then author the reusable artifacts — a project stylesheet, a canonical shell template, and a component gallery — that make blueprint prototypes look like the app instead of invented fiction. Use when a project has no blueprint stylesheet yet, when prototypes keep drifting from the app''s real look, or when asked to build or refresh a design system for blueprints.'
---
<!-- managed-by: hangar-agent-skills slug=livery -->

# Livery — Blueprint Design Language

A livery is an aircraft's paint scheme. This skill builds a project's blueprint design language: the reusable artifacts that make every future prototype page look like the TARGET application, not like a generic dashboard from a model's imagination. It exists because of a diagnosed failure mode: drafting agents have never seen the app, so anything the stylesheet does not specify — especially COMPOSITION (shell, sidebar, chrome) — gets invented from training priors and reads instantly wrong to the human who lives in the app.

The corrective principle, in one line: **hand future agents artifacts to copy, never prose to interpret.**

## The three artifacts

All stored through the hangar blueprint MCP tools (`hangar_set_blueprint_stylesheet`, `hangar_set_blueprint_page`), so they persist in the app and every future drafter finds them via `hangar_list_blueprint_stylesheets`:

1. **The project stylesheet** — namespaced component classes (pick a short prefix, e.g. `xy-`) covering the app's real ingredients: palette, type scale, buttons, inputs, cards, tables, badges, status dots. Use the target app's ACTUAL colors and fonts, hardcoded — prototypes must look like that app under any Hangar theme, so do not use Hangar's theme tokens (the exception is a livery for Hangar itself, which should ride its own tokens).
2. **The shell template** — a `prototype` page (slug `shell-template`) rendering the app's real chrome (sidebar/nav/topbar composition, real section names, real density) with a clearly-marked empty content region. This is the artifact that fixes the worst failure: components are ingredients, the shell is composition, and composition must be copyable. Future prototypes that show the whole app copy this page verbatim and fill the content region.
3. **The gallery** — a `prototype` page (slug `gallery`) showing every stylesheet component with realistic data. It is the fidelity contract: one glance tells a human whether the livery matches reality.

House the pages on a dedicated blueprint named for the design system (create it if absent) so they are findable and don't pollute feature blueprints.

## The process

1. **Survey the real app first — never start in CSS.** Read the actual UI source: the component directory, global styles, design tokens, layout/shell components. Extract the real values (colors, fonts, spacing, radii, sidebar width, row heights) and the real composition (what the shell contains, in what order, at what density). If the project has design docs, read them; the source still wins.
2. **Author the stylesheet** from those extracted values. Extend what exists — if the project already has a blueprint stylesheet, add and correct rather than fork.
3. **Author the shell template** from the real shell markup. Fidelity beats completeness: a plain but accurate shell is worth more than a rich fiction.
4. **Author the gallery** using every class you shipped.
5. **Verify with the human — this gate is mandatory.** You cannot see the running app; the human can. Ask them to compare the gallery and shell template against the real thing (a screenshot beside the rendered page) and correct what they flag before calling the livery done.
6. **Record the livery's scope** in a note on the design-system blueprint: which app surfaces it covers, what it deliberately omits, and the date of the source survey — so a future refresh knows where to start.

## Maintenance

- Refresh when the app's design meaningfully shifts (reruns of this skill extend the same artifacts; the survey date in the scope note tells you how stale you are).
- Never let feature blueprints carry private copies of shell markup or component CSS — if a prototype needs something new, promote it into the stylesheet/gallery so the next drafter inherits it.

## Guardrails

- Artifacts over prose: guidance for future agents belongs in class names and template pages, not paragraphs.
- Composition over ingredients: a livery without a shell template is unfinished.
- Never invent: every color, label, and layout in the artifacts traces to something you read in the app's source or the human confirmed.
