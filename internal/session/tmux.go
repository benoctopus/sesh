package session

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rotisserie/eris"
)

// TmuxManager implements the SessionManager interface for tmux
type TmuxManager struct{}

// NewTmuxManager creates a new TmuxManager
func NewTmuxManager() *TmuxManager {
	return &TmuxManager{}
}

// Create creates a new tmux session with the given name at the specified path
func (t *TmuxManager) Create(name, path string) error {
	// Check if session already exists
	exists, err := t.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		return eris.Errorf("session '%s' already exists", name)
	}

	// Create detached session at the specified path
	cmd := exec.Command("tmux", "new-session", "-d", "-s", name, "-c", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to create tmux session: %s", string(output))
	}

	return nil
}

// Attach attaches to an existing tmux session
// This replaces the current process with tmux attach
func (t *TmuxManager) Attach(name string) error {
	// Check if session exists
	exists, err := t.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// If we're already inside tmux, use switch instead
	if t.IsInsideSession() {
		return t.Switch(name)
	}

	// Use exec to replace current process with tmux attach
	// This ensures we don't nest processes
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return eris.Wrap(err, "tmux not found in PATH")
	}

	err = syscall.Exec(tmuxPath, []string{"tmux", "attach-session", "-t", name}, os.Environ())
	if err != nil {
		return eris.Wrap(err, "failed to exec tmux attach")
	}

	return nil
}

// Switch switches to a different tmux session (when already inside tmux)
func (t *TmuxManager) Switch(name string) error {
	// Check if session exists
	exists, err := t.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// Check if we're inside tmux
	if !t.IsInsideSession() {
		return eris.New("not inside a tmux session, use Attach instead")
	}

	// Switch to the session
	cmd := exec.Command("tmux", "switch-client", "-t", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to switch tmux session: %s", string(output))
	}

	return nil
}

// List returns all active tmux session names
func (t *TmuxManager) List() ([]string, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// If no sessions exist, tmux returns an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, eris.Wrap(err, "failed to list tmux sessions")
	}

	return parseTmuxList(string(output)), nil
}

// Delete kills a tmux session
func (t *TmuxManager) Delete(name string) error {
	// Check if session exists
	exists, err := t.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	cmd := exec.Command("tmux", "kill-session", "-t", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to kill tmux session: %s", string(output))
	}

	return nil
}

// Exists checks if a tmux session exists
func (t *TmuxManager) Exists(name string) (bool, error) {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means session doesn't exist
			return false, nil
		}
		return false, eris.Wrap(err, "failed to check tmux session existence")
	}
	return true, nil
}

// IsRunning checks if tmux server is running
func (t *TmuxManager) IsRunning() (bool, error) {
	// Check if tmux is installed
	if !isCommandAvailable("tmux") {
		return false, nil
	}

	// Try to list sessions - if server is running, this will succeed
	cmd := exec.Command("tmux", "list-sessions")
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means no sessions, but server might still be running
			return true, nil
		}
		// Server not running
		return false, nil
	}
	return true, nil
}

// Name returns the backend name
func (t *TmuxManager) Name() string {
	return string(BackendTmux)
}

// IsInsideSession checks if currently inside a tmux session
func (t *TmuxManager) IsInsideSession() bool {
	return IsInsideTmux()
}

// parseTmuxList parses the output of tmux list-sessions
func parseTmuxList(output string) []string {
	var sessions []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions
}

// GetCurrentSession returns the name of the current tmux session
// Returns empty string if not inside a session
func (t *TmuxManager) GetCurrentSession() (string, error) {
	if !t.IsInsideSession() {
		return "", eris.New("not inside a tmux session")
	}

	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get current session name")
	}

	return strings.TrimSpace(string(output)), nil
}

// CreateOrAttach creates a session if it doesn't exist, or attaches to it if it does
func (t *TmuxManager) CreateOrAttach(name, path string) error {
	exists, err := t.Exists(name)
	if err != nil {
		return err
	}

	if exists {
		return t.Attach(name)
	}

	if err := t.Create(name, path); err != nil {
		return err
	}

	return t.Attach(name)
}

// Rename renames a tmux session
func (t *TmuxManager) Rename(oldName, newName string) error {
	// Check if old session exists
	exists, err := t.Exists(oldName)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", oldName)
	}

	// Check if new name is already taken
	exists, err = t.Exists(newName)
	if err != nil {
		return err
	}
	if exists {
		return eris.Errorf("session '%s' already exists", newName)
	}

	cmd := exec.Command("tmux", "rename-session", "-t", oldName, newName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to rename tmux session: %s", string(output))
	}

	return nil
}
