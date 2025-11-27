# AGENTS.md

This document provides guidance for AI coding assistants working with the `sesh` project.

## Project Overview

**sesh** is a modern git workspace and session manager written in Go. It helps developers quickly switch between different project contexts by managing git worktrees and terminal multiplexer sessions (tmux, zellij) in a unified way. The tool uses bare repositories with git worktrees to allow simultaneous work on multiple branches without stashing or switching.

## Core Technologies

- **Language**: Go 1.24
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra) - for command-line interface. cobra-cli should be used for scaffolding new commands.
- **Error Handling**: [eris](https://github.com/rotisserie/eris) - for rich error handling with stack traces
- **Database**: SQLite - for persistent application state
- **Build System**: Nix flakes - for reproducible development environment and packaging
- **Task Runner**: go-task (Taskfile) - for common development tasks
- **Session Backends**: tmux, zellij - terminal multiplexers for session management

## Project Structure

```
sesh/
├── cmd/              # CLI command definitions (Cobra commands)
├── internal/         # Private application code
│   ├── config/      # Configuration management
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

### Console output

ALWAYS log user facing messages to stderr. Only log to stdout when the output a "result" that may be piped to other commands.

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

Keep migrations in `internal/db/migrations/` and use a simple migration system. The database tracks:
- Projects (name, remote_url, local_path, created_at, last_fetched)
- Worktrees (project_id, branch, path, is_main, created_at, last_used)
- Sessions (worktree_id, tmux_session_name, created_at, last_attached)
- Session History (session_name, project_name, branch, accessed_at) - for the pop command

**Important:** The session history table is actively used by the switch and pop commands. Session history is recorded automatically when switching sessions and is used by the pop command to navigate back to previous sessions.

## Development Environment

### Dependencies

External dependencies (non-Go) are managed via `flake.nix`. The development shell includes:

- **Build tools**: go 1.24, go-task, cobra-cli
- **Code quality**: gopls, gofumpt, golines
- **Session managers**: tmux, zellij (for testing and development)
- **Utilities**: git, fzf, jq, yq, tree, shellcheck

When adding new external tools:

1. Add to the `packages` list in `flake.nix`:
```nix
devShells.default = pkgs.mkShell {
  packages = [
    pkgs.go_1_24
    pkgs.tmux
    pkgs.zellij
    # Add new tools here
  ];
};
```

2. Run `nix flake update` if needed
3. For runtime dependencies of the built package, update the `postInstall` wrapper in the `packages.default` section

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

### Session History Tracking

Session history is tracked automatically in the database to support the pop command:

```go
import "github.com/benoctopus/sesh/internal/db"
import "github.com/benoctopus/sesh/internal/config"

// Recording session history (done automatically in switch command)
func recordSessionHistory(sessionName, projectName, branch string) {
    dbPath, _ := config.GetDBPath()
    config.EnsureConfigDir()
    database, _ := db.InitDB(dbPath)
    defer database.Close()
    db.AddSessionHistory(database, sessionName, projectName, branch)
}

// Retrieving previous session (used in pop command)
func getPreviousSession() {
    dbPath, _ := config.GetDBPath()
    database, _ := db.InitDB(dbPath)
    defer database.Close()

    currentSession, _ := sessionMgr.GetCurrentSessionName()
    previousSession, err := db.GetPreviousSession(database, currentSession)
    // previousSession contains session_name, project_name, branch
}
```

**Pattern Notes:**
- Session history recording is best-effort (errors don't fail the command)
- The database is initialized on-demand when needed
- GetPreviousSession excludes the current session to prevent switching to self

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

### Session Manager Integration

The application supports multiple session backends (tmux, zellij). Execute session manager commands using `os/exec`:

```go
// Example: tmux
cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
if err := cmd.Run(); err != nil {
    return eris.Wrapf(err, "failed to create tmux session %s", sessionName)
}

// Example: zellij
cmd := exec.Command("zellij", "attach", sessionName)
if err := cmd.Run(); err != nil {
    return eris.Wrapf(err, "failed to attach to zellij session %s", sessionName)
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
6. **Use `task` for builds and use as a go tool** - Always use `go tool task build`, `go tool task test`, etc. instead of direct `go` commands
7. **Binary output to `dist/`** - All built binaries must go to the `dist/` directory
8. **Keep it simple** - This is a CLI tool, not a web service. Avoid over-engineering.
9. **Leverage existing tools** - Use fzf or peco for interactive selection
10. **Bare repository structure** - All projects use bare repositories with worktrees, not regular clones
11. **Worktree tracking** - Recent fixes ensure proper upstream tracking for worktree branches
12. **Multiple session backends** - Support tmux and zellij, with auto-detection capability
13. **Tree output format** - The list command now outputs in tree format for better visualization
14. Update this document when new patterns or conventions are established.

## Recent Updates (Context for Development)

- **Session history tracking**: Added session history database to track session switches for the pop command
- **Pop command**: New command to switch back to previous sessions (aliases: p, back)
- **Database usage**: Now actively uses SQLite database for session history tracking
- **Nix packaging**: The project is now packaged as a Nix flake and can be installed via `nix profile add`
- **Worktree improvements**: Fixed upstream tracking and remote branch configuration for bare repositories
- **Session manager support**: Added zellij support alongside tmux in the development shell
- **List visualization**: Changed from table format to tree format for better project/worktree hierarchy display
- **CI integration**: Added Nix flake build checks to GitHub Actions workflow

## Task Runner

This project uses [go-task](https://taskfile.dev) for common development tasks. All build, test, and development commands should be executed via `go tool task`.

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
- It is ok to use //nolint: comments to ignore linting errors for error checking where the error is not important or part of a defer statement
- Please remember to check lint/test before determining a task that modifies go code is complete
