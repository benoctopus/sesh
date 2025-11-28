# sesh

A modern git workspace and session manager that seamlessly integrates git worktrees with terminal multiplexers (tmux, zellij, etc.).

## Features

- 🚀 **Fast branch switching** - Instant switching between branches with fuzzy finding
- 📁 **Git worktree management** - Automatic creation and management of git worktrees
- 🖥️ **Session management** - Integrated tmux/zellij session management
- 🎯 **Project auto-detection** - Automatically detects your current project
- 🔧 **Startup commands** - Run custom commands when creating sessions
- 🎨 **Beautiful UI** - Colored output and formatted tables
- ⚡ **Zero configuration** - Works out of the box with sensible defaults

## Why sesh?

Traditional git workflows require constantly switching branches, stashing changes, and managing multiple terminal sessions. **sesh** eliminates this friction by:

1. Using git worktrees to keep each branch in a separate directory
2. Creating dedicated terminal sessions for each worktree
3. Providing instant switching between branches with fuzzy finding
4. Managing everything from a centralized workspace directory

## Installation

### Using Nix (Recommended)

The recommended way to install sesh is via Nix, which automatically handles all dependencies:

```bash
# Install to your profile
nix profile add github:benoctopus/sesh

# Or run directly without installing
nix run github:benoctopus/sesh
```

The Nix package automatically includes git and fzf as runtime dependencies.

### Using Go

```bash
go install github.com/benoctopus/sesh@latest
```

**Note:** When installing via `go install`, you'll need to ensure the following dependencies are installed:

**Required:**
- Git (version 2.30 or later recommended for worktree support)

**Optional (but recommended):**
- A terminal multiplexer: `tmux` or `zellij`
- A fuzzy finder: `fzf` or `peco` (for interactive branch selection)

### From Source

```bash
git clone https://github.com/benoctopus/sesh.git
cd sesh
go build -o sesh .
mv sesh /usr/local/bin/
```

Same dependency requirements apply as with `go install`.

## Quick Start

### 1. Clone a repository

```bash
sesh clone git@github.com:user/repo.git
```

This will:
- Clone the repository as a bare repo in `~/.sesh/github.com/user/repo/`
- Create a worktree for the default branch
- Create and attach to a tmux session

### 2. Switch between branches

```bash
# Interactive fuzzy search for branches
sesh switch

# Switch to an existing branch
sesh switch feature-branch

# Create a new branch automatically (if it doesn't exist)
sesh switch new-feature

# Interactive project and session selection
sesh switch -p ""
```

### 3. List all sessions

```bash
# List all sessions
sesh list

# List only projects
sesh list --projects

# Output as JSON
sesh list --json
```

### 4. Pop back to previous session

```bash
# Switch back to the previous session
sesh pop

# Or use the short aliases
sesh p
sesh back
```

## Usage

### Commands

#### `sesh clone <remote-url>`

Clone a git repository into the workspace folder.

```bash
sesh clone git@github.com:user/repo.git
sesh clone https://github.com/user/repo.git
```

#### `sesh switch [branch]`

Switch to a branch, creating a worktree and session if they don't exist.

If the branch doesn't exist locally or remotely, it will be created automatically.

```bash
# Interactive fuzzy branch selection
sesh switch

# Switch to existing branch
sesh switch main

# Create new branch automatically
sesh switch feature-foo

# Specify project explicitly
sesh switch --project myproject feature-bar

# Interactive project and session selection
sesh switch -p ""

# Run a startup command
sesh switch -c "direnv allow" feature-baz
```

**Interactive Project & Session Selection:**

When you use `sesh switch -p ""` (the `-p` flag without a value), sesh provides a two-step interactive selection process:

1. First, select a project from all available projects using fuzzy finding
2. Then, select a session from that project's active sessions
3. Finally, attach to the selected session

This is useful when you want to quickly jump between projects and sessions without knowing the exact names.

#### `sesh list`

List all projects, worktrees, and sessions.

```bash
# List all sessions (default)
sesh list

# List only projects
sesh list --projects

# Output in JSON format
sesh list --json

# Filter to sessions for current project only
sesh list --current-project

# Show only running sessions
sesh list --running

# Output session names only (useful for piping to fzf)
sesh list --plain
```

#### `sesh delete [branch]`

Delete a worktree and its associated session.

```bash
# Delete specific worktree
sesh delete feature-foo

# Delete entire project
sesh delete --all
```

#### `sesh pop`

Switch to the previous session in history.

The pop command (aliases: `p`, `back`) switches back to the last session you were working on, using the session history stack. This is useful for quickly toggling between two sessions.

```bash
# Switch to previous session
sesh pop

# Short aliases
sesh p
sesh back
```

**Note:** Session history is automatically tracked when you switch sessions. The pop command will fail if there's no previous session in the history.

#### `sesh status`

Show current session and project information.

```bash
sesh status
```

#### `sesh fetch [project]`

Fetch latest changes from remote.

```bash
# Fetch current project
sesh fetch

# Fetch specific project
sesh fetch myproject

# Fetch all projects
sesh fetch --all
```

#### `sesh edit`

Open the sesh configuration file in your default editor (determined by `$VISUAL` or `$EDITOR`).

The configuration is validated after editing, and you'll be prompted to fix any errors before saving.

```bash
sesh edit
```

## Configuration

sesh can be configured via a config file or environment variables.

### Config File

The config file is located at:
- **Linux/macOS**: `~/.config/sesh/config.yaml`
- **Windows**: `%APPDATA%\sesh\config.yaml`

You can edit it manually or use `sesh edit` to open it in your default editor with validation.

```yaml
version: "1"                        # Config file version (for backwards compatibility)
workspace_dir: ~/Code/workspaces    # Where to store repositories
session_backend: tmux               # tmux, zellij, screen, or auto
fuzzy_finder: fzf                   # fzf, peco, or auto
startup_command: direnv allow       # Command to run on session creation
```

**Available Options:**
- `version`: Config file format version (currently "1")
- `workspace_dir`: Directory where repositories are stored (supports `~` expansion)
- `session_backend`: Session manager to use (`tmux`, `zellij`, `screen`, or `auto` to detect)
- `fuzzy_finder`: Fuzzy finder for branch selection (`fzf`, `peco`, or `auto` to detect)
- `startup_command`: Command to run when creating new sessions

### Per-Project Configuration

Create `.sesh.yaml` in your project root:

```yaml
startup_command: |
  direnv allow
  npm install
```

### Environment Variables

```bash
export SESH_WORKSPACE=~/my-workspace
export SESH_SESSION_BACKEND=tmux
export SESH_FUZZY_FINDER=fzf
```

### Configuration Hierarchy

Configuration is resolved in the following order (highest to lowest priority):

1. **Command-line flags** - `sesh switch -c "command"`
2. **Per-project config** - `.sesh.yaml` in project root
3. **Environment variables** - `$SESH_WORKSPACE`, `$SESH_SESSION_BACKEND`, `$SESH_FUZZY_FINDER`
4. **Global config** - `~/.config/sesh/config.yaml`
5. **Defaults** - `~/.sesh` workspace, `auto` backend, `auto` fuzzy finder

## Workspace Structure

sesh organizes your projects in a centralized workspace directory:

```
~/.sesh/
├── github.com/
│   └── user/
│       └── repo/
│           ├── .git/              # Bare repository
│           ├── main/              # Main branch worktree
│           └── feature-foo/       # Feature branch worktree
└── gitlab.com/
    └── org/
        └── project/
            ├── .git/
            └── develop/
```

## Shell Completion

sesh supports shell completion for bash, zsh, fish, and powershell.

### Bash

```bash
# Load completion for current session
source <(sesh completion bash)

# Install permanently
sesh completion bash > /etc/bash_completion.d/sesh
```

### Zsh

```bash
# Load completion for current session
source <(sesh completion zsh)

# Install permanently
sesh completion zsh > "${fpath[1]}/_sesh"
```

### Fish

```bash
sesh completion fish | source

# Install permanently
sesh completion fish > ~/.config/fish/completions/sesh.fish
```

## Troubleshooting

### tmux not found

sesh requires tmux (or another session manager) to be installed:

```bash
# Ubuntu/Debian
sudo apt install tmux

# macOS
brew install tmux

# Arch Linux
sudo pacman -S tmux
```

### Project not detected

Make sure you're inside a git repository and it has a remote:

```bash
git remote -v
```

### Sessions not attaching

Check if tmux is running:

```bash
tmux ls
```

## Advanced Usage

### Startup Commands

Run commands automatically when creating sessions:

```bash
# One-time via flag
sesh switch -c "direnv allow && npm install" feature-branch

# Per-project via .sesh.yaml
echo "startup_command: direnv allow" > .sesh.yaml

# Globally via config
echo "startup_command: direnv allow" >> ~/.config/sesh/config.yaml
```

### Multiple Session Managers

sesh supports multiple session manager backends:

```yaml
# config.yaml
session_backend: tmux  # or: zellij, screen, auto, none
```

### Tmux Integration

sesh provides seamless tmux integration with convenient keybindings for quick session switching.

#### Installing Tmux Keybindings

The easiest way to set up tmux integration is using the install command:

```bash
sesh tmux install
```

This will automatically:
1. Detect your tmux configuration file location (`~/.tmux.conf` or `~/.config/tmux/tmux.conf`)
2. Add the recommended keybindings if not already present
3. Update existing keybindings if they were previously installed

After installation, reload your tmux configuration:

```bash
tmux source-file ~/.tmux.conf
```

#### Available Keybindings

Once installed, you'll have the following keybindings available:

| Keybinding | Action | Description |
|------------|--------|-------------|
| `prefix + f` | Session switcher | Opens a fuzzy finder popup to switch between branches with preview |
| `prefix + F` | PR switcher | Opens a fuzzy finder popup to switch to pull request branches with preview |
| `prefix + L` | Last session | Quickly switch to the previous session |

**Note:** `prefix` is your tmux prefix key (default: `Ctrl-b`)

#### Preview Your Keybindings

To see the keybindings without installing them:

```bash
sesh tmux keybindings
```

This outputs the keybinding configuration that you can manually copy to your `tmux.conf` if preferred.

#### Manual Installation

If you prefer to manually add keybindings to your `tmux.conf`:

```tmux
# Fuzzy session switcher with preview (prefix + f)
bind-key f display-popup -E -w 80% -h 60% \
  "/path/to/sesh switch"

# Fuzzy pull request switcher with preview (prefix + F)
bind-key F display-popup -E -w 80% -h 60% \
  "/path/to/sesh switch --pr"

# Quick switch to last/previous session (prefix + L)
bind-key L run-shell "/path/to/sesh last"
```

Replace `/path/to/sesh` with the output of `which sesh`.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [tmux](https://github.com/tmux/tmux) - Terminal multiplexer
- [git-worktree](https://git-scm.com/docs/git-worktree) - Manage multiple working trees
- [zellij](https://github.com/zellij-org/zellij) - Modern terminal workspace

## Acknowledgments

Inspired by the need for better git workflow management and the power of git worktrees.
