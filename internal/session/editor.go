package session

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rotisserie/eris"
)

// EditorMode represents the mode of operation for editor backends
type EditorMode string

const (
	EditorModeOpen      EditorMode = "open"      // Opens in a new window
	EditorModeWorkspace EditorMode = "workspace" // Adds to current workspace
	EditorModeReplace   EditorMode = "replace"   // Replaces current window
)

// EditorManager implements the SessionManager interface for VS Code and Cursor editors
type EditorManager struct {
	command string     // "code" or "cursor"
	mode    EditorMode // open, workspace, replace
}

// NewEditorManager creates a new EditorManager for the specified editor command and mode
func NewEditorManager(command string, mode EditorMode) *EditorManager {
	return &EditorManager{
		command: command,
		mode:    mode,
	}
}

// Create opens the specified path in the editor
// For editor backends, 'name' is ignored - we only use the path
func (e *EditorManager) Create(name, path string) error {
	return e.openPath(path)
}

// Attach opens the worktree path in the editor
// For editor backends, this behaves the same as Create
func (e *EditorManager) Attach(name string) error {
	// For editor backends, we need the path, not just the name
	// This is typically called after Create, but for editors we can't
	// attach to a "session" - we'd need the path again
	fmt.Fprintf(os.Stderr, "attach is not supported with the %s backend\n", e.Name())
	return nil
}

// Switch opens the worktree path in the editor
// For editor backends, this behaves the same as Create
func (e *EditorManager) Switch(name string) error {
	// For editor backends, we need the path, not just the name
	fmt.Fprintf(os.Stderr, "switch is not supported with the %s backend\n", e.Name())
	return nil
}

// List returns an error as editor backends don't support session listing
func (e *EditorManager) List() ([]string, error) {
	return nil, eris.Errorf("listing sessions is not supported with the %s backend", e.Name())
}

// Delete returns an error as editor backends don't support session deletion
func (e *EditorManager) Delete(name string) error {
	return eris.Errorf("deleting sessions is not supported with the %s backend", e.Name())
}

// Exists always returns false as editor backends can't track sessions
func (e *EditorManager) Exists(name string) (bool, error) {
	return false, nil
}

// IsRunning checks if the editor CLI command is available in PATH
func (e *EditorManager) IsRunning() (bool, error) {
	return isCommandAvailable(e.command), nil
}

// Name returns the full backend name (e.g., "code:open", "cursor:replace")
func (e *EditorManager) Name() string {
	return fmt.Sprintf("%s:%s", e.command, e.mode)
}

// IsInsideSession always returns false as editors don't have a session concept
func (e *EditorManager) IsInsideSession() bool {
	return false
}

// GetCurrentSessionName returns an error as editor backends don't have sessions
func (e *EditorManager) GetCurrentSessionName() (string, error) {
	return "", eris.Errorf("getting current session is not supported with the %s backend", e.Name())
}

// openPath opens the given path in the editor using the configured mode
func (e *EditorManager) openPath(path string) error {
	args := e.buildArgs(path)

	cmd := exec.Command(e.command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to open %s in %s: %s", path, e.Name(), string(output))
	}

	return nil
}

// buildArgs constructs the command line arguments based on the editor mode
func (e *EditorManager) buildArgs(path string) []string {
	switch e.mode {
	case EditorModeWorkspace:
		return []string{"--add", path}
	case EditorModeReplace:
		return []string{"-r", path}
	case EditorModeOpen:
		fallthrough
	default:
		return []string{path}
	}
}

// ParseEditorBackend parses a backend string like "code:open" or "cursor:replace"
// and returns the command and mode, or an error if invalid
func ParseEditorBackend(backend string) (command string, mode EditorMode, err error) {
	// Map of valid editor backends
	validBackends := map[string]struct {
		command string
		mode    EditorMode
	}{
		"code:open":        {"code", EditorModeOpen},
		"code:workspace":   {"code", EditorModeWorkspace},
		"code:replace":     {"code", EditorModeReplace},
		"cursor:open":      {"cursor", EditorModeOpen},
		"cursor:workspace": {"cursor", EditorModeWorkspace},
		"cursor:replace":   {"cursor", EditorModeReplace},
	}

	if cfg, ok := validBackends[backend]; ok {
		return cfg.command, cfg.mode, nil
	}

	return "", "", eris.Errorf("invalid editor backend: %s", backend)
}

// IsEditorBackend checks if the given backend string is an editor backend
func IsEditorBackend(backend string) bool {
	_, _, err := ParseEditorBackend(backend)
	return err == nil
}
