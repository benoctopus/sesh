package session

import (
	"os"
	"os/exec"

	"github.com/rotisserie/eris"
)

// SessionManager defines the interface that all session backends must implement
type SessionManager interface {
	// Create creates a new session with the given name at the specified path
	Create(name, path string) error

	// Attach attaches to an existing session
	Attach(name string) error

	// Switch switches to a session (used when already inside a session)
	Switch(name string) error

	// List returns all active session names
	List() ([]string, error)

	// Delete deletes/kills a session
	Delete(name string) error

	// Exists checks if a session exists
	Exists(name string) (bool, error)

	// IsRunning checks if the session manager is running/available
	IsRunning() (bool, error)

	// Name returns the backend name (e.g., "tmux", "zellij")
	Name() string

	// IsInsideSession checks if currently inside a session
	IsInsideSession() bool

	// GetCurrentSessionName returns the name of the current session, or empty string if not in a session
	GetCurrentSessionName() (string, error)
}

// BackendType represents the type of session backend
type BackendType string

const (
	BackendTmux   BackendType = "tmux"
	BackendZellij BackendType = "zellij"
	BackendScreen BackendType = "screen"
	BackendNone   BackendType = "none"
	BackendAuto   BackendType = "auto"

	// Editor backends (VS Code, Cursor, and Zed)
	BackendCodeOpen        BackendType = "code:open"
	BackendCodeWorkspace   BackendType = "code:workspace"
	BackendCodeReplace     BackendType = "code:replace"
	BackendCursorOpen      BackendType = "cursor:open"
	BackendCursorWorkspace BackendType = "cursor:workspace"
	BackendCursorReplace   BackendType = "cursor:replace"
	BackendZedOpen         BackendType = "zed:open"
	BackendZedReuse        BackendType = "zed:reuse"
)

// NewSessionManager creates a new session manager based on the specified backend
// If backend is "auto", it will auto-detect the available backend
func NewSessionManager(backend string) (SessionManager, error) {
	backendType := BackendType(backend)

	// Check for editor backends first (code:* and cursor:*)
	if IsEditorBackend(backend) {
		command, mode, err := ParseEditorBackend(backend)
		if err != nil {
			return nil, err
		}
		return NewEditorManager(command, mode), nil
	}

	// Auto-detect if requested
	if backendType == BackendAuto || backendType == "" {
		detectedBackend, err := DetectBackend()
		if err != nil {
			return nil, err
		}
		backendType = detectedBackend
	}

	switch backendType {
	case BackendTmux:
		return NewTmuxManager(), nil
	case BackendZellij:
		return NewZellijManager(), nil
	case BackendNone:
		return NewNoneManager(), nil
	default:
		return nil, eris.Errorf("unsupported session backend: %s", backend)
	}
}

// DetectBackend auto-detects the available session manager backend
// Priority: tmux -> zellij -> screen -> none
func DetectBackend() (BackendType, error) {
	// Check for tmux
	if isCommandAvailable("tmux") {
		return BackendTmux, nil
	}

	// Check for zellij
	if isCommandAvailable("zellij") {
		return BackendZellij, nil
	}

	// Check for screen
	if isCommandAvailable("screen") {
		return BackendScreen, nil
	}

	// No backend available, use none
	return BackendNone, nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// GetBackendName returns the human-readable name of the backend
func GetBackendName(backend BackendType) string {
	switch backend {
	case BackendTmux:
		return "Tmux"
	case BackendZellij:
		return "Zellij"
	case BackendScreen:
		return "GNU Screen"
	case BackendNone:
		return "None (no session manager)"
	case BackendCodeOpen:
		return "VS Code (new window)"
	case BackendCodeWorkspace:
		return "VS Code (add to workspace)"
	case BackendCodeReplace:
		return "VS Code (replace window)"
	case BackendCursorOpen:
		return "Cursor (new window)"
	case BackendCursorWorkspace:
		return "Cursor (add to workspace)"
	case BackendCursorReplace:
		return "Cursor (replace window)"
	case BackendZedOpen:
		return "Zed (new window)"
	case BackendZedReuse:
		return "Zed (reuse window)"
	default:
		return "Unknown"
	}
}

// NoneManager is a no-op session manager for when no backend is available or desired
type NoneManager struct{}

// NewNoneManager creates a new NoneManager
func NewNoneManager() *NoneManager {
	return &NoneManager{}
}

func (n *NoneManager) Create(name, path string) error {
	return eris.New("no session manager available")
}

func (n *NoneManager) Attach(name string) error {
	return eris.New("no session manager available")
}

func (n *NoneManager) Switch(name string) error {
	return eris.New("no session manager available")
}

func (n *NoneManager) List() ([]string, error) {
	return []string{}, nil
}

func (n *NoneManager) Delete(name string) error {
	return eris.New("no session manager available")
}

func (n *NoneManager) Exists(name string) (bool, error) {
	return false, nil
}

func (n *NoneManager) IsRunning() (bool, error) {
	return false, nil
}

func (n *NoneManager) Name() string {
	return string(BackendNone)
}

func (n *NoneManager) IsInsideSession() bool {
	return false
}

func (n *NoneManager) GetCurrentSessionName() (string, error) {
	return "", nil
}

// IsInsideTmux checks if the current process is running inside tmux
func IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// IsInsideZellij checks if the current process is running inside zellij
func IsInsideZellij() bool {
	return os.Getenv("ZELLIJ") != ""
}

// IsInsideScreen checks if the current process is running inside screen
func IsInsideScreen() bool {
	return os.Getenv("STY") != ""
}

// IsInsideAnySession checks if the current process is running inside any session manager
func IsInsideAnySession() bool {
	return IsInsideTmux() || IsInsideZellij() || IsInsideScreen()
}
