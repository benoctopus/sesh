# AGENTS.md

This document provides guidance for AI coding assistants working with the `sesh` project.

## Project Overview

**sesh** is a git workspace and tmux session manager written in Go. It helps developers quickly switch between different project contexts by managing both git repositories and tmux sessions in a unified way.

## Core Technologies

- **Language**: Go 1.24
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) - for command-line interface. cobra-cli should be used for scaffolding new commands.
- **Error Handling**: [eris](https://github.com/rotisserie/eris) - for rich error handling with stack traces
- **Database**: SQLite - for persistent application state
- **Build System**: Nix flakes - for reproducible development environment
- **Task Runner**: go-task (Taskfile) - for common development tasks

## Project Structure

```
sesh/
├── cmd/              # CLI command definitions (Cobra commands)
├── internal/         # Private application code
│   ├── config/      # Configuration management
│   ├── db/          # Database layer (SQLite)
│   ├── session/     # Tmux session management
│   ├── workspace/   # Git workspace management
│   └── ...
├── pkg/             # Public libraries (if any)
├── dist/            # Build output directory (gitignored)
├── flake.nix        # Nix flake for development environment
├── go.mod           # Go module definition
├── main.go          # Application entry point
└── Taskfile.yaml    # Task definitions (go-task)
```

## Key Conventions

### Error Handling

**ALWAYS** use the `eris` package for error handling:

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

### State Management

Application state is stored in an SQLite database located in:
```
{OS_CONFIG_DIR}/sesh/sesh.db
```

Where `OS_CONFIG_DIR` is:
- Linux: `$XDG_CONFIG_HOME` or `~/.config`
- macOS: `~/Library/Application Support`
- Windows: `%APPDATA%`

Use Go's `os.UserConfigDir()` to get the appropriate directory.

### CLI Structure

Commands follow this pattern using Cobra:

```go
// cmd/root.go - root command
// cmd/session.go - session subcommands
// cmd/workspace.go - workspace subcommands
```

Example command structure:
```
sesh list                    # List all sessions
sesh attach <name>          # Attach to a session
sesh create <name> <path>   # Create new session
sesh delete <name>          # Delete a session
```

### Database Schema

Keep migrations in `internal/db/migrations/` and use a simple migration system. The database should track:
- Sessions (name, tmux session name, workspace path, created_at, last_used)
- Workspaces (path, git remote, branch, tags)
- Session history and metadata

## Development Environment

### Dependencies

External dependencies (non-Go) are managed via `flake.nix`. When adding new external tools:

1. Add to the `packages` list in `flake.nix`:
```nix
devShells.default = pkgs.mkShell {
  packages = [
    pkgs.go_1_25
    pkgs.tmux          # Add new tools here
    # ...
  ];
};
```

2. Run `nix flake update` if needed

### Go Dependencies

Manage Go dependencies normally with:
```bash
go get github.com/package/name
go mod tidy
```

## Code Style

- Follow standard Go conventions (gofmt, golangci-lint)
- Use `gofumpt` for stricter formatting (available in flake.nix)
- Keep line length reasonable using `golines` (available in flake.nix)
- Write tests alongside code (`*_test.go`)
- Use table-driven tests where appropriate
- Write commits for incremental changes. Use conventional commit messages.
- Document public functions and types with comments.
- Keep functions small and focused (single responsibility principle).
- Extract reusable code into helper functions and packages.
- Split functionality into separate packages using go conventions, e.g. `internal/{}`, `pkg/{}`, etc.
- Avoid creating overgeneralized "util" packages.
- Use interfaces to allow future extensibility with different session or workspace backends.

## Common Patterns

### Configuration Loading

```go
// Load config from OS config directory
configDir, err := os.UserConfigDir()
if err != nil {
    return eris.Wrap(err, "failed to get config directory")
}

seshDir := filepath.Join(configDir, "sesh")
```

### Database Connection

```go
import "database/sql"
import _ "modernc.org/sqlite"  // or github.com/mattn/go-sqlite3

db, err := sql.Open("sqlite", dbPath)
if err != nil {
    return eris.Wrap(err, "failed to open database")
}
```

### Tmux Integration

Execute tmux commands using `os/exec`:

```go
cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
if err := cmd.Run(); err != nil {
    return eris.Wrapf(err, "failed to create tmux session %s", sessionName)
}
```

## Testing

- Write unit tests for business logic
- Use integration tests for database operations
- Mock external dependencies (tmux, git) where appropriate
- Test files should be in the same package as the code they test

## Important Notes for AI Assistants

1. **Always use eris for errors** - Never use `fmt.Errorf` or plain errors
2. **State goes in SQLite** - Don't use JSON files or other formats for persistence
3. **External tools via Nix** - Add non-Go dependencies to `flake.nix`
4. **Cobra for CLI** - Don't create custom command parsing
5. **Config directory** - Use `os.UserConfigDir()` + `sesh` subdirectory
6. **Use `task` for builds** - Always use `task build`, `task test`, etc. instead of direct `go` commands
7. **Binary output to `dist/`** - All built binaries must go to the `dist/` directory
8. **Keep it simple** - This is a CLI tool, not a web service. Avoid over-engineering.
9. Leverage existing tools like fzf or peco for interactive selection if needed.
10. Update this document when new patterns or conventions are established.

## Task Runner

This project uses [go-task](https://taskfile.dev) for common development tasks. All build, test, and development commands should be executed via `task`.

### Common Tasks

```bash
# List all available tasks
task

# Build the binary (output to dist/)
task build

# Run tests
task test

# Run tests with coverage
task test:coverage

# Run linter
task lint

# Format code
task fmt

# Run all checks (format check, lint, test)
task check

# Clean build artifacts
task clean

# Run the application with arguments
task run -- --help
task run -- list

# Add a new cobra command
task cobra:add -- commandName

# Install binary to $GOPATH/bin
task install
```

### Important Task Usage Rules

1. **Always use `task build`** instead of `go build` - ensures output goes to `dist/`
2. **Use `task check`** before commits - runs format check, linter, and tests
3. **Use `task cobra:add`** to scaffold new commands - maintains consistency
4. **Binary output location** - all built binaries go to `dist/` directory

## Build and Run

```bash
# Build (creates dist/sesh)
task build

# Run
task run -- --help

# Development with Nix
nix develop

# Quick build and run
task build && ./dist/sesh --help
```

## Resources

- [Cobra Documentation](https://cobra.dev/)
- [Eris Documentation](https://github.com/rotisserie/eris)
- [SQLite in Go](https://github.com/mattn/go-sqlite3)
- [Tmux Manual](https://man.openbsd.org/tmux)
