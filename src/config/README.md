# Config Module

This module handles sesh configuration management.

## Overview

The config module provides:
- Loading configuration from `~/.config/sesh/config.toml`
- Default configuration values when no config file exists
- Config validation to ensure valid settings
- Saving configuration to disk

## Configuration File

Location: `~/.config/sesh/config.toml` (or `$XDG_CONFIG_HOME/sesh/config.toml` on Linux)

### Structure

```toml
[workspace]
projects_dir = "~/.sesh/projects"
worktrees_dir = "~/.sesh/worktrees"

[session]
backend = "tmux"
startup_command = ""

[picker]
finder = "auto"
```

### Valid Values

#### `session.backend`
- `tmux` - Use tmux for session management (default)
- `code`, `code:open`, `code:workspace`, `code:replace` - Use VS Code
- `cursor`, `cursor:open`, `cursor:workspace`, `cursor:replace` - Use Cursor

#### `picker.finder`
- `auto` - Auto-detect available fuzzy finder (default)
- `fzf` - Use fzf
- `skim` - Use skim (Rust-based)

#### `workspace.projects_dir` and `workspace.worktrees_dir`
Any valid path string. Supports tilde (`~`) expansion.

## Usage

```rust
use sesh::config;

// Load config (returns defaults if file doesn't exist)
let config = config::load()?;

// Access config values
let projects_dir = config.projects_dir();
let backend = &config.session.backend;

// Save config
config::save(&config)?;

// Validate config
config.validate()?;
```

## Environment Variable Override

The config directory can be overridden using the `SESH_CONFIG_DIR_OVERRIDE` environment variable.
This is primarily used for testing.

## Related Documentation

- See [`docs/`](../../docs/) for user-facing documentation
- See [`src/cli/edit.rs`](../cli/edit.rs) for the `sesh edit` command implementation

