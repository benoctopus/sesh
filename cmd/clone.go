package cmd

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <remote-url>",
	Short: "Clone a git repository into the workspace folder",
	Long: `Clone a git repository into the workspace folder as a bare repo,
create the main worktree, and set up a session.

Examples:
  sesh clone git@github.com:user/repo.git
  sesh clone https://github.com/user/repo.git`,
	Args: cobra.ExactArgs(1),
	RunE: runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) error {
	remoteURL := args[0]

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Ensure workspace directory exists
	if err := config.EnsureWorkspaceDir(); err != nil {
		return eris.Wrap(err, "failed to ensure workspace directory")
	}

	// Initialize database
	dbPath, err := config.GetDBPath()
	if err != nil {
		return eris.Wrap(err, "failed to get database path")
	}

	database, err := db.InitDB(dbPath)
	if err != nil {
		return eris.Wrap(err, "failed to initialize database")
	}
	defer database.Close()

	// Generate project name from remote URL
	projectName, err := git.GenerateProjectName(remoteURL)
	if err != nil {
		return eris.Wrap(err, "failed to generate project name from remote URL")
	}

	// Check if project already exists
	existingProject, err := db.GetProject(database, projectName)
	if err != nil && err != sql.ErrNoRows {
		return eris.Wrap(err, "failed to check for existing project")
	}
	if existingProject != nil {
		return eris.Errorf("project %s already exists in workspace", projectName)
	}

	// Get project path in workspace
	projectPath := workspace.GetProjectPath(cfg.WorkspaceDir, projectName)

	// Clone repository as bare repo
	bareRepoPath := filepath.Join(projectPath, ".git")
	fmt.Printf("Cloning %s to %s...\n", remoteURL, projectPath)
	if err := git.Clone(remoteURL, bareRepoPath); err != nil {
		return eris.Wrap(err, "failed to clone repository")
	}

	// Create project record in database
	project := &models.Project{
		Name:      projectName,
		RemoteURL: remoteURL,
		LocalPath: bareRepoPath,
	}
	if err := db.CreateProject(database, project); err != nil {
		return eris.Wrap(err, "failed to create project in database")
	}

	// Get default branch
	defaultBranch, err := git.GetDefaultBranch(bareRepoPath)
	if err != nil {
		return eris.Wrap(err, "failed to get default branch")
	}

	// Create main worktree
	worktreePath := workspace.GetWorktreePath(projectPath, defaultBranch)
	fmt.Printf("Creating worktree for branch %s...\n", defaultBranch)
	if err := git.CreateWorktree(bareRepoPath, defaultBranch, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create worktree")
	}

	// Create worktree record in database
	worktree := &models.Worktree{
		ProjectID: project.ID,
		Branch:    defaultBranch,
		Path:      worktreePath,
		IsMain:    true,
	}
	if err := db.CreateWorktree(database, worktree); err != nil {
		return eris.Wrap(err, "failed to create worktree in database")
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Generate session name
	sessionName := workspace.GenerateSessionName(projectName, defaultBranch)

	// Create session
	fmt.Printf("Creating %s session %s...\n", sessionMgr.Name(), sessionName)
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	// Create session record in database
	sess := &models.Session{
		WorktreeID:      worktree.ID,
		TmuxSessionName: sessionName,
	}
	if err := db.CreateSession(database, sess); err != nil {
		return eris.Wrap(err, "failed to create session in database")
	}

	fmt.Printf("\nSuccessfully cloned %s\n", projectName)
	fmt.Printf("Worktree: %s\n", worktreePath)
	fmt.Printf("Session: %s\n", sessionName)
	fmt.Printf("\nAttaching to session...\n")

	// Attach to the new session
	if err := sessionMgr.Attach(sessionName); err != nil {
		return eris.Wrap(err, "failed to attach to session")
	}

	return nil
}
