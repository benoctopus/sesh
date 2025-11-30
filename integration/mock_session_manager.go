//go:build integration
// +build integration

package integration

import (
	"sync"

	"github.com/rotisserie/eris"
)

// MockSessionManager is a mock implementation of session.SessionManager for testing.
// It tracks sessions in memory without requiring any actual session manager.
type MockSessionManager struct {
	mu             sync.RWMutex
	sessions       map[string]string // session name -> working directory
	currentSession string
	insideSession  bool
}

// NewMockSessionManager creates a new MockSessionManager
func NewMockSessionManager() *MockSessionManager {
	return &MockSessionManager{
		sessions: make(map[string]string),
	}
}

// Create creates a new mock session
func (m *MockSessionManager) Create(name, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[name]; exists {
		return eris.Errorf("session %s already exists", name)
	}

	m.sessions[name] = path
	return nil
}

// Attach attaches to an existing mock session
func (m *MockSessionManager) Attach(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[name]; !exists {
		return eris.Errorf("session %s does not exist", name)
	}

	m.currentSession = name
	m.insideSession = true
	return nil
}

// Switch switches to a different mock session
func (m *MockSessionManager) Switch(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[name]; !exists {
		return eris.Errorf("session %s does not exist", name)
	}

	m.currentSession = name
	return nil
}

// List returns all mock session names
func (m *MockSessionManager) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		sessions = append(sessions, name)
	}
	return sessions, nil
}

// Delete deletes a mock session
func (m *MockSessionManager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[name]; !exists {
		return eris.Errorf("session %s does not exist", name)
	}

	delete(m.sessions, name)

	// If we deleted the current session, clear it
	if m.currentSession == name {
		m.currentSession = ""
		m.insideSession = false
	}

	return nil
}

// Exists checks if a mock session exists
func (m *MockSessionManager) Exists(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.sessions[name]
	return exists, nil
}

// IsRunning always returns true for mock sessions
func (m *MockSessionManager) IsRunning() (bool, error) {
	return true, nil
}

// Name returns the backend name
func (m *MockSessionManager) Name() string {
	return "mock"
}

// IsInsideSession returns whether we're inside a mock session
func (m *MockSessionManager) IsInsideSession() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.insideSession
}

// GetCurrentSessionName returns the current mock session name
func (m *MockSessionManager) GetCurrentSessionName() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.insideSession {
		return "", nil
	}
	return m.currentSession, nil
}

// SetInsideSession sets the inside session state for testing
func (m *MockSessionManager) SetInsideSession(inside bool, sessionName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insideSession = inside
	m.currentSession = sessionName
}

// GetSessionPath returns the working directory for a session
func (m *MockSessionManager) GetSessionPath(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	path, exists := m.sessions[name]
	if !exists {
		return "", eris.Errorf("session %s does not exist", name)
	}
	return path, nil
}

// Clear clears all mock sessions
func (m *MockSessionManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions = make(map[string]string)
	m.currentSession = ""
	m.insideSession = false
}

// SendKeys is a no-op for the mock session manager (mimics tmux SendKeys)
func (m *MockSessionManager) SendKeys(sessionName, keys string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.sessions[sessionName]; !exists {
		return eris.Errorf("session %s does not exist", sessionName)
	}
	return nil
}
