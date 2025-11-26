package session

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rotisserie/eris"
)

// ZellijManager implements the SessionManager interface for zellij
type ZellijManager struct{}

// NewZellijManager creates a new ZellijManager
func NewZellijManager() *ZellijManager {
	return &ZellijManager{}
}

// Create creates a new zellij session with the given name at the specified path
func (z *ZellijManager) Create(name, path string) error {
	// Check if session already exists
	exists, err := z.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		return eris.Errorf("session '%s' already exists", name)
	}

	// Create detached session at the specified path
	// Zellij doesn't have a direct "detached" mode like tmux, so we run it in background
	// We use a shell command that creates the session and exits
	// Using 'zellij attach <name> --create' with shell backgrounding
	shellScript := `cd "` + path + `" && (setsid zellij --session "` + name + `" > /dev/null 2>&1 &)`
	cmd := exec.Command("sh", "-c", shellScript)

	if err := cmd.Run(); err != nil {
		return eris.Wrapf(err, "failed to create zellij session")
	}

	// Wait a moment for the session to be created
	// This is necessary because zellij session creation is asynchronous
	cmd = exec.Command("sleep", "0.5")
	_ = cmd.Run()

	return nil
}

// Attach attaches to an existing zellij session
// This replaces the current process with zellij attach
func (z *ZellijManager) Attach(name string) error {
	// Check if session exists
	exists, err := z.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// If we're already inside zellij, we can't nest sessions
	// Instead, we should switch sessions or inform the user
	if z.IsInsideSession() {
		return z.Switch(name)
	}

	// Use exec to replace current process with zellij attach
	// This ensures we don't nest processes
	zellijPath, err := exec.LookPath("zellij")
	if err != nil {
		return eris.Wrap(err, "zellij not found in PATH")
	}

	err = syscall.Exec(zellijPath, []string{"zellij", "attach", name}, os.Environ())
	if err != nil {
		return eris.Wrap(err, "failed to exec zellij attach")
	}

	return nil
}

// Switch switches to a different zellij session (when already inside zellij)
func (z *ZellijManager) Switch(name string) error {
	// Check if session exists
	exists, err := z.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// Check if we're inside zellij
	if !z.IsInsideSession() {
		return eris.New("not inside a zellij session, use Attach instead")
	}

	// Zellij requires using action to switch sessions
	// We need to use zellij action switch-session
	cmd := exec.Command("zellij", "action", "switch-session", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to switch zellij session: %s", string(output))
	}

	return nil
}

// List returns all active zellij session names
func (z *ZellijManager) List() ([]string, error) {
	cmd := exec.Command("zellij", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		// If no sessions exist, zellij might return an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, eris.Wrap(err, "failed to list zellij sessions")
	}

	return parseZellijList(string(output)), nil
}

// Delete kills a zellij session
func (z *ZellijManager) Delete(name string) error {
	// Check if session exists
	exists, err := z.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	cmd := exec.Command("zellij", "delete-session", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to delete zellij session: %s", string(output))
	}

	return nil
}

// Exists checks if a zellij session exists
func (z *ZellijManager) Exists(name string) (bool, error) {
	sessions, err := z.List()
	if err != nil {
		return false, err
	}

	for _, session := range sessions {
		if session == name {
			return true, nil
		}
	}

	return false, nil
}

// IsRunning checks if zellij is available
func (z *ZellijManager) IsRunning() (bool, error) {
	// Check if zellij is installed
	if !isCommandAvailable("zellij") {
		return false, nil
	}

	// Try to list sessions - if this succeeds, zellij is available
	_, err := z.List()
	if err != nil {
		return false, nil
	}

	return true, nil
}

// Name returns the backend name
func (z *ZellijManager) Name() string {
	return string(BackendZellij)
}

// IsInsideSession checks if currently inside a zellij session
func (z *ZellijManager) IsInsideSession() bool {
	return IsInsideZellij()
}

// parseZellijList parses the output of zellij list-sessions
// Zellij list-sessions output format is typically one session per line
func parseZellijList(output string) []string {
	var sessions []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "ACTIVE") {
			// Extract just the session name
			// Zellij output might be formatted, so we take the first word
			parts := strings.Fields(line)
			if len(parts) > 0 {
				sessions = append(sessions, parts[0])
			}
		}
	}
	return sessions
}

// GetCurrentSessionName returns the name of the current zellij session
// Returns empty string if not inside a session
func (z *ZellijManager) GetCurrentSessionName() (string, error) {
	if !z.IsInsideSession() {
		return "", nil
	}

	// Zellij stores session name in ZELLIJ_SESSION_NAME environment variable
	sessionName := os.Getenv("ZELLIJ_SESSION_NAME")
	if sessionName != "" {
		return sessionName, nil
	}

	// Fallback: try to get it from zellij action
	cmd := exec.Command("zellij", "action", "query-tab-names")
	output, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get current session name")
	}

	// This is a best-effort approach; zellij's API for this is limited
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}

	return "", nil
}

// CreateOrAttach creates a session if it doesn't exist, or attaches to it if it does
func (z *ZellijManager) CreateOrAttach(name, path string) error {
	exists, err := z.Exists(name)
	if err != nil {
		return err
	}

	if exists {
		return z.Attach(name)
	}

	if err := z.Create(name, path); err != nil {
		return err
	}

	return z.Attach(name)
}

// SendKeys sends commands to a zellij session
func (z *ZellijManager) SendKeys(name, command string) error {
	// Check if session exists
	exists, err := z.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// Zellij doesn't have a direct equivalent to tmux send-keys
	// We can use zellij action write to send text to the current pane
	// Note: This is a limitation compared to tmux
	cmd := exec.Command("zellij", "action", "write", "27", command) // 27 is the escape key code
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to send keys to zellij session: %s", string(output))
	}

	return nil
}
