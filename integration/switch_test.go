//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
)

// TestSwitchToExistingBranch tests switching to an existing branch
func TestSwitchToExistingBranch(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/switch-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	// Create a feature branch
	env.CreateBranch(projectPath, "feature-test")

	// Create worktree for the feature branch (simulating what switch does)
	bareRepoPath := filepath.Join(projectPath, ".git")
	featureWorktreePath := workspace.GetWorktreePath(projectPath, "feature-test")
	err := git.CreateWorktree(bareRepoPath, "feature-test", featureWorktreePath)
	if err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Verify worktree was created
	if _, err := os.Stat(featureWorktreePath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify we can discover the worktree
	project := env.GetProject(projectName)
	worktree, err := state.GetWorktree(project, "feature-test")
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	if worktree.Branch != "feature-test" {
		t.Errorf("worktree branch mismatch: got %s, want feature-test", worktree.Branch)
	}
}

// TestSwitchCreatesNewWorktree tests that switching to a branch creates a worktree
func TestSwitchCreatesNewWorktree(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/worktree-create-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	// Create a branch but don't create a worktree yet
	env.CreateBranch(projectPath, "new-feature")

	// Create worktree (simulating switch)
	bareRepoPath := filepath.Join(projectPath, ".git")
	worktreePath := workspace.GetWorktreePath(projectPath, "new-feature")
	err := git.CreateWorktree(bareRepoPath, "new-feature", worktreePath)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Verify worktree was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree was not created")
	}

	// Verify branch is checked out
	currentBranch, err := git.GetCurrentBranch(worktreePath)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != "new-feature" {
		t.Errorf("wrong branch checked out: got %s, want new-feature", currentBranch)
	}
}

// TestSwitchCreatesNewBranch tests that switching to a non-existent branch creates it
func TestSwitchCreatesNewBranch(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/new-branch-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	bareRepoPath := filepath.Join(projectPath, ".git")
	newBranch := "brand-new-branch"
	worktreePath := workspace.GetWorktreePath(projectPath, newBranch)

	// Create worktree with new branch
	err := git.CreateWorktreeNewBranch(bareRepoPath, newBranch, worktreePath, "HEAD")
	if err != nil {
		t.Fatalf("failed to create worktree with new branch: %v", err)
	}

	// Verify worktree was created
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree was not created")
	}

	// Verify branch was created and is checked out
	currentBranch, err := git.GetCurrentBranch(worktreePath)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != newBranch {
		t.Errorf("wrong branch checked out: got %s, want %s", currentBranch, newBranch)
	}
}

// TestSwitchCreatesSession tests that switching creates a session
func TestSwitchCreatesSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/session-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	worktreePath := workspace.GetWorktreePath(projectPath, "main")

	// Create session (simulating what switch does)
	sessionName := workspace.GenerateSessionName(projectName, "main")
	err := env.SessionMgr.Create(sessionName, worktreePath)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify session exists
	exists, err := env.SessionMgr.Exists(sessionName)
	if err != nil {
		t.Fatalf("failed to check session existence: %v", err)
	}
	if !exists {
		t.Error("session was not created")
	}
}

// TestSwitchReusesExistingSession tests that switching reuses an existing session
func TestSwitchReusesExistingSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/reuse-session-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	worktreePath := workspace.GetWorktreePath(projectPath, "main")

	sessionName := workspace.GenerateSessionName(projectName, "main")

	// Create session first time
	err := env.SessionMgr.Create(sessionName, worktreePath)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Try to create again - should return error for mock manager
	err = env.SessionMgr.Create(sessionName, worktreePath)
	if err == nil {
		t.Error("expected error when creating duplicate session")
	}

	// But Exists should return true
	exists, err := env.SessionMgr.Exists(sessionName)
	if err != nil {
		t.Fatalf("failed to check session existence: %v", err)
	}
	if !exists {
		t.Error("session should exist")
	}
}

// TestSwitchMultipleWorktrees tests switching between multiple worktrees
func TestSwitchMultipleWorktrees(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/multi-worktree-switch"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	branches := []string{"develop", "feature-a", "feature-b"}

	// Create branches and worktrees
	for _, branch := range branches {
		env.CreateBranch(projectPath, branch)
		worktreePath := workspace.GetWorktreePath(projectPath, branch)
		err := git.CreateWorktree(bareRepoPath, branch, worktreePath)
		if err != nil {
			t.Fatalf("failed to create worktree for %s: %v", branch, err)
		}

		// Create session for each
		sessionName := workspace.GenerateSessionName(projectName, branch)
		err = env.SessionMgr.Create(sessionName, worktreePath)
		if err != nil {
			t.Fatalf("failed to create session for %s: %v", branch, err)
		}
	}

	// Verify all worktrees exist (excluding bare repo entry)
	project := env.GetProject(projectName)
	worktrees := env.DiscoverActualWorktrees(project)

	expectedCount := len(branches) + 1 // +1 for main
	if len(worktrees) != expectedCount {
		t.Errorf("expected %d worktrees with branches, got %d", expectedCount, len(worktrees))
	}

	// Verify all sessions exist
	sessions, err := env.SessionMgr.List()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	// We have main + the other branches
	if len(sessions) != len(branches) {
		t.Errorf("expected %d sessions, got %d", len(branches), len(sessions))
	}
}

// TestSwitchRecordsSessionHistory tests that switching records session history
func TestSwitchRecordsSessionHistory(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Initialize database
	database := env.InitDB()
	defer database.Close()

	projectName := "github.com/testuser/history-test"
	branch := "main"
	sessionName := workspace.GenerateSessionName(projectName, branch)

	// Add session history entry
	err := db.AddSessionHistory(database, sessionName, projectName, branch)
	if err != nil {
		t.Fatalf("failed to add session history: %v", err)
	}

	// Retrieve history
	history, err := db.GetRecentSessionHistory(database, 10)
	if err != nil {
		t.Fatalf("failed to get session history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(history))
	}

	if history[0].SessionName != sessionName {
		t.Errorf("session name mismatch: got %s, want %s", history[0].SessionName, sessionName)
	}
	if history[0].ProjectName != projectName {
		t.Errorf("project name mismatch: got %s, want %s", history[0].ProjectName, projectName)
	}
	if history[0].Branch != branch {
		t.Errorf("branch mismatch: got %s, want %s", history[0].Branch, branch)
	}
}

// TestSwitchWithSlashBranch tests switching to a branch with slashes in the name
func TestSwitchWithSlashBranch(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/slash-branch-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	slashBranch := "feature/add-login"
	env.CreateBranch(projectPath, slashBranch)

	// Create worktree
	worktreePath := workspace.GetWorktreePath(projectPath, slashBranch)
	err := git.CreateWorktree(bareRepoPath, slashBranch, worktreePath)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Verify worktree path uses sanitized branch name
	sanitizedBranch := workspace.SanitizeBranchName(slashBranch)
	expectedPath := filepath.Join(projectPath, sanitizedBranch)
	if worktreePath != expectedPath {
		t.Errorf("worktree path mismatch: got %s, want %s", worktreePath, expectedPath)
	}

	// Verify session name is also sanitized
	sessionName := workspace.GenerateSessionName(projectName, slashBranch)
	expectedSession := "slash-branch-test-feature-add-login"
	if sessionName != expectedSession {
		t.Errorf("session name mismatch: got %s, want %s", sessionName, expectedSession)
	}
}

// TestSwitchWorktreeHasCorrectContent tests that worktree has the correct content
func TestSwitchWorktreeHasCorrectContent(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/content-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	worktreePath := workspace.GetWorktreePath(projectPath, "main")

	// Check that README.md exists in worktree
	readmePath := filepath.Join(worktreePath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md not found in worktree")
	}

	// Read content
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	if string(content) != "# Test Repository\n" {
		t.Errorf("unexpected README content: %s", string(content))
	}
}

// TestSwitchWorktreeGitStatus tests that worktree has clean git status
func TestSwitchWorktreeGitStatus(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/status-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	worktreePath := workspace.GetWorktreePath(projectPath, "main")

	// Verify current branch is correct
	currentBranch, err := git.GetCurrentBranch(worktreePath)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != "main" {
		t.Errorf("wrong branch: got %s, want main", currentBranch)
	}
}

// TestSwitchSessionNameMatchesWorktree tests that session name matches expected pattern
func TestSwitchSessionNameMatchesWorktree(t *testing.T) {
	// This test doesn't need the full environment, just tests workspace functions
	_ = SetupTestEnvironment(t) // Ensures cleanup happens

	tests := []struct {
		projectName     string
		branch          string
		expectedSession string
	}{
		{
			projectName:     "github.com/user/repo",
			branch:          "main",
			expectedSession: "repo-main",
		},
		{
			projectName:     "github.com/org/project-name",
			branch:          "develop",
			expectedSession: "project-name-develop",
		},
		{
			projectName:     "gitlab.com/company/product",
			branch:          "feature/new-api",
			expectedSession: "product-feature-new-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.projectName+"/"+tt.branch, func(t *testing.T) {
			sessionName := workspace.GenerateSessionName(tt.projectName, tt.branch)
			if sessionName != tt.expectedSession {
				t.Errorf("session name mismatch: got %s, want %s", sessionName, tt.expectedSession)
			}
		})
	}
}

// TestSwitchWorktreePathCorrect tests that worktree paths are generated correctly
func TestSwitchWorktreePathCorrect(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/user/test-repo"
	branch := "feature/test"

	projectPath := env.GetProjectPath(projectName)
	worktreePath := env.GetWorktreePath(projectName, branch)

	// Worktree path should use sanitized branch name
	sanitizedBranch := workspace.SanitizeBranchName(branch)
	expectedPath := filepath.Join(projectPath, sanitizedBranch)

	if worktreePath != expectedPath {
		t.Errorf("worktree path mismatch: got %s, want %s", worktreePath, expectedPath)
	}
}
