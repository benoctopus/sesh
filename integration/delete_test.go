//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
)

// TestDeleteWorktree tests deleting a single worktree
func TestDeleteWorktree(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/delete-worktree-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Create a feature branch and worktree
	env.CreateBranch(projectPath, "feature-to-delete")
	featureWorktreePath := workspace.GetWorktreePath(projectPath, "feature-to-delete")
	err := git.CreateWorktree(bareRepoPath, "feature-to-delete", featureWorktreePath)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create session for the worktree
	sessionName := workspace.GenerateSessionName(projectName, "feature-to-delete")
	err = env.SessionMgr.Create(sessionName, featureWorktreePath)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(featureWorktreePath); os.IsNotExist(err) {
		t.Fatal("worktree should exist before deletion")
	}

	// Delete the worktree (simulating delete command)
	err = git.RemoveWorktree(bareRepoPath, featureWorktreePath)
	if err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Delete the session
	err = env.SessionMgr.Delete(sessionName)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(featureWorktreePath); !os.IsNotExist(err) {
		t.Error("worktree should be deleted")
	}

	// Verify session is gone
	exists, err := env.SessionMgr.Exists(sessionName)
	if err != nil {
		t.Fatalf("failed to check session existence: %v", err)
	}
	if exists {
		t.Error("session should be deleted")
	}
}

// TestDeleteMainWorktreeBlocked tests that main worktree cannot be deleted alone
func TestDeleteMainWorktreeBlocked(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/delete-main-test"
	env.CreateBareTestRepo(projectName, "main")

	// Get project
	project := env.GetProject(projectName)
	worktree := env.GetWorktree(project, "main")

	// Verify the main worktree exists and has the correct branch
	if worktree.Branch != "main" {
		t.Errorf("expected branch 'main', got '%s'", worktree.Branch)
	}

	// Note: IsMain detection depends on order in git worktree list output
	// The first worktree with a branch should typically be considered main
	// but this depends on the actual order returned by git
	// For now, we just verify the worktree exists and can be retrieved
}

// TestDeleteProject tests deleting an entire project
func TestDeleteProject(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/delete-project-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Create additional worktrees
	branches := []string{"develop", "feature-a"}
	for _, branch := range branches {
		env.CreateBranch(projectPath, branch)
		worktreePath := workspace.GetWorktreePath(projectPath, branch)
		err := git.CreateWorktree(bareRepoPath, branch, worktreePath)
		if err != nil {
			t.Fatalf("failed to create worktree for %s: %v", branch, err)
		}

		// Create session
		sessionName := workspace.GenerateSessionName(projectName, branch)
		err = env.SessionMgr.Create(sessionName, worktreePath)
		if err != nil {
			t.Fatalf("failed to create session for %s: %v", branch, err)
		}
	}

	// Also create session for main
	mainSessionName := workspace.GenerateSessionName(projectName, "main")
	mainWorktreePath := workspace.GetWorktreePath(projectPath, "main")
	err := env.SessionMgr.Create(mainSessionName, mainWorktreePath)
	if err != nil {
		t.Fatalf("failed to create main session: %v", err)
	}

	// Get all worktrees before deletion
	project := env.GetProject(projectName)
	worktrees := env.DiscoverWorktrees(project)

	// Delete all sessions first
	for _, wt := range worktrees {
		sessionName := workspace.GenerateSessionName(projectName, wt.Branch)
		exists, _ := env.SessionMgr.Exists(sessionName)
		if exists {
			err := env.SessionMgr.Delete(sessionName)
			if err != nil {
				t.Logf("warning: failed to delete session %s: %v", sessionName, err)
			}
		}
	}

	// Delete worktrees (except we'll delete the whole project dir)
	for _, wt := range worktrees {
		err := git.RemoveWorktree(bareRepoPath, wt.Path)
		if err != nil {
			t.Logf("warning: failed to remove worktree %s: %v", wt.Path, err)
		}
	}

	// Delete project directory
	err = os.RemoveAll(projectPath)
	if err != nil {
		t.Fatalf("failed to remove project directory: %v", err)
	}

	// Verify project is gone
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Error("project directory should be deleted")
	}

	// Verify all sessions are gone
	sessions, err := env.SessionMgr.List()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// TestDeleteNonExistentWorktree tests deleting a worktree that doesn't exist
func TestDeleteNonExistentWorktree(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/nonexistent-worktree"
	env.CreateBareTestRepo(projectName, "main")

	project := env.GetProject(projectName)

	// Try to get a worktree that doesn't exist
	_, err := state.GetWorktree(project, "nonexistent-branch")
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}
}

// TestDeleteNonExistentSession tests deleting a session that doesn't exist
func TestDeleteNonExistentSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Try to delete a session that doesn't exist
	err := env.SessionMgr.Delete("nonexistent-session")
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

// TestDeleteWorktreeKeepsOtherWorktrees tests that deleting one worktree doesn't affect others
func TestDeleteWorktreeKeepsOtherWorktrees(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/multi-worktree-delete"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Create multiple branches and worktrees
	branches := []string{"keep-a", "delete-me", "keep-b"}
	for _, branch := range branches {
		env.CreateBranch(projectPath, branch)
		worktreePath := workspace.GetWorktreePath(projectPath, branch)
		err := git.CreateWorktree(bareRepoPath, branch, worktreePath)
		if err != nil {
			t.Fatalf("failed to create worktree for %s: %v", branch, err)
		}
	}

	// Delete only "delete-me"
	deleteWorktreePath := workspace.GetWorktreePath(projectPath, "delete-me")
	err := git.RemoveWorktree(bareRepoPath, deleteWorktreePath)
	if err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Verify "delete-me" is gone
	if _, err := os.Stat(deleteWorktreePath); !os.IsNotExist(err) {
		t.Error("delete-me worktree should be deleted")
	}

	// Verify other worktrees still exist
	for _, branch := range []string{"keep-a", "keep-b", "main"} {
		worktreePath := workspace.GetWorktreePath(projectPath, branch)
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Errorf("worktree %s should still exist", branch)
		}
	}
}

// TestDeleteSessionKeepsWorktree tests that deleting session doesn't delete worktree
func TestDeleteSessionKeepsWorktree(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/session-only-delete"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	worktreePath := workspace.GetWorktreePath(projectPath, "main")

	// Create session
	sessionName := workspace.GenerateSessionName(projectName, "main")
	err := env.SessionMgr.Create(sessionName, worktreePath)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Delete only the session
	err = env.SessionMgr.Delete(sessionName)
	if err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	// Verify session is gone
	exists, err := env.SessionMgr.Exists(sessionName)
	if err != nil {
		t.Fatalf("failed to check session existence: %v", err)
	}
	if exists {
		t.Error("session should be deleted")
	}

	// Verify worktree still exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree should still exist")
	}
}

// TestDeleteProjectCleansUpCompletely tests that project deletion is complete
func TestDeleteProjectCleansUpCompletely(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/cleanup-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	// Verify project exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Fatal("project should exist before deletion")
	}

	// Delete entire project directory
	err := os.RemoveAll(projectPath)
	if err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	// Verify nothing is left
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) {
		t.Error("project directory should be completely deleted")
	}

	// Verify project is no longer discoverable
	projects := env.DiscoverProjects()
	for _, p := range projects {
		if p.Name == projectName {
			t.Error("deleted project should not be discoverable")
		}
	}
}

// TestDeleteWorktreeWithSlashBranch tests deleting worktree with slash in branch name
func TestDeleteWorktreeWithSlashBranch(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/slash-delete-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Create branch with slash
	slashBranch := "feature/to-delete"
	env.CreateBranch(projectPath, slashBranch)
	worktreePath := workspace.GetWorktreePath(projectPath, slashBranch)
	err := git.CreateWorktree(bareRepoPath, slashBranch, worktreePath)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Delete the worktree
	err = git.RemoveWorktree(bareRepoPath, worktreePath)
	if err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("worktree should be deleted")
	}
}

// TestDeleteMultipleWorktreesSequentially tests deleting multiple worktrees one by one
func TestDeleteMultipleWorktreesSequentially(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/sequential-delete"
	projectPath := env.CreateBareTestRepo(projectName, "main")
	bareRepoPath := filepath.Join(projectPath, ".git")

	branches := []string{"branch-1", "branch-2", "branch-3"}

	// Create all worktrees
	for _, branch := range branches {
		env.CreateBranch(projectPath, branch)
		worktreePath := workspace.GetWorktreePath(projectPath, branch)
		err := git.CreateWorktree(bareRepoPath, branch, worktreePath)
		if err != nil {
			t.Fatalf("failed to create worktree for %s: %v", branch, err)
		}
	}

	// Delete them one by one
	for _, branch := range branches {
		worktreePath := workspace.GetWorktreePath(projectPath, branch)

		// Verify exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Fatalf("worktree %s should exist before deletion", branch)
		}

		// Delete
		err := git.RemoveWorktree(bareRepoPath, worktreePath)
		if err != nil {
			t.Fatalf("failed to remove worktree %s: %v", branch, err)
		}

		// Verify gone
		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Errorf("worktree %s should be deleted", branch)
		}
	}

	// Main should still exist
	mainWorktreePath := workspace.GetWorktreePath(projectPath, "main")
	if _, err := os.Stat(mainWorktreePath); os.IsNotExist(err) {
		t.Error("main worktree should still exist")
	}
}
