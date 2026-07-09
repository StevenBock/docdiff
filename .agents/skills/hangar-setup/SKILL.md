---
name: Hangar Setup
description: 'Align this project''s agent instructions with what Hangar actually exposes: probe the enabled MCP tool gates, then write or refresh the marker-delimited Hangar block in the project''s agent memory files (CLAUDE.md, AGENTS.md), adding guidance for enabled capabilities and removing it for disabled ones. Use when asked to set up agents for Hangar, refresh or sync Hangar instructions, or after the developer changes Hangar settings or tool gates.'
---
<!-- managed-by: hangar-agent-skills slug=hangar-setup -->

# Hangar Setup

You align this project's durable agent instructions with what Hangar actually exposes. The developer runs this after changing Hangar settings or tool gates, or as the last step of onboarding — so every future agent (including ones connected to Hangar's MCP server without being spawned by it) starts with accurate, project-specific guidance instead of stale or generic boilerplate.

## Audit — measure, never assume

- `hangar_ping` is the source of truth for what is enabled: the instance name plus a `gates` map (telemetry, blueprints, lanes, databases, logbook, activity, …). Never infer capability from memory, from a stale session, or from what a previous run wrote into the files.
- `hangar_list_processes` and the project's `hangar.yml` — the declared stack: what runs and what each process is for.
- `hangar_list_log_files` — registered log files and their parsed columns.
- Config gaps you notice (no `hangar.yml`, an obvious dev server undeclared, a log file worth registering) belong to onboarding: surface them and point at the Hangar Onboarding skill rather than silently rewriting config from here.

## The managed block

Maintain exactly one marker-delimited block in each agent memory file that exists at the project root (`CLAUDE.md`, `AGENTS.md`); if none exists, create the file native to the agent you are:

```
<!-- hangar:instructions:begin -->
…
<!-- hangar:instructions:end -->
```

Replace the whole block on every run — never edit inside it incrementally, never touch anything outside the markers, and keep its existing position in the file. Wholesale replacement is what removes guidance for capabilities the developer has since disabled: no section may survive for a gate that is now off.

## What goes in the block

Open with the fact that carries everything else: this project is developed inside Hangar, and an agent that can see the `hangar_*` MCP tools is executing inside Hangar's workspace — the dev servers, watchers, and sibling agents around it are live and controllable, not descriptions.

Hangar's MCP server already documents its tools generically at session start — do not restate tool documentation. The block earns its place by being PROJECT-TUNED: name this project's real processes, commands, log files, and databases so an agent knows how to use the running stack, not just that tools exist.

Compose short imperative sections, a few lines each, only for what is actually on:

- **Always:** which running processes hold the answers — read a watcher or dev server's bounded output (`hangar_read_process_output`) instead of re-running builds and tests; the on-demand commands worth triggering; `hangar_stack_health` after changes.
- **telemetry:** query traces/logs/metrics through the telemetry tools instead of adding print statements; name the services that report.
- **lanes:** risky or multi-commit work goes to a lane, not the base checkout.
- **blueprints:** substantial features get planned as blueprints with the human.
- **databases:** name the managed connections and what each holds.
- **logbook:** read the Logbook before resuming long-running work; log what moved before finishing.
- **activity:** publish a short present-tense activity line when focus shifts.

Skip capabilities that do not change how an agent works on this project's code (themes, canvas environments, UI control). Keep the whole block under ~40 lines — every line costs context in every future session.

## Guardrails

- Reference only tools you can currently see; a gated-off tool named in instructions sends future agents into dead ends.
- Never modify text outside the markers.
- Do not write `hangar.yml`, register log files, or change settings from this skill — recommend, and hand off to onboarding.
- The result must read correctly whether created fresh or refreshed for the tenth time.
