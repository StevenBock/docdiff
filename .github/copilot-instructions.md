# GitHub Copilot Instructions for docdiff

## Project Overview

**docdiff** is a language-agnostic CLI tool that detects stale documentation by tracking `@doc` annotations in source code comments.

**Key Features:**
- Dependency-aware issue tracking via bd (beads)
- Multi-language support (Go, Java, PHP, Python, JavaScript, Ruby)
- Multiple output formats (human, JSON, SARIF)

## Tech Stack

- **Language**: Go 1.23+
- **CLI Framework**: Cobra
- **Testing**: Go standard testing
- **CI/CD**: GitHub Actions, GitLab CI

## Coding Guidelines

### Testing
- Always write tests for new features
- Run `go test ./...` before committing
- Use table-driven tests following existing patterns

### Code Style
- Follow existing patterns in `internal/` packages
- Use the strategy pattern for new language support
- Keep functions focused and small

## Issue Tracking with bd

**CRITICAL**: This project uses **bd** for ALL task tracking. Do NOT create markdown TODO lists.

### Essential Commands

```bash
# Find work
bd ready --json                    # Unblocked issues

# Create and manage
bd create "Title" -t bug|feature|task -p 0-4 --json
bd create "Subtask" --parent <epic-id> --json  # Hierarchical subtask
bd update <id> --status in_progress --json
bd close <id> --reason "Done" --json

# Search
bd list --status open --priority 1 --json
bd show <id> --json
```

### Workflow

1. **Check ready work**: `bd ready --json`
2. **Claim task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** `bd create "Found bug" -p 1 --deps discovered-from:<parent-id> --json`
5. **Complete**: `bd close <id> --reason "Done" --json`

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

## Project Structure

```
docdiff/
├── cmd/docdiff/         # CLI entry point
├── internal/
│   ├── commands/        # Cobra commands
│   ├── config/          # Configuration handling
│   ├── filetype/        # File type detection
│   ├── git/             # Git operations
│   ├── language/        # Language strategies
│   ├── metadata/        # Doc version metadata
│   ├── report/          # Output formatters
│   └── scanner/         # Annotation scanner
└── .beads/
    └── issues.jsonl     # Git-synced issue storage
```

## CLI Help

Run `bd <command> --help` to see all available flags for any command.

## Important Rules

- Use bd for ALL task tracking
- Always use `--json` flag for programmatic use
- Do NOT create markdown TODO lists
- Do NOT commit `.beads/beads.db` (JSONL only)

---

**For detailed workflows and advanced features, see [AGENTS.md](../AGENTS.md)**
