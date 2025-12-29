# Sesh - Rust Rewrite

A session manager for git worktrees, rewritten in Rust for improved performance and maintainability.

## Building

```bash
cargo build --release
```

The binary will be available at `target/release/sesh`.

## Usage

### Help
```bash
sesh --help
```

### Clone a Repository
```bash
sesh clone https://github.com/user/repo
```

### Switch to a Branch
```bash
sesh switch main
sesh switch feature-branch --project myproject
```

### List Projects/Worktrees/Sessions
```bash
sesh list                 # List all
sesh list --projects      # List projects only
sesh list --worktrees     # List worktrees only
sesh list --sessions      # List sessions only
```

### Delete
```bash
sesh delete --project myproject
sesh delete --worktree myproject:feature-branch
sesh delete --session mysession
```

### Clean Up Stale Entries
```bash
sesh clean --stale       # Remove worktrees with missing paths
sesh clean --orphaned    # Remove sessions for deleted worktrees
```

### Pop to Previous Session
```bash
sesh pop
```

### View Status
```bash
sesh status
```

### View Logs
```bash
sesh logs --lines 100
sesh logs --date 2025-01-15
```

### Generate Shell Completions
```bash
sesh completions bash > ~/.bash_completion.d/sesh
sesh completions zsh > ~/.zsh/completions/_sesh
sesh completions fish > ~/.config/fish/completions/sesh.fish
```

## Configuration

Configuration file: `~/.config/sesh/config.toml`

```toml
[workspace]
default_path = "~/workspace"

[session]
backend = "tmux"  # or "code" or "cursor"

[fuzzy]
backend = "skim"  # or "fzf"
```

## Logging

Logs are written to `~/.config/sesh/logs/sesh.log` with daily rotation.

Enable verbose terminal logging:
```bash
export SESH_LOG=debug
sesh list
```

## Database

State is stored in SQLite: `~/.config/sesh/sesh.db`

## Architecture

- **Store Layer**: SQLite database for state management
- **Git Layer**: Repository, worktree, and branch operations via libgit2
- **Backend Layer**: Pluggable session backends (tmux, VS Code, Cursor)
- **Frontend Layer**: Pluggable fuzzy finders (skim, fzf)
- **Manager Layer**: Business logic for projects, worktrees, sessions, and history

## Development

Run tests:
```bash
cargo test
```

Check for issues:
```bash
cargo clippy
```

Format code:
```bash
cargo fmt
```

## Migration from Go Version

To migrate from the Go version (not yet implemented):
```bash
sesh migrate
```

