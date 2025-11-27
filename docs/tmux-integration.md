# Deeper tmux Integration for sesh

## Overview

This document explores possibilities for deeper tmux integration with sesh, focusing on keybinding-based workflow and fuzzy search integration within tmux itself.

---

## Current State

sesh currently integrates with tmux at a basic level:
- Session creation (`tmux new-session -d -s <name> -c <path>`)
- Attachment (`tmux attach-session -t <name>`)
- Switching (`tmux switch-client -t <name>`)
- Session discovery (`tmux list-sessions`)
- Command injection (`tmux send-keys`)

**Not currently used:** `display-popup`, `display-menu`, `run-shell`, `bind-key`, custom keybindings.

---

## Integration Opportunities

### 1. Popup-Based Session/Worktree Switcher

**Concept:** Use `tmux display-popup` to show an fzf-powered selector for switching sessions/worktrees without leaving tmux.

```bash
# User presses prefix + f
bind-key f display-popup -E -w 80% -h 60% "sesh switch --popup"
```

**Implementation Options:**

#### Option A: Output-Only Mode (Minimal)
Add a flag to `sesh switch` or `sesh list` that outputs in a format suitable for piping to fzf:

```bash
# In tmux.conf
bind-key f display-popup -E "sesh list --plain | fzf --preview 'sesh preview {}' | xargs sesh switch"
```

Requires:
- `--plain` flag for machine-readable output
- Optional: `sesh preview <session>` command for fzf preview pane

#### Option B: Built-in Popup Mode
Add `--popup` flag that handles the entire fzf interaction internally:

```bash
bind-key f display-popup -E "sesh switch --popup"
```

The `--popup` mode would:
1. List all available worktrees/sessions
2. Invoke fzf with proper formatting
3. Handle selection and switching

#### Option C: Dedicated `sesh tmux` Subcommand
Create a `sesh tmux popup` command that encapsulates popup behavior:

```bash
bind-key f display-popup -E "sesh tmux popup switch"
```

---

### 2. Last Session Quick Switch

**Concept:** Quickly toggle between current and previous session.

```bash
# In tmux.conf
bind-key L run-shell "sesh last"
```

**Implementation:**
- Track last-used session in SQLite database
- `sesh last` command switches to the previous session
- Could also support `sesh last --list` to show recent sessions

---

### 3. Project-Scoped Session List

**Concept:** Show only sessions for the current project.

```bash
bind-key p display-popup -E "sesh list --current-project | fzf | xargs sesh switch"
```

**Implementation:**
- Detect current project from session name or working directory
- Filter session list to matching project prefix

---

### 4. Keybinding Configuration Helper

**Concept:** Generate recommended tmux keybindings that users can add to their config.

```bash
$ sesh tmux keybindings

# Add these to your ~/.tmux.conf:
bind-key f display-popup -E -w 80% -h 60% "sesh switch --popup"
bind-key L run-shell "sesh last"
bind-key P display-popup -E "sesh list --current-project --plain | fzf | xargs sesh switch"
bind-key C display-popup -E "sesh clone --popup"
```

Could also provide:
- `sesh tmux keybindings --install` to append to tmux.conf
- `sesh tmux keybindings --source` to output sourceable format

---

### 5. Preview Pane Integration

**Concept:** Show useful information in fzf's preview pane when selecting sessions.

```bash
# fzf with preview
sesh list --plain | fzf --preview 'sesh info {}'
```

**Preview content could include:**
- Git status summary
- Last commit message
- Branch name
- Session status (running/stopped)
- Last accessed time

**Implementation:**
- `sesh info <session-name>` command for preview data
- Or `sesh preview <session-name>` with formatted output

---

### 6. Native tmux Menu Integration

**Concept:** Use `tmux display-menu` for quick keyboard-driven selection without fzf.

```bash
# Generate menu dynamically
bind-key m run-shell 'sesh tmux menu'
```

The command would generate:
```bash
tmux display-menu -T "Sessions" \
  "main" 1 "run-shell 'sesh switch main'" \
  "feature-auth" 2 "run-shell 'sesh switch feature-auth'" \
  ...
```

**Advantages:**
- No external dependencies (no fzf required)
- Native tmux look and feel
- Single-key selection

**Disadvantages:**
- Limited to ~10 visible options
- No fuzzy search
- Less flexible than fzf

---

### 7. Async Operations with run-shell

**Concept:** Use `run-shell -b` for non-blocking operations.

```bash
# Clone in background with notification
bind-key C run-shell -b "sesh clone URL && tmux display-message 'Clone complete'"
```

**Use cases:**
- Background repository cloning
- Worktree creation without blocking
- Status updates and notifications

---

### 8. Window/Pane-Aware Operations

**Concept:** Integration with tmux windows and panes, not just sessions.

```bash
# Open worktree in new window instead of new session
sesh switch feature-branch --window

# Open in split pane
sesh switch feature-branch --split
```

**Implementation considerations:**
- Would need to track which worktrees are open as windows vs sessions
- More complex state management
- May blur the session-per-worktree model

---

## Recommended Approach

### fzf Integration Strategy: Plain Output First

**Recommendation:** Implement plain output mode as the primary interface, with optional built-in fzf mode as a convenience layer.

**Rationale:**
1. **Composability** - Plain output follows Unix philosophy; users can pipe to any tool
2. **Flexibility** - Users customize fzf options to their preference
3. **Testability** - Plain output is easy to test without mocking fzf
4. **Incremental** - Start simple, add convenience wrappers later

```bash
# Primary interface (plain output)
sesh list --plain | fzf --preview 'sesh info {}' | xargs sesh switch

# Optional convenience wrapper (added later)
sesh switch --popup  # handles fzf internally
```

### Implementation Phases

#### Phase 1: Foundation
1. Add `--plain` / `--format=plain` output to `sesh list`
2. Implement `sesh last` for quick session toggling
3. Add `sesh info <session>` command for preview content

#### Phase 2: Filtering & Discovery
1. Add `--current-project` filter to `sesh list`
2. Add `--running` / `--all` filters for session state
3. Implement session history tracking in SQLite

#### Phase 3: User Configuration
1. Add `sesh tmux keybindings` to generate config snippets
2. Create `docs/tmux-integration.md` with examples
3. Optional: `sesh tmux menu` for native menu generation

#### Phase 4: Convenience & Polish
1. Add `sesh switch --popup` built-in fzf mode
2. Enhanced preview with git status integration
3. Async operations with progress feedback

---

## Example User Workflow

After integration, a user's `tmux.conf` might include:

```bash
# Fuzzy switch between all sessions/worktrees
bind-key f display-popup -E -w 80% -h 60% \
  "sesh list --plain | fzf --reverse --preview 'sesh info {}' | xargs sesh switch"

# Quick switch to last session
bind-key L run-shell "sesh last"

# Switch within current project only
bind-key p display-popup -E -w 60% -h 40% \
  "sesh list --current-project --plain | fzf --reverse | xargs sesh switch"

# Clone new repository with popup
bind-key C command-prompt -p "Clone URL:" "display-popup -E 'sesh clone %1'"
```

---

## Technical Considerations

### tmux Version Requirements
- `display-popup` requires tmux 3.2+ (released 2021)
- Should gracefully degrade or warn on older versions

### fzf Integration
- fzf-tmux provides `--tmux` flag for native popup support
- Consider supporting both inline and popup modes

### Exit Code Handling
- Commands in popups should handle exits cleanly
- Escape should dismiss without action
- Successful selection should switch and close

### Performance
- Popup operations should be fast (<100ms to display)
- Consider caching session lists
- Preview content should be generated quickly

---

## New Commands Summary

| Command | Description |
|---------|-------------|
| `sesh list --plain` | Machine-readable output for piping |
| `sesh list --current-project` | Filter to current project sessions |
| `sesh last` | Switch to previous session |
| `sesh info <session>` | Show session details (for fzf preview) |
| `sesh tmux keybindings` | Generate recommended tmux.conf snippets |
| `sesh switch --popup` | Built-in fzf popup mode (convenience) |

## Open Questions for Future Consideration

1. Should there be full session history (beyond just "last")?
2. How important is native `display-menu` support vs fzf-based selection?
3. Should window/pane modes be supported or keep focus on sessions?
4. Should `sesh tmux keybindings --install` auto-modify tmux.conf?

## References

- [tmux display-popup docs](https://man7.org/linux/man-pages/man1/tmux.1.html)
- [joshmedeski/sesh](https://github.com/joshmedeski/sesh) - Similar Go-based session manager
- [ThePrimeagen/tmux-sessionizer](https://github.com/ThePrimeagen/tmux-sessionizer) - Bash-based approach
- [fzf-tmux integration](https://github.com/junegunn/fzf#using-with-tmux)
