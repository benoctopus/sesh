# Copilot Instructions for sesh

This repository contains **sesh**, a git workspace and tmux session manager written in Go.

## Project Overview

Sesh helps developers quickly switch between different project contexts by managing both git repositories and tmux sessions in a unified way. It's a CLI tool that stores session state in SQLite and integrates with tmux for session management.

## Tech Stack

- **Language**: Go 1.25.2
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) for command-line interface
- **Error Handling**: [eris](https://github.com/rotisserie/eris) for rich error handling with stack traces
- **Database**: SQLite (driver: modernc.org/sqlite) for persistent application state
- **Build System**: Nix flakes for reproducible development environment
- **Task Runner**: go-task (Taskfile) for common development tasks

## Critical Conventions

### Error Handling (REQUIRED)

**ALWAYS** use the `eris` package for error handling. Never use `fmt.Errorf` or plain errors.

```go
import "github.com/rotisserie/eris"

// Wrapping errors
if err != nil {
    return eris.Wrap(err, "failed to create session")
}

// Creating new errors
return eris.New("session not found")

// Wrapping with formatted message
return eris.Wrapf(err, "failed to connect to workspace %s", name)
```

### Database

- Use `modernc.org/sqlite` as the SQLite driver with blank import: `import _ "modernc.org/sqlite"`
- Enable foreign key constraints: `PRAGMA foreign_keys = ON`
- Use transactions for migrations
- Database location: `{OS_CONFIG_DIR}/sesh/sesh.db` (use `os.UserConfigDir()`)
- Keep migrations in `internal/db/migrations/`

### Build and Test Commands

**ALWAYS** use `task` commands instead of direct `go` commands:
- Build: `task build` (outputs to `dist/`)
- Test: `task test`
- Lint: `task lint`
- Format: `task fmt`
- All checks: `task check`

### CLI Structure

- Use Cobra for all command-line interface work
- Use `task cobra:add -- commandName` to scaffold new commands
- Commands are in `cmd/` directory
- Follow existing command patterns (root.go, session.go, workspace.go)

## Code Style

- Follow standard Go conventions (gofmt, golangci-lint)
- Use `gofumpt` for stricter formatting
- Write tests alongside code (`*_test.go`)
- Use table-driven tests where appropriate
- Document public functions and types with comments
- Keep functions small and focused (single responsibility principle)
- Use conventional commit messages
- Split functionality into separate packages: `internal/{}`, `pkg/{}`, etc.
- Avoid creating overgeneralized "util" packages
- Use interfaces for extensibility

## Project Structure

```
sesh/
├── cmd/              # CLI command definitions (Cobra commands)
├── internal/         # Private application code
│   ├── config/      # Configuration management (env vars > config file > defaults)
│   ├── db/          # Database layer (SQLite)
│   ├── session/     # Tmux session management
│   └── workspace/   # Git workspace management
├── dist/            # Build output directory (gitignored)
├── flake.nix        # Nix flake for development environment
├── go.mod           # Go module definition
├── main.go          # Application entry point
└── Taskfile.yaml    # Task definitions
```

## Common Patterns

### Configuration Loading

```go
// Configuration uses three-tier hierarchy: environment variables (highest), config file, defaults (lowest)
configDir, err := os.UserConfigDir()
if err != nil {
    return eris.Wrap(err, "failed to get config directory")
}
seshDir := filepath.Join(configDir, "sesh")
```

### Database Connection

```go
import "database/sql"
import _ "modernc.org/sqlite"

db, err := sql.Open("sqlite", dbPath)
if err != nil {
    return eris.Wrap(err, "failed to open database")
}
```

### Tmux Integration

```go
cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
if err := cmd.Run(); err != nil {
    return eris.Wrapf(err, "failed to create tmux session %s", sessionName)
}
```

## Dependency Management

- **External (non-Go) dependencies**: Add to `flake.nix` in the `packages` list
- **Go dependencies**: Use standard `go get` and `go mod tidy`

## Important Guidelines

1. **State persistence**: Use SQLite only, not JSON files or other formats
2. **Binary output**: All built binaries go to `dist/` directory
3. **Config directory**: Use `os.UserConfigDir()` + `sesh` subdirectory
4. **Keep it simple**: This is a CLI tool, avoid over-engineering
5. **Minimal changes**: Make surgical, precise modifications
6. **Test changes**: Write tests consistent with existing test patterns
7. **Interactive tools**: Leverage fzf or peco for interactive selection if needed

## Testing

- Write unit tests for business logic
- Use integration tests for database operations
- Mock external dependencies (tmux, git) where appropriate
- Test files should be in the same package as the code they test

## Additional Context

For more detailed information, see the `AGENTS.md` file in the repository root, which contains comprehensive guidance for AI coding assistants working with this project.
