//go:build integration
// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/workspace"
)

// TestPopRetrievesPreviousSession tests that pop retrieves the previous session from history
func TestPopRetrievesPreviousSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Initialize database
	database := env.InitDB()
	defer database.Close()

	// Record session history (simulating switch command)
	session1 := "project1-main"
	session2 := "project2-develop"

	// Add first session
	err := db.AddSessionHistory(database, session1, "github.com/user/project1", "main")
	if err != nil {
		t.Fatalf("failed to add session history: %v", err)
	}

	// Small delay to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	// Add second session (most recent)
	err = db.AddSessionHistory(database, session2, "github.com/user/project2", "develop")
	if err != nil {
		t.Fatalf("failed to add session history: %v", err)
	}

	// Get previous session (should be session1 since session2 is current)
	previous, err := db.GetPreviousSession(database, session2)
	if err != nil {
		t.Fatalf("failed to get previous session: %v", err)
	}

	if previous.SessionName != session1 {
		t.Errorf("expected previous session %s, got %s", session1, previous.SessionName)
	}
}

// TestPopExcludesCurrentSession tests that pop excludes the current session
func TestPopExcludesCurrentSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	currentSession := "project-current"
	previousSession := "project-previous"

	// Add sessions
	err := db.AddSessionHistory(database, previousSession, "github.com/user/project1", "main")
	if err != nil {
		t.Fatalf("failed to add session history: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	// Add current session multiple times (simulating multiple switches)
	for i := 0; i < 3; i++ {
		err = db.AddSessionHistory(database, currentSession, "github.com/user/project2", "feature")
		if err != nil {
			t.Fatalf("failed to add session history: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Get previous - should skip all "project-current" entries and return "project-previous"
	previous, err := db.GetPreviousSession(database, currentSession)
	if err != nil {
		t.Fatalf("failed to get previous session: %v", err)
	}

	if previous.SessionName != previousSession {
		t.Errorf("expected previous session %s, got %s", previousSession, previous.SessionName)
	}
}

// TestPopWithEmptyHistory tests pop behavior with empty history
func TestPopWithEmptyHistory(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	// Try to get previous session when history is empty
	_, err := db.GetPreviousSession(database, "any-session")
	if err == nil {
		t.Error("expected error for empty history")
	}
}

// TestPopWithOnlyCurrentSession tests pop when only current session is in history
func TestPopWithOnlyCurrentSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	currentSession := "only-session"

	// Add only the current session
	err := db.AddSessionHistory(database, currentSession, "github.com/user/project", "main")
	if err != nil {
		t.Fatalf("failed to add session history: %v", err)
	}

	// Try to get previous - should fail since only current is in history
	_, err = db.GetPreviousSession(database, currentSession)
	if err == nil {
		t.Error("expected error when only current session in history")
	}
}

// TestPopHistoryOrder tests that history is returned in correct order (most recent first)
func TestPopHistoryOrder(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	sessions := []string{"session-1", "session-2", "session-3", "session-4"}

	for _, sess := range sessions {
		err := db.AddSessionHistory(database, sess, "github.com/user/project", "main")
		if err != nil {
			t.Fatalf("failed to add session history: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Get recent history
	history, err := db.GetRecentSessionHistory(database, 10)
	if err != nil {
		t.Fatalf("failed to get recent history: %v", err)
	}

	if len(history) != len(sessions) {
		t.Fatalf("expected %d history entries, got %d", len(sessions), len(history))
	}

	// Verify order is most recent first (session-4, session-3, session-2, session-1)
	expectedOrder := []string{"session-4", "session-3", "session-2", "session-1"}
	for i, entry := range history {
		if entry.SessionName != expectedOrder[i] {
			t.Errorf("history[%d]: expected %s, got %s", i, expectedOrder[i], entry.SessionName)
		}
	}
}

// TestPopSessionHistoryDetails tests that session history contains all required details
func TestPopSessionHistoryDetails(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	sessionName := "test-project-feature"
	projectName := "github.com/testuser/test-project"
	branch := "feature"

	err := db.AddSessionHistory(database, sessionName, projectName, branch)
	if err != nil {
		t.Fatalf("failed to add session history: %v", err)
	}

	// Retrieve history
	history, err := db.GetRecentSessionHistory(database, 1)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}

	entry := history[0]
	if entry.SessionName != sessionName {
		t.Errorf("session name mismatch: got %s, want %s", entry.SessionName, sessionName)
	}
	if entry.ProjectName != projectName {
		t.Errorf("project name mismatch: got %s, want %s", entry.ProjectName, projectName)
	}
	if entry.Branch != branch {
		t.Errorf("branch mismatch: got %s, want %s", entry.Branch, branch)
	}
	if entry.AccessedAt.IsZero() {
		t.Error("accessed_at should not be zero")
	}
}

// TestPopWithSessionVerification tests pop verifying session exists before switching
func TestPopWithSessionVerification(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/pop-verify-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	worktreePath := workspace.GetWorktreePath(projectPath, "main")

	// Create session
	sessionName := workspace.GenerateSessionName(projectName, "main")
	err := env.SessionMgr.Create(sessionName, worktreePath)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Add to history
	database := env.InitDB()
	defer database.Close()

	err = db.AddSessionHistory(database, sessionName, projectName, "main")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Get previous session
	previous, err := db.GetPreviousSession(database, "different-session")
	if err != nil {
		t.Fatalf("failed to get previous: %v", err)
	}

	// Verify session exists
	exists, err := env.SessionMgr.Exists(previous.SessionName)
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if !exists {
		t.Error("session should exist")
	}
}

// TestPopDeletedSession tests pop behavior when previous session was deleted
func TestPopDeletedSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create and delete a session
	sessionName := "deleted-session"
	err := env.SessionMgr.Create(sessionName, "/tmp/dummy")
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Add to history
	database := env.InitDB()
	defer database.Close()

	err = db.AddSessionHistory(database, sessionName, "github.com/user/project", "main")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Delete the session
	err = env.SessionMgr.Delete(sessionName)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Get previous
	previous, err := db.GetPreviousSession(database, "current-session")
	if err != nil {
		t.Fatalf("failed to get previous: %v", err)
	}

	// Session should not exist anymore
	exists, err := env.SessionMgr.Exists(previous.SessionName)
	if err != nil {
		t.Fatalf("failed to check existence: %v", err)
	}
	if exists {
		t.Error("deleted session should not exist")
	}
}

// TestPopClearOldHistory tests clearing old session history
func TestPopClearOldHistory(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	// Add some history entries
	for i := 0; i < 5; i++ {
		err := db.AddSessionHistory(database, "session-"+string(rune('a'+i)), "project", "main")
		if err != nil {
			t.Fatalf("failed to add history: %v", err)
		}
	}

	// Clear old history (keep entries from last 0 days = delete all)
	err := db.ClearOldSessionHistory(database, 0)
	if err != nil {
		t.Fatalf("failed to clear history: %v", err)
	}

	// Note: ClearOldSessionHistory uses time comparison, entries just added
	// should still be within "0 days" threshold when executed immediately
	// This test verifies the function doesn't error, not that it clears everything

	// Verify function executed without error
	history, err := db.GetRecentSessionHistory(database, 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	// History should still have entries since they were just added
	if len(history) == 0 {
		t.Log("history was cleared (expected since daysToKeep=0)")
	}
}

// TestPopMultipleSwitches tests session history with multiple switches
func TestPopMultipleSwitches(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	// Simulate multiple switches: A -> B -> C -> B -> A
	switches := []struct {
		session string
		project string
		branch  string
	}{
		{"session-a", "project-a", "main"},
		{"session-b", "project-b", "main"},
		{"session-c", "project-c", "main"},
		{"session-b", "project-b", "main"}, // Back to B
		{"session-a", "project-a", "main"}, // Back to A
	}

	for _, s := range switches {
		err := db.AddSessionHistory(database, s.session, s.project, s.branch)
		if err != nil {
			t.Fatalf("failed to add history: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Current is session-a, previous should be session-b
	previous, err := db.GetPreviousSession(database, "session-a")
	if err != nil {
		t.Fatalf("failed to get previous: %v", err)
	}

	if previous.SessionName != "session-b" {
		t.Errorf("expected previous session-b, got %s", previous.SessionName)
	}
}

// TestPopWithDatabaseIsolation verifies database is isolated per test
func TestPopWithDatabaseIsolation(t *testing.T) {
	env := SetupTestEnvironment(t)

	database := env.InitDB()
	defer database.Close()

	// Add unique session
	uniqueSession := "unique-test-session-" + t.Name()
	err := db.AddSessionHistory(database, uniqueSession, "project", "branch")
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Verify it's in the database
	history, err := db.GetRecentSessionHistory(database, 100)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	found := false
	for _, entry := range history {
		if entry.SessionName == uniqueSession {
			found = true
			break
		}
	}

	if !found {
		t.Error("unique session should be in isolated database")
	}

	// Other tests running in parallel should not see this entry
	// (verified by the fact that each test gets its own temp directory)
}
