---
name: Hangar Onboarding
description: 'Onboard a project into Hangar: survey the repo, declare its dev stack and lane preparation in hangar.yml, register log files, and configure its agent instructions. Use when a project is first added, when asked to onboard or configure it for Hangar, when hangar.yml is missing, or when fresh lanes lack dependencies or generated source.'
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
- **Never hardcode a port.** A project can run in several lanes at once, and two lanes starting the same command with `PORT: 3000` collide. Write `${hangar.port}` and let Hangar allocate a durable port per lane; use `${hangar.port.<command>}` when one command needs another's. A config that only works because nobody opened a second lane is a config that breaks the first time someone does.
- **Decide placement per command.** Anything belonging to the working copy — the app, watchers, workers — runs once per lane: leave `working_dir` relative or omitted. Anything genuinely shared — a database the whole project talks to — should run once for everyone: give it an absolute `working_dir`, which pins it as a singleton that lane teardown never stops. Getting this backwards is the common mistake: a per-lane database silently gives each lane its own empty data.

## 3. Prepare lanes when enabled

- Check the running instance: lane tools must be enabled and `hangar_config_schema` must document lane preparation before declaring it. An older build needs updating first.
- Trace the repo's checks to their prerequisites, including nested dependency trees and gitignored generated source. Add exact ignored cache paths to `lanes.warm_paths`; use `lanes.setup` for the shortest existing command that prepares the lane and verifies readiness. Prefer standalone generators when a full application build is unnecessary.
- Cached dependencies are seeds: setup must install missing or stale dependencies when needed, or fail with a useful error. Keep secrets and mutable runtime data lane-local; never link writable trees back to base.
- Review the effective config, including any `hangar.local.yml` override, through the existing config trust flow. Verify a fresh warm lane from the intended base with the repo's checks before declaring preparation complete. Creation awaits setup; resolve its reported failure before handing the lane to an agent. Keep dependency and generator refresh commands discoverable for later changes in reused lanes.

## 4. Wire up observability

- Register real application log files worth tailing with `hangar_add_log_file` (files the app writes — process stdout is already captured).
- If `hangar_ping` shows the telemetry gate on and the stack can emit OpenTelemetry, note that Hangar auto-wires `OTEL_EXPORTER_OTLP_ENDPOINT` into processes it spawns and recommend instrumentation — don't force it, and never flip settings yourself.

## 5. Configure the agents

Finish with the Hangar Setup skill (`hangar-setup`, placed alongside this one — read its SKILL.md and follow it): probe the enabled tool gates and write the managed Hangar block into the project's agent memory files (`CLAUDE.md`, `AGENTS.md`), so every future agent knows it is executing inside Hangar and how to leverage this project's actual stack.

## Guardrails

- Never invent processes for technologies the repo doesn't use.
- Preserve an existing `hangar.yml` the human already shaped — extend it, don't rewrite it wholesale.
- When done, report what you configured and, for each command, the real source you traced it to.
