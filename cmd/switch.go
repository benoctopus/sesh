package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/fuzzy"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	switchCreateBranch bool
	switchProjectName  string
)

var switchCmd = &cobra.Command{
	Use:   "switch [branch]",
	Short: "Switch to a branch (create worktree if needed)",
	Long: `Switch to a branch, creating a worktree and session if they don't exist.
If no branch is specified, an interactive fuzzy finder will show all available branches.

The project is automatically detected from the current working directory,
or can be specified explicitly with the --project flag.

Examples:
  sesh switch feature-foo          # Switch to feature-foo branch
  sesh switch -b new-feature       # Create new branch and switch
  sesh switch                      # Interactive fuzzy branch selection
  sesh switch --project myproject feature-bar  # Explicit project`,
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
	switchCmd.Flags().BoolVarP(&switchCreateBranch, "create", "b", false, "Create a new branch (like git checkout -b)")
	switchCmd.Flags().StringVarP(&switchProjectName, "project", "p", "", "Specify project explicitly")
}

func runSwitch(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
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

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Resolve project
	proj, err := project.ResolveProject(database, switchProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	var branch string
	if len(args) > 0 {
		branch = args[0]
	} else {
		// No branch specified, use fuzzy finder
		fmt.Println("Fetching latest branches...")
		if err := git.Fetch(proj.LocalPath); err != nil {
			fmt.Printf("Warning: failed to fetch: %v\n", err)
		}

		// Get all branches
		branches, err := git.ListRemoteBranches(proj.LocalPath)
		if err != nil {
			return eris.Wrap(err, "failed to list branches")
		}

		if len(branches) == 0 {
			return eris.New("no branches found")
		}

		// Use fuzzy finder to select branch
		selectedBranch, err := fuzzy.SelectBranch(branches)
		if err != nil {
			return eris.Wrap(err, "failed to select branch")
		}

		branch = selectedBranch
	}

	// Check if worktree already exists
	existingWorktree, err := db.GetWorktree(database, proj.ID, branch)
	if err != nil && err != sql.ErrNoRows {
		return eris.Wrap(err, "failed to check for existing worktree")
	}

	if existingWorktree != nil {
		// Worktree exists, attach to existing session
		fmt.Printf("Switching to existing worktree: %s\n", existingWorktree.Path)

		// Get or create session
		sess, err := db.GetSessionByWorktree(database, existingWorktree.ID)
		if err != nil && err != sql.ErrNoRows {
			return eris.Wrap(err, "failed to get session")
		}

		// Initialize session manager
		sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
		if err != nil {
			return eris.Wrap(err, "failed to initialize session manager")
		}

		if sess != nil {
			// Session exists, check if it's still running
			exists, err := sessionMgr.Exists(sess.TmuxSessionName)
			if err != nil {
				return eris.Wrap(err, "failed to check session existence")
			}

			if exists {
				// Update last used timestamp
				if err := db.UpdateWorktreeLastUsed(database, existingWorktree.ID); err != nil {
					fmt.Printf("Warning: failed to update worktree timestamp: %v\n", err)
				}
				if err := db.UpdateSessionLastAttached(database, sess.ID); err != nil {
					fmt.Printf("Warning: failed to update session timestamp: %v\n", err)
				}

				// Attach to session
				return sessionMgr.Attach(sess.TmuxSessionName)
			}

			// Session doesn't exist anymore, recreate it
			fmt.Printf("Recreating %s session...\n", sessionMgr.Name())
			if err := sessionMgr.Create(sess.TmuxSessionName, existingWorktree.Path); err != nil {
				return eris.Wrap(err, "failed to recreate session")
			}

			return sessionMgr.Attach(sess.TmuxSessionName)
		}

		// No session exists, create one
		sessionName := workspace.GenerateSessionName(proj.Name, branch)
		fmt.Printf("Creating %s session %s...\n", sessionMgr.Name(), sessionName)
		if err := sessionMgr.Create(sessionName, existingWorktree.Path); err != nil {
			return eris.Wrap(err, "failed to create session")
		}

		// Create session record
		newSession := &models.Session{
			WorktreeID:      existingWorktree.ID,
			TmuxSessionName: sessionName,
		}
		if err := db.CreateSession(database, newSession); err != nil {
			return eris.Wrap(err, "failed to create session in database")
		}

		return sessionMgr.Attach(sessionName)
	}

	// Worktree doesn't exist, create it
	if switchCreateBranch {
		// Check if branch already exists
		exists, err := git.DoesBranchExist(proj.LocalPath, branch)
		if err != nil {
			return eris.Wrap(err, "failed to check branch existence")
		}
		if exists {
			return eris.Errorf("branch %s already exists, use without -b to switch to it", branch)
		}

		fmt.Printf("Creating new branch and worktree: %s\n", branch)
	} else {
		// Check if branch exists remotely
		exists, err := git.DoesRemoteBranchExist(proj.LocalPath, branch)
		if err != nil {
			return eris.Wrap(err, "failed to check remote branch existence")
		}
		if !exists {
			return eris.Errorf("branch %s does not exist remotely, use -b to create it", branch)
		}

		fmt.Printf("Creating worktree for branch: %s\n", branch)
	}

	// Get worktree path
	projectPath := workspace.GetProjectPath(cfg.WorkspaceDir, proj.Name)
	worktreePath := workspace.GetWorktreePath(projectPath, branch)

	// Create worktree
	if switchCreateBranch {
		// Create new branch from HEAD
		if err := git.CreateWorktreeFromRef(proj.LocalPath, branch, worktreePath); err != nil {
			return eris.Wrap(err, "failed to create worktree for new branch")
		}
	} else {
		// Create worktree from existing branch
		if err := git.CreateWorktree(proj.LocalPath, branch, worktreePath); err != nil {
			return eris.Wrap(err, "failed to create worktree")
		}
	}

	// Create worktree record in database
	newWorktree := &models.Worktree{
		ProjectID: proj.ID,
		Branch:    branch,
		Path:      worktreePath,
		IsMain:    false,
	}
	if err := db.CreateWorktree(database, newWorktree); err != nil {
		return eris.Wrap(err, "failed to create worktree in database")
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Create session
	sessionName := workspace.GenerateSessionName(proj.Name, branch)
	fmt.Printf("Creating %s session %s...\n", sessionMgr.Name(), sessionName)
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	// Create session record in database
	newSession := &models.Session{
		WorktreeID:      newWorktree.ID,
		TmuxSessionName: sessionName,
	}
	if err := db.CreateSession(database, newSession); err != nil {
		return eris.Wrap(err, "failed to create session in database")
	}

	fmt.Printf("\nSuccessfully switched to %s\n", branch)
	fmt.Printf("Worktree: %s\n", worktreePath)
	fmt.Printf("Session: %s\n", sessionName)
	fmt.Printf("\nAttaching to session...\n")

	// Attach to session
	return sessionMgr.Attach(sessionName)
}
