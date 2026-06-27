# GitHub Copilot Instructions for docdiff

## Project Overview

**docdiff** is a language-agnostic CLI tool that detects stale documentation by tracking `@doc` annotations in source code comments.

**Key Features:**
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
└── docs/
    └── .doc-versions.json  # Tracked doc commit hashes
```
