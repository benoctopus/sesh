//go:build integration
// +build integration

package integration

import (
	"path/filepath"
	"testing"

	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
)

// TestListProjectsEmpty tests listing projects when none exist
func TestListProjectsEmpty(t *testing.T) {
	env := SetupTestEnvironment(t)

	projects := env.DiscoverProjects()
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

// TestListProjectsWithSingleProject tests listing a single project
func TestListProjectsWithSingleProject(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/single-project"
	env.CreateBareTestRepo(projectName, "main")

	projects := env.DiscoverProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	if projects[0].Name != projectName {
		t.Errorf("project name mismatch: got %s, want %s", projects[0].Name, projectName)
	}
}

// TestListProjectsWithMultipleProjects tests listing multiple projects
func TestListProjectsWithMultipleProjects(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectNames := []string{
		"github.com/user1/project-a",
		"github.com/user2/project-b",
		"gitlab.com/org/project-c",
	}

	for _, name := range projectNames {
		env.CreateBareTestRepo(name, "main")
	}

	projects := env.DiscoverProjects()
	if len(projects) != len(projectNames) {
		t.Errorf("expected %d projects, got %d", len(projectNames), len(projects))
	}
}

// TestListWorktreesForProject tests listing worktrees for a specific project
func TestListWorktreesForProject(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/multi-worktree"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	// Create additional branch
	env.CreateBranch(projectPath, "feature-branch")

	// Create worktree for the feature branch
	bareRepoPath := filepath.Join(projectPath, ".git")
	featureWorktreePath := workspace.GetWorktreePath(projectPath, "feature-branch")
	err := git.CreateWorktree(bareRepoPath, "feature-branch", featureWorktreePath)
	if err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Get project and list actual worktrees (excludes bare repo entry)
	project := env.GetProject(projectName)
	worktrees := env.DiscoverActualWorktrees(project)

	if len(worktrees) != 2 {
		t.Errorf("expected 2 worktrees with branches, got %d", len(worktrees))
	}

	// Verify branches
	branches := make(map[string]bool)
	for _, wt := range worktrees {
		branches[wt.Branch] = true
	}

	if !branches["main"] {
		t.Error("main branch worktree not found")
	}
	if !branches["feature-branch"] {
		t.Error("feature-branch worktree not found")
	}
}

// TestListSessionsEmpty tests listing sessions when none exist
func TestListSessionsEmpty(t *testing.T) {
	env := SetupTestEnvironment(t)

	sessions, err := env.SessionMgr.List()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// TestListSessionsWithActiveSessions tests listing active sessions
func TestListSessionsWithActiveSessions(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create some sessions
	sessionNames := []string{
		"project1-main",
		"project2-develop",
		"project3-feature-test",
	}

	for _, name := range sessionNames {
		err := env.SessionMgr.Create(name, "/tmp/dummy")
		if err != nil {
			t.Fatalf("failed to create session %s: %v", name, err)
		}
	}

	// List sessions
	sessions, err := env.SessionMgr.List()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != len(sessionNames) {
		t.Errorf("expected %d sessions, got %d", len(sessionNames), len(sessions))
	}
}

// TestListProjectWithNoWorktrees tests listing a project that has no worktrees
// (edge case - bare repo exists but no worktrees created yet)
func TestListProjectWithNoWorktrees(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create source repo
	sourceRepo := env.CreateTestRepo("no-worktree-repo")

	// Clone as bare repo but don't create worktree
	projectName := "github.com/testuser/no-worktree-repo"
	projectPath := workspace.GetProjectPath(env.WorkspaceDir, projectName)
	bareRepoPath := filepath.Join(projectPath, ".git")

	err := git.Clone(sourceRepo, bareRepoPath)
	if err != nil {
		t.Fatalf("failed to clone: %v", err)
	}

	// Project should be discoverable
	projects := env.DiscoverProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	// But actual worktrees (with branches) should be empty
	// Note: git worktree list on a bare repo includes the bare repo itself
	// as an entry without a branch, so we filter those out
	worktrees, err := state.DiscoverWorktrees(projects[0])
	if err != nil {
		t.Fatalf("failed to discover worktrees: %v", err)
	}

	actualWorktrees := filterActualWorktrees(worktrees)
	if len(actualWorktrees) != 0 {
		t.Errorf("expected 0 actual worktrees with branches, got %d", len(actualWorktrees))
	}
}

// TestListWorktreePaths tests that worktree paths are correct
func TestListWorktreePaths(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/path-test"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	project := env.GetProject(projectName)
	worktrees := env.DiscoverActualWorktrees(project)

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree with branch, got %d", len(worktrees))
	}

	expectedPath := workspace.GetWorktreePath(projectPath, "main")
	if worktrees[0].Path != expectedPath {
		t.Errorf("worktree path mismatch: got %s, want %s", worktrees[0].Path, expectedPath)
	}
}

// TestListProjectsByShortName tests finding projects by short name
func TestListProjectsByShortName(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create projects with same repo name but different paths
	env.CreateBareTestRepo("github.com/user1/common-name", "main")
	env.CreateBareTestRepo("github.com/user2/unique-project", "main")

	// Should find unique project by short name
	project, err := state.GetProjectByShortName(env.WorkspaceDir, "unique-project")
	if err != nil {
		t.Fatalf("failed to get project by short name: %v", err)
	}
	if project.Name != "github.com/user2/unique-project" {
		t.Errorf("wrong project returned: %s", project.Name)
	}

	// Should fail for ambiguous short name if we had multiple with same name
	// (only one project with "common-name" exists, so this should work)
	project, err = state.GetProjectByShortName(env.WorkspaceDir, "common-name")
	if err != nil {
		t.Fatalf("failed to get project by short name: %v", err)
	}
	if project.Name != "github.com/user1/common-name" {
		t.Errorf("wrong project returned: %s", project.Name)
	}
}

// TestListMultipleProjectsSameShortName tests behavior with multiple projects having same repo name
func TestListMultipleProjectsSameShortName(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create projects with same repo name
	env.CreateBareTestRepo("github.com/user1/same-repo", "main")
	env.CreateBareTestRepo("github.com/user2/same-repo", "main")

	// Should fail because ambiguous
	_, err := state.GetProjectByShortName(env.WorkspaceDir, "same-repo")
	if err == nil {
		t.Error("expected error for ambiguous short name")
	}
}

// TestListWorktreeBranchNames tests correct branch name extraction
func TestListWorktreeBranchNames(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/branch-names"
	projectPath := env.CreateBareTestRepo(projectName, "main")

	// Create branches with various naming patterns
	branchNames := []string{
		"feature/add-login",
		"bugfix/fix-auth",
		"release/v1.0.0",
	}

	bareRepoPath := filepath.Join(projectPath, ".git")
	for _, branch := range branchNames {
		env.CreateBranch(projectPath, branch)
		worktreePath := workspace.GetWorktreePath(projectPath, branch)
		err := git.CreateWorktree(bareRepoPath, branch, worktreePath)
		if err != nil {
			t.Fatalf("failed to create worktree for %s: %v", branch, err)
		}
	}

	// Get actual worktrees (excludes bare repo entry)
	project := env.GetProject(projectName)
	worktrees := env.DiscoverActualWorktrees(project)

	expectedCount := len(branchNames) + 1 // +1 for main
	if len(worktrees) != expectedCount {
		t.Errorf("expected %d worktrees with branches, got %d", expectedCount, len(worktrees))
	}

	// Verify all branches are present
	discoveredBranches := make(map[string]bool)
	for _, wt := range worktrees {
		discoveredBranches[wt.Branch] = true
	}

	if !discoveredBranches["main"] {
		t.Error("main branch not found")
	}
	for _, branch := range branchNames {
		if !discoveredBranches[branch] {
			t.Errorf("branch %s not found in worktrees", branch)
		}
	}
}
