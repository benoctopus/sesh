package session

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/rotisserie/eris"
)

// ScreenManager implements the SessionManager interface for GNU Screen
type ScreenManager struct{}

// NewScreenManager creates a new ScreenManager
func NewScreenManager() *ScreenManager {
	return &ScreenManager{}
}

// Create creates a new screen session with the given name at the specified path
func (s *ScreenManager) Create(name, path string) error {
	// Check if session already exists
	exists, err := s.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		return eris.Errorf("session '%s' already exists", name)
	}

	// Create detached session at the specified path
	// screen -dmS <name> starts a detached session with the given name
	// We need to change to the directory first
	cmd := exec.Command("screen", "-dmS", name, "bash", "-c", fmt.Sprintf("cd %s && exec bash", path))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to create screen session: %s", string(output))
	}

	return nil
}

// Attach attaches to an existing screen session
// This replaces the current process with screen attach
func (s *ScreenManager) Attach(name string) error {
	// Check if session exists
	exists, err := s.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// If we're already inside screen, we cannot switch to another session
	// Screen doesn't support switching between sessions like tmux
	if s.IsInsideSession() {
		return eris.New("already inside a screen session; screen does not support switching between sessions")
	}

	// Use exec to replace current process with screen attach
	// This ensures we don't nest processes
	screenPath, err := exec.LookPath("screen")
	if err != nil {
		return eris.Wrap(err, "screen not found in PATH")
	}

	// -r attaches to the session, -d detaches it elsewhere if needed
	err = syscall.Exec(screenPath, []string{"screen", "-r", name}, os.Environ())
	if err != nil {
		return eris.Wrap(err, "failed to exec screen attach")
	}

	return nil
}

// Switch switches to a different screen session
// Note: GNU Screen does not natively support switching between sessions
// This method returns an error as screen lacks this capability
func (s *ScreenManager) Switch(name string) error {
	return eris.New("screen does not support switching between sessions; please detach and reattach")
}

// List returns all active screen session names
func (s *ScreenManager) List() ([]string, error) {
	cmd := exec.Command("screen", "-ls")
	output, err := cmd.Output()
	if err != nil {
		// screen -ls returns exit code 1 when no sessions exist
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Check if output indicates no sessions
			if strings.Contains(string(output), "No Sockets found") {
				return []string{}, nil
			}
		}
		// For other errors, still try to parse output as screen may return
		// non-zero exit codes even when sessions exist
	}

	return parseScreenList(string(output)), nil
}

// Delete kills a screen session
func (s *ScreenManager) Delete(name string) error {
	// Check if session exists
	exists, err := s.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// Send quit command to the session
	cmd := exec.Command("screen", "-S", name, "-X", "quit")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to kill screen session: %s", string(output))
	}

	return nil
}

// Exists checks if a screen session exists
func (s *ScreenManager) Exists(name string) (bool, error) {
	sessions, err := s.List()
	if err != nil {
		return false, eris.Wrap(err, "failed to check screen session existence")
	}

	for _, session := range sessions {
		if session == name {
			return true, nil
		}
	}

	return false, nil
}

// IsRunning checks if screen server is running (i.e., if any sessions exist)
func (s *ScreenManager) IsRunning() (bool, error) {
	// Check if screen is installed
	if !isCommandAvailable("screen") {
		return false, nil
	}

	// Try to list sessions - if any exist, screen is running
	sessions, err := s.List()
	if err != nil {
		return false, nil
	}

	return len(sessions) > 0, nil
}

// Name returns the backend name
func (s *ScreenManager) Name() string {
	return string(BackendScreen)
}

// IsInsideSession checks if currently inside a screen session
func (s *ScreenManager) IsInsideSession() bool {
	return IsInsideScreen()
}

// parseScreenList parses the output of screen -ls
// Screen output format:
//
//	There is a screen on:
//	    12345.session_name  (Attached)
//	There are screens on:
//	    12345.session_name  (Detached)
//	    67890.other_session (Attached)
func parseScreenList(output string) []string {
	sessions := []string{}
	// Regex to match screen session lines: PID.session_name (status)
	// Example: "12345.my-session (Attached)" or "12345.my-session (Detached)"
	sessionRegex := regexp.MustCompile(`^\s*\d+\.([^\s]+)\s+\((?:Attached|Detached)\)`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		matches := sessionRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			sessions = append(sessions, matches[1])
		}
	}

	return sessions
}

// GetCurrentSessionName returns the name of the current screen session
// Returns empty string if not inside a session
func (s *ScreenManager) GetCurrentSessionName() (string, error) {
	if !s.IsInsideSession() {
		return "", nil
	}

	// The STY environment variable contains the session info
	// Format: PID.session_name
	sty := os.Getenv("STY")
	if sty == "" {
		return "", nil
	}

	// Extract session name after the PID and dot
	parts := strings.SplitN(sty, ".", 2)
	if len(parts) < 2 {
		return "", eris.Errorf("invalid STY format: %s", sty)
	}

	return parts[1], nil
}

// SendKeys sends keys/commands to a screen session
// This is an extra method not in the SessionManager interface
func (s *ScreenManager) SendKeys(name, command string) error {
	// Check if session exists
	exists, err := s.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return eris.Errorf("session '%s' does not exist", name)
	}

	// Send the command to the session
	// screen -S <name> -X stuff "command^M"
	// The ^M is a carriage return (Enter key)
	cmd := exec.Command("screen", "-S", name, "-X", "stuff", command+"\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to send keys to screen session: %s", string(output))
	}

	return nil
}
