# Configuration Management

The `sesh edit` command provides an interactive way to edit the sesh configuration file with built-in validation.

## Overview

The configuration file is located at:
- **Linux/macOS**: `~/.config/sesh/config.toml`
- **Windows**: `%APPDATA%\sesh\config.toml`

The configuration file uses the TOML format and supports the following settings.

## Usage

### Edit Configuration

```bash
# Open config in your editor
sesh edit
```

The command will:
1. Create the config file with default values if it doesn't exist
2. Open it in your preferred editor (`$VISUAL`, `$EDITOR`, or `vi`)
3. Detect if you made changes (using SHA256 hash comparison)
4. Validate the configuration after editing
5. Prompt you to fix errors if validation fails

### Interactive Validation

If the config has errors after editing, you'll be prompted with three options:

1. **Edit again** - Re-open the editor to fix the errors
2. **Discard changes** - Exit and manually fix or delete the config file
3. **Keep invalid config** - Save anyway (not recommended, may cause issues)

## Configuration Schema

### Example Configuration

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

### Settings Reference

#### `[workspace]` Section

##### `projects_dir`
- **Type**: String (path)
- **Default**: `~/.sesh/projects`
- **Description**: Directory where git repositories are cloned
- **Example**: `projects_dir = "~/Code/projects"`

##### `worktrees_dir`
- **Type**: String (path)
- **Default**: `~/.sesh/worktrees`
- **Description**: Directory where git worktrees are created
- **Example**: `worktrees_dir = "~/Code/worktrees"`

Both paths support tilde (`~`) expansion for the user's home directory.

#### `[session]` Section

##### `backend`
- **Type**: String
- **Default**: `tmux`
- **Valid values**:
  - `tmux` - Use tmux for session management
  - `code`, `code:open`, `code:workspace`, `code:replace` - VS Code variants
  - `cursor`, `cursor:open`, `cursor:workspace`, `cursor:replace` - Cursor variants
- **Description**: Session backend to use for managing sessions

**Backend Modes**:
- `code` or `cursor` (or `:open`) - Open project in new window
- `:workspace` - Add project to workspace
- `:replace` - Replace current window

##### `startup_command`
- **Type**: String (optional)
- **Default**: Empty string
- **Description**: Command to run when creating a new session
- **Example**: `startup_command = "direnv allow"`

#### `[picker]` Section

##### `finder`
- **Type**: String
- **Default**: `auto`
- **Valid values**:
  - `auto` - Auto-detect available fuzzy finder
  - `fzf` - Use fzf (if installed)
  - `skim` - Use skim (Rust-based fuzzy finder)
- **Description**: Fuzzy finder for interactive selection

## Validation

The configuration is validated for:
- Valid backend names
- Valid finder names
- Path expandability (tilde expansion must work)

If validation fails, you'll see a clear error message explaining what's wrong.

## Environment Variable Override

For testing purposes, you can override the config directory:

```bash
sesh --config-dir /tmp/test-config edit
```

This is primarily used in automated testing and development.

## Related Commands

- `sesh list` - List projects, worktrees, or sessions
- `sesh switch` - Switch to a different branch/project
- `sesh clone` - Clone a new repository

## Implementation Details

For developers working on sesh, see:
- [`src/config/README.md`](../src/config/README.md) - Internal config module documentation
- [`src/cli/edit.rs`](../src/cli/edit.rs) - Edit command implementation
- [`tests/config_validation.rs`](../tests/config_validation.rs) - Config validation tests

