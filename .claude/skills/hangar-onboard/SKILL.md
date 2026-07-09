---
name: Hangar Onboarding
description: 'Onboard a project into Hangar: survey the repo, declare its dev stack in hangar.yml, register log files worth tailing, and finish by configuring the project''s agent instructions through the Hangar Setup skill. Use when a project is first added to Hangar, when asked to onboard or configure a project for Hangar, or when hangar.yml does not exist yet.'
---
<!-- managed-by: hangar-agent-skills slug=hangar-onboard -->

# Hangar Onboarding

You are onboarding this project into Hangar — turning a plain directory into a declared, observable dev workspace. This runs when a project is first added (Hangar's setup flow may have spawned you for exactly this) or whenever the human asks to onboard or reconfigure a project. Narrate what you find as you go; when a human is driving interactively, show the proposed config before the first write — a setup brief that tells you to configure directly means proceed and report.

## 1. Survey the repo

Find what actually runs this project:

- `package.json` scripts (dev, start, serve, watch), `Makefile` / `Procfile` / `docker-compose.yml`, `Cargo.toml`, `go.mod`, `manage.py`, `artisan`, `Gemfile` — plus the databases, queue workers, and background watchers those imply.
- Before including any command, confirm it resolves to something real: a script, a target, a compose service, or an installed binary. Never write an aspirational config.

## 2. Declare the stack — hangar.yml

- Call `hangar_config_schema` FIRST for the exact format, every field, the valid icons, and worked examples — read it before writing so you don't guess the shape.
- Long-running dev commands (dev server, watchers, databases, queues) become processes: descriptive names ("Dev Server", not the raw command), matching icons, `auto_restart: true` where a crash should self-heal, `depends_on` where start order matters.
- One-off tasks worth a button (seed the DB, run migrations, full test pass) become on-demand commands (`autostart: false`) instead of staying undiscoverable — they still start their `depends_on` chain when triggered.
- Do not declare one-off build/test/lint steps as autostarting processes.

## 3. Wire up observability

- Register real application log files worth tailing with `hangar_add_log_file` (files the app writes — process stdout is already captured).
- If `hangar_ping` shows the telemetry gate on and the stack can emit OpenTelemetry, note that Hangar auto-wires `OTEL_EXPORTER_OTLP_ENDPOINT` into processes it spawns and recommend instrumentation — don't force it, and never flip settings yourself.

## 4. Configure the agents

Finish with the Hangar Setup skill (`hangar-setup`, placed alongside this one — read its SKILL.md and follow it): probe the enabled tool gates and write the managed Hangar block into the project's agent memory files (`CLAUDE.md`, `AGENTS.md`), so every future agent knows it is executing inside Hangar and how to leverage this project's actual stack.

## Guardrails

- Never invent processes for technologies the repo doesn't use.
- Preserve an existing `hangar.yml` the human already shaped — extend it, don't rewrite it wholesale.
- When done, report what you configured and, for each command, the real source you traced it to.
