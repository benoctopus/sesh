//go:build integration
// +build integration

// Package integration contains integration tests for sesh commands.
// These tests create isolated environments and do not modify global sesh configuration.
//
// Run integration tests with:
//
//	go test -tags=integration -v ./integration/...
//
// Or use the task runner:
//
//	task test:integration
package integration

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
)

// TestEnvironment holds the isolated test environment configuration
type TestEnvironment struct {
	// WorkspaceDir is the isolated workspace directory for this test
	WorkspaceDir string
	// ConfigDir is the isolated config directory for this test
	ConfigDir string
	// DBPath is the path to the test database
	DBPath string
	// OrigWorkspaceEnv stores the original SESH_WORKSPACE env var
	OrigWorkspaceEnv string
	// OrigConfigEnv stores the original XDG_CONFIG_HOME env var
	OrigConfigEnv string
	// OrigBackendEnv stores the original SESH_SESSION_BACKEND env var
	OrigBackendEnv string
	// SessionMgr is the session manager for this test
	SessionMgr session.SessionManager
	// t is the test context
	t *testing.T
}

// SetupTestEnvironment creates an isolated test environment that doesn't affect
// global sesh configuration. It returns a cleanup function that must be called
// when the test is complete.
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	env := &TestEnvironment{t: t}

	// Store original environment variables
	env.OrigWorkspaceEnv = os.Getenv("SESH_WORKSPACE")
	env.OrigConfigEnv = os.Getenv("XDG_CONFIG_HOME")
	env.OrigBackendEnv = os.Getenv("SESH_SESSION_BACKEND")

	// Create temporary directories
	tempDir := t.TempDir()
	env.WorkspaceDir = filepath.Join(tempDir, "workspace")
	env.ConfigDir = filepath.Join(tempDir, "config", "sesh")
	env.DBPath = filepath.Join(env.ConfigDir, "sesh.db")

	// Create the directories
	if err := os.MkdirAll(env.WorkspaceDir, 0o755); err != nil {
		t.Fatalf("failed to create workspace dir: %v", err)
	}
	if err := os.MkdirAll(env.ConfigDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	// Set environment variables to isolate test
	os.Setenv("SESH_WORKSPACE", env.WorkspaceDir)
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, "config"))
	os.Setenv("SESH_SESSION_BACKEND", "none") // Use mock session manager

	// Create mock session manager
	env.SessionMgr = NewMockSessionManager()

	// Register cleanup
	t.Cleanup(func() {
		env.Cleanup()
	})

	return env
}

// Cleanup restores the original environment and cleans up test resources
func (env *TestEnvironment) Cleanup() {
	// Restore original environment variables
	if env.OrigWorkspaceEnv != "" {
		os.Setenv("SESH_WORKSPACE", env.OrigWorkspaceEnv)
	} else {
		os.Unsetenv("SESH_WORKSPACE")
	}

	if env.OrigConfigEnv != "" {
		os.Setenv("XDG_CONFIG_HOME", env.OrigConfigEnv)
	} else {
		os.Unsetenv("XDG_CONFIG_HOME")
	}

	if env.OrigBackendEnv != "" {
		os.Setenv("SESH_SESSION_BACKEND", env.OrigBackendEnv)
	} else {
		os.Unsetenv("SESH_SESSION_BACKEND")
	}
}

// LoadConfig loads configuration for the test environment
func (env *TestEnvironment) LoadConfig() *config.Config {
	env.t.Helper()
	cfg, err := config.LoadConfig()
	if err != nil {
		env.t.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

// InitDB initializes the test database
func (env *TestEnvironment) InitDB() *sql.DB {
	env.t.Helper()
	database, err := db.InitDB(env.DBPath)
	if err != nil {
		env.t.Fatalf("failed to init database: %v", err)
	}
	return database
}

// CreateTestRepo creates a test git repository that can be used for testing.
// Returns the path to the repository.
func (env *TestEnvironment) CreateTestRepo(name string) string {
	env.t.Helper()

	repoDir := filepath.Join(env.t.TempDir(), name)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		env.t.Fatalf("failed to create repo dir: %v", err)
	}

	// Initialize a git repository
	if err := runGitCommand(repoDir, "init"); err != nil {
		env.t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	if err := runGitCommand(repoDir, "config", "user.email", "test@example.com"); err != nil {
		env.t.Fatalf("failed to configure git email: %v", err)
	}
	if err := runGitCommand(repoDir, "config", "user.name", "Test User"); err != nil {
		env.t.Fatalf("failed to configure git name: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0o644); err != nil {
		env.t.Fatalf("failed to write README: %v", err)
	}
	if err := runGitCommand(repoDir, "add", "."); err != nil {
		env.t.Fatalf("failed to git add: %v", err)
	}
	// Use config options to disable signing for commit
	if err := runGitCommandWithConfig(repoDir, "commit", "-m", "Initial commit"); err != nil {
		env.t.Fatalf("failed to git commit: %v", err)
	}

	// Create a branch called "main" if needed (for older git versions that default to "master")
	currentBranch, err := git.GetCurrentBranch(repoDir)
	if err != nil {
		env.t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != "main" {
		if err := runGitCommand(repoDir, "branch", "-M", "main"); err != nil {
			env.t.Fatalf("failed to rename branch to main: %v", err)
		}
	}

	return repoDir
}

// CreateBareTestRepo creates a bare repository in the workspace, simulating what clone does.
// Returns the project path.
func (env *TestEnvironment) CreateBareTestRepo(projectName, defaultBranch string) string {
	env.t.Helper()

	// Create source repo to clone from
	sourceRepo := env.CreateTestRepo("source-" + filepath.Base(projectName))

	// Create project path in workspace
	projectPath := workspace.GetProjectPath(env.WorkspaceDir, projectName)
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Clone as bare repo
	if err := git.Clone(sourceRepo, bareRepoPath); err != nil {
		env.t.Fatalf("failed to clone bare repo: %v", err)
	}

	// Create main worktree
	worktreePath := workspace.GetWorktreePath(projectPath, defaultBranch)
	if err := git.CreateWorktree(bareRepoPath, defaultBranch, worktreePath); err != nil {
		env.t.Fatalf("failed to create worktree: %v", err)
	}

	return projectPath
}

// GetProjectPath returns the project path for a given project name
func (env *TestEnvironment) GetProjectPath(projectName string) string {
	return workspace.GetProjectPath(env.WorkspaceDir, projectName)
}

// GetWorktreePath returns the worktree path for a given project and branch
func (env *TestEnvironment) GetWorktreePath(projectName, branch string) string {
	projectPath := env.GetProjectPath(projectName)
	return workspace.GetWorktreePath(projectPath, branch)
}

// DiscoverProjects returns all projects in the workspace
func (env *TestEnvironment) DiscoverProjects() []*models.Project {
	env.t.Helper()
	projects, err := state.DiscoverProjects(env.WorkspaceDir)
	if err != nil {
		env.t.Fatalf("failed to discover projects: %v", err)
	}
	return projects
}

// DiscoverWorktrees returns all worktrees for a project
func (env *TestEnvironment) DiscoverWorktrees(project *models.Project) []*models.Worktree {
	env.t.Helper()
	worktrees, err := state.DiscoverWorktrees(project)
	if err != nil {
		env.t.Fatalf("failed to discover worktrees: %v", err)
	}
	return worktrees
}

// DiscoverActualWorktrees returns worktrees that have a branch (excludes bare repo entries)
// When git worktree list is run on a bare repo, it includes the bare repo itself as an entry
// with no branch. This helper filters those out.
func (env *TestEnvironment) DiscoverActualWorktrees(project *models.Project) []*models.Worktree {
	env.t.Helper()
	worktrees := env.DiscoverWorktrees(project)
	return filterActualWorktrees(worktrees)
}

// filterActualWorktrees filters out bare repo entries (worktrees without branches)
func filterActualWorktrees(worktrees []*models.Worktree) []*models.Worktree {
	var actual []*models.Worktree
	for _, wt := range worktrees {
		if wt.Branch != "" {
			actual = append(actual, wt)
		}
	}
	return actual
}

// CountActualWorktrees returns the count of actual worktrees (excluding bare repo entries)
func CountActualWorktrees(worktrees []*models.Worktree) int {
	count := 0
	for _, wt := range worktrees {
		if wt.Branch != "" {
			count++
		}
	}
	return count
}

// GetProject retrieves a project by name
func (env *TestEnvironment) GetProject(projectName string) *models.Project {
	env.t.Helper()
	project, err := state.GetProject(env.WorkspaceDir, projectName)
	if err != nil {
		env.t.Fatalf("failed to get project %s: %v", projectName, err)
	}
	return project
}

// GetWorktree retrieves a worktree by project and branch
func (env *TestEnvironment) GetWorktree(project *models.Project, branch string) *models.Worktree {
	env.t.Helper()
	worktree, err := state.GetWorktree(project, branch)
	if err != nil {
		env.t.Fatalf("failed to get worktree for branch %s: %v", branch, err)
	}
	return worktree
}

// CreateBranch creates a new branch in the test repository
func (env *TestEnvironment) CreateBranch(projectPath, branchName string) {
	env.t.Helper()
	bareRepoPath := filepath.Join(projectPath, ".git")

	// Find any worktree to run git commands in
	worktrees, err := git.ListWorktrees(bareRepoPath)
	if err != nil || len(worktrees) == 0 {
		env.t.Fatalf("failed to find worktrees to create branch: %v", err)
	}

	// Create branch from main
	if err := runGitCommand(worktrees[0].Path, "branch", branchName); err != nil {
		env.t.Fatalf("failed to create branch %s: %v", branchName, err)
	}
}

// runGitCommand runs a git command in the specified directory
func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Set environment variables for git operations
	// Disable commit signing to avoid issues in test environments
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	return cmd.Run()
}

// runGitCommandWithConfig runs a git command with specific config options
func runGitCommandWithConfig(dir string, args ...string) error {
	// Prepend config options to disable signing
	fullArgs := append([]string{"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	return cmd.Run()
}
