//go:build integration
// +build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/workspace"
)

// TestCloneCreatesProjectStructure tests that cloning a repository creates
// the correct project structure in the workspace
func TestCloneCreatesProjectStructure(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create a source repository to clone from
	sourceRepo := env.CreateTestRepo("test-repo")

	// Clone using git.Clone (simulating what the clone command does)
	projectName := "github.com/testuser/test-repo"
	projectPath := workspace.GetProjectPath(env.WorkspaceDir, projectName)
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Clone as bare repo
	err := git.Clone(sourceRepo, bareRepoPath)
	if err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	// Verify bare repo was created
	if _, err := os.Stat(bareRepoPath); os.IsNotExist(err) {
		t.Error("bare repository was not created")
	}

	// Verify it's a bare repo by checking for config file
	configPath := filepath.Join(bareRepoPath, "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("bare repository config file not found")
	}

	// Verify project path exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("project directory was not created")
	}
}

// TestCloneCreatesMainWorktree tests that cloning creates the main worktree
func TestCloneCreatesMainWorktree(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create a source repository
	sourceRepo := env.CreateTestRepo("worktree-test-repo")

	// Clone and create worktree
	projectName := "github.com/testuser/worktree-test-repo"
	projectPath := workspace.GetProjectPath(env.WorkspaceDir, projectName)
	bareRepoPath := filepath.Join(projectPath, ".git")
	defaultBranch := "main"

	err := git.Clone(sourceRepo, bareRepoPath)
	if err != nil {
		t.Fatalf("failed to clone repository: %v", err)
	}

	// Create main worktree
	worktreePath := workspace.GetWorktreePath(projectPath, defaultBranch)
	err = git.CreateWorktree(bareRepoPath, defaultBranch, worktreePath)
	if err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}

	// Verify README.md exists in worktree
	readmePath := filepath.Join(worktreePath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md not found in worktree")
	}

	// Verify .git file (not directory) exists in worktree
	gitPath := filepath.Join(worktreePath, ".git")
	info, err := os.Stat(gitPath)
	if os.IsNotExist(err) {
		t.Error(".git file not found in worktree")
	} else if info.IsDir() {
		t.Error(".git should be a file (not directory) for worktrees")
	}
}

// TestCloneCreatesSession tests that cloning creates a session
func TestCloneCreatesSession(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/session-test-repo"
	defaultBranch := "main"

	// Create the project
	projectPath := env.CreateBareTestRepo(projectName, defaultBranch)
	worktreePath := workspace.GetWorktreePath(projectPath, defaultBranch)

	// Create session using mock session manager
	sessionName := workspace.GenerateSessionName(projectName, defaultBranch)
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

	// Verify session name is correct
	expectedName := "session-test-repo-main"
	if sessionName != expectedName {
		t.Errorf("session name mismatch: got %s, want %s", sessionName, expectedName)
	}
}

// TestCloneDoesNotDuplicateProject tests that cloning doesn't create duplicate projects
func TestCloneDoesNotDuplicateProject(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/duplicate-test-repo"
	defaultBranch := "main"

	// Create the project first time
	env.CreateBareTestRepo(projectName, defaultBranch)

	// Verify project exists
	projects := env.DiscoverProjects()
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}

	// Try to clone again - should fail
	sourceRepo := env.CreateTestRepo("another-source")
	projectPath := workspace.GetProjectPath(env.WorkspaceDir, projectName)
	bareRepoPath := filepath.Join(projectPath, ".git")

	// This should fail because the directory already exists
	err := git.Clone(sourceRepo, bareRepoPath)
	if err == nil {
		t.Error("expected error when cloning to existing path")
	}
}

// TestClonedProjectIsDiscoverable tests that cloned projects are discovered by state
func TestClonedProjectIsDiscoverable(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create multiple projects
	projectNames := []string{
		"github.com/testuser/project1",
		"github.com/testuser/project2",
		"github.com/anotheruser/project3",
	}

	for _, name := range projectNames {
		env.CreateBareTestRepo(name, "main")
	}

	// Discover all projects
	projects := env.DiscoverProjects()

	if len(projects) != len(projectNames) {
		t.Errorf("expected %d projects, got %d", len(projectNames), len(projects))
	}

	// Verify all projects are discovered
	discoveredNames := make(map[string]bool)
	for _, proj := range projects {
		discoveredNames[proj.Name] = true
	}

	for _, name := range projectNames {
		if !discoveredNames[name] {
			t.Errorf("project %s was not discovered", name)
		}
	}
}

// TestClonedWorktreesAreDiscoverable tests that worktrees are discovered correctly
func TestClonedWorktreesAreDiscoverable(t *testing.T) {
	env := SetupTestEnvironment(t)

	projectName := "github.com/testuser/worktree-discovery-test"
	env.CreateBareTestRepo(projectName, "main")

	// Get the project
	project := env.GetProject(projectName)

	// Discover actual worktrees (excludes bare repo entry)
	worktrees := env.DiscoverActualWorktrees(project)

	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree with branch, got %d", len(worktrees))
	}

	if len(worktrees) > 0 && worktrees[0].Branch != "main" {
		t.Errorf("expected worktree branch 'main', got '%s'", worktrees[0].Branch)
	}
}

// TestSessionNameGeneration tests that session names are generated correctly
func TestSessionNameGeneration(t *testing.T) {
	tests := []struct {
		projectName  string
		branch       string
		expectedName string
	}{
		{
			projectName:  "github.com/user/my-project",
			branch:       "main",
			expectedName: "my-project-main",
		},
		{
			projectName:  "github.com/user/another-project",
			branch:       "feature/add-tests",
			expectedName: "another-project-feature-add-tests",
		},
		{
			projectName:  "gitlab.com/org/sub-group/repo",
			branch:       "develop",
			expectedName: "repo-develop",
		},
		{
			projectName:  "bitbucket.org/team/project",
			branch:       "hotfix/urgent-fix",
			expectedName: "project-hotfix-urgent-fix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.projectName+"/"+tt.branch, func(t *testing.T) {
			sessionName := workspace.GenerateSessionName(tt.projectName, tt.branch)
			if sessionName != tt.expectedName {
				t.Errorf("session name mismatch: got %s, want %s", sessionName, tt.expectedName)
			}
		})
	}
}

// TestWorkspaceDirectoryIsolation verifies that tests use isolated workspace directories
func TestWorkspaceDirectoryIsolation(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Verify workspace is not the default ~/.sesh
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	defaultWorkspace := filepath.Join(homeDir, ".sesh")
	if env.WorkspaceDir == defaultWorkspace {
		t.Errorf("test workspace should not be the default: %s", defaultWorkspace)
	}

	// Verify workspace is in a temp directory
	if !isInTempDir(env.WorkspaceDir) {
		t.Errorf("test workspace should be in a temp directory: %s", env.WorkspaceDir)
	}
}

// TestConfigDirectoryIsolation verifies that tests use isolated config directories
func TestConfigDirectoryIsolation(t *testing.T) {
	env := SetupTestEnvironment(t)
	cfg := env.LoadConfig()

	// Verify workspace directory from config matches our test environment
	if cfg.WorkspaceDir != env.WorkspaceDir {
		t.Errorf("config workspace mismatch: got %s, want %s", cfg.WorkspaceDir, env.WorkspaceDir)
	}
}

// isInTempDir checks if a path is within a system temp directory
func isInTempDir(path string) bool {
	tempDir := os.TempDir()
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		return false
	}
	return len(absPath) >= len(absTempDir) && absPath[:len(absTempDir)] == absTempDir
}
