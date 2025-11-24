package cmd

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	deleteAll         bool
	deleteForce       bool
	deleteProjectName string
)

var deleteCmd = &cobra.Command{
	Use:   "delete [branch]",
	Short: "Delete worktree, session, or entire project",
	Long: `Delete a worktree and its associated session, or delete an entire project.

By default, deletes the specified branch's worktree and session.
Use --all to delete the entire project including all worktrees.

The project is automatically detected from the current working directory,
or can be specified explicitly with the --project flag.

Examples:
  sesh delete feature-foo          # Delete feature-foo worktree/session
  sesh delete --all                # Delete entire project (requires confirmation)
  sesh delete --all --force        # Delete entire project without confirmation
  sesh delete --project myproject --all  # Delete specific project`,
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVar(&deleteAll, "all", false, "Delete entire project")
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
	deleteCmd.Flags().StringVarP(&deleteProjectName, "project", "p", "", "Specify project explicitly")
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Ensure config directory exists (needed for database)
	if err := config.EnsureConfigDir(); err != nil {
		return eris.Wrap(err, "failed to ensure config directory")
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
	proj, err := project.ResolveProject(database, deleteProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	if deleteAll {
		return deleteProject(database, cfg, proj)
	}

	// Delete specific branch
	if len(args) == 0 {
		return eris.New("branch name required (or use --all to delete entire project)")
	}

	branch := args[0]
	return deleteBranch(database, cfg, proj, branch)
}

func deleteProject(database *sql.DB, cfg *config.Config, proj *models.Project) error {
	// Get all worktrees for this project
	worktrees, err := db.GetWorktreesByProject(database, proj.ID)
	if err != nil {
		return eris.Wrap(err, "failed to get worktrees")
	}

	if !deleteForce {
		// Ask for confirmation
		fmt.Printf(
			"This will delete project '%s' with %d worktree(s) and all associated sessions.\n",
			proj.Name,
			len(worktrees),
		)
		fmt.Printf("Project path: %s\n", proj.LocalPath)
		fmt.Print("Are you sure? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Delete all sessions
	for _, wt := range worktrees {
		sess, err := db.GetSessionByWorktree(database, wt.ID)
		if err != nil && err != sql.ErrNoRows {
			fmt.Printf("Warning: failed to get session for worktree %d: %v\n", wt.ID, err)
			continue
		}

		if sess != nil {
			// Kill session if it exists
			exists, err := sessionMgr.Exists(sess.TmuxSessionName)
			if err != nil {
				fmt.Printf("Warning: failed to check session existence: %v\n", err)
			} else if exists {
				fmt.Printf("Killing %s session: %s\n", sessionMgr.Name(), sess.TmuxSessionName)
				if err := sessionMgr.Delete(sess.TmuxSessionName); err != nil {
					fmt.Printf("Warning: failed to kill session: %v\n", err)
				}
			}
		}

		// Remove worktree
		fmt.Printf("Removing worktree: %s\n", wt.Path)
		if err := git.RemoveWorktree(proj.LocalPath, wt.Path); err != nil {
			fmt.Printf("Warning: failed to remove worktree: %v\n", err)
		}
	}

	// Delete project directory (including bare repo)
	projectPath := filepath.Dir(proj.LocalPath)
	fmt.Printf("Removing project directory: %s\n", projectPath)
	if err := os.RemoveAll(projectPath); err != nil {
		return eris.Wrap(err, "failed to remove project directory")
	}

	// Delete from database
	if err := db.DeleteProject(database, proj.ID); err != nil {
		return eris.Wrap(err, "failed to delete project from database")
	}

	fmt.Printf("\nSuccessfully deleted project: %s\n", proj.Name)
	return nil
}

func deleteBranch(database *sql.DB, cfg *config.Config, proj *models.Project, branch string) error {
	// Get worktree
	worktree, err := db.GetWorktree(database, proj.ID, branch)
	if err != nil {
		if err == sql.ErrNoRows {
			return eris.Errorf("worktree for branch %s not found", branch)
		}
		return eris.Wrap(err, "failed to get worktree")
	}

	// Check if this is the main worktree
	if worktree.IsMain {
		return eris.Errorf("cannot delete main worktree, use --all to delete the entire project")
	}

	if !deleteForce {
		// Ask for confirmation
		fmt.Printf("This will delete worktree for branch '%s' and its associated session.\n", branch)
		fmt.Printf("Worktree path: %s\n", worktree.Path)
		fmt.Print("Are you sure? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Get and delete session
	sess, err := db.GetSessionByWorktree(database, worktree.ID)
	if err != nil && err != sql.ErrNoRows {
		return eris.Wrap(err, "failed to get session")
	}

	if sess != nil {
		// Kill session if it exists
		exists, err := sessionMgr.Exists(sess.TmuxSessionName)
		if err != nil {
			fmt.Printf("Warning: failed to check session existence: %v\n", err)
		} else if exists {
			fmt.Printf("Killing %s session: %s\n", sessionMgr.Name(), sess.TmuxSessionName)
			if err := sessionMgr.Delete(sess.TmuxSessionName); err != nil {
				fmt.Printf("Warning: failed to kill session: %v\n", err)
			}
		}

		// Delete session from database
		if err := db.DeleteSession(database, sess.ID); err != nil {
			return eris.Wrap(err, "failed to delete session from database")
		}
	}

	// Remove worktree
	fmt.Printf("Removing worktree: %s\n", worktree.Path)
	if err := git.RemoveWorktree(proj.LocalPath, worktree.Path); err != nil {
		return eris.Wrap(err, "failed to remove worktree")
	}

	// Delete worktree from database
	if err := db.DeleteWorktree(database, worktree.ID); err != nil {
		return eris.Wrap(err, "failed to delete worktree from database")
	}

	fmt.Printf("\nSuccessfully deleted worktree for branch: %s\n", branch)
	return nil
}
