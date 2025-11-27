package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	deleteAll         bool
	deleteForce       bool
	deleteProjectName string
)

var deleteCmd = &cobra.Command{
	Use:     "delete [branch]",
	Aliases: []string{"del", "remove", "rm"},
	Short:   "Delete worktree, session, or entire project",
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
	deleteCmd.Flags().
		StringVarP(&deleteProjectName, "project", "p", "", "Specify project explicitly")
}

func runDelete(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Resolve project from filesystem state
	proj, err := project.ResolveProject(cfg.WorkspaceDir, deleteProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	if deleteAll {
		return deleteProject(cfg, proj, disp)
	}

	// Delete specific branch
	if len(args) == 0 {
		return eris.New("branch name required (or use --all to delete entire project)")
	}

	branch := args[0]
	return deleteBranch(cfg, proj, branch, disp)
}

func deleteProject(cfg *config.Config, proj *models.Project, disp display.Printer) error {
	// Get all worktrees for this project
	worktrees, err := state.DiscoverWorktrees(proj)
	if err != nil {
		return eris.Wrap(err, "failed to discover worktrees")
	}

	if !deleteForce {
		// Ask for confirmation
		disp.Printf(
			"This will delete project '%s' with %d worktree(s) and all associated sessions.\n",
			proj.Name,
			len(worktrees),
		)
		disp.Printf("Project path: %s\n", proj.LocalPath)
		disp.Print("Are you sure? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			disp.Println("Deletion cancelled.")
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
		// Generate session name
		sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)

		// Kill session if it exists
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			disp.Printf("Warning: failed to check session existence: %v\n", err)
		} else if exists {
			disp.Printf("Killing %s session: %s\n", sessionMgr.Name(), sessionName)
			if err := sessionMgr.Delete(sessionName); err != nil {
				disp.Printf("Warning: failed to kill session: %v\n", err)
			}
		}

		// Remove worktree
		disp.Printf("Removing worktree: %s\n", wt.Path)
		if err := git.RemoveWorktree(proj.LocalPath, wt.Path); err != nil {
			disp.Printf("Warning: failed to remove worktree: %v\n", err)
		}
	}

	// Delete project directory (including bare repo)
	projectPath := filepath.Dir(proj.LocalPath)
	disp.Printf("Removing project directory: %s\n", projectPath)
	if err := os.RemoveAll(projectPath); err != nil {
		return eris.Wrap(err, "failed to remove project directory")
	}

	disp.Printf("\nSuccessfully deleted project: %s\n", proj.Name)
	return nil
}

func deleteBranch(cfg *config.Config, proj *models.Project, branch string, disp display.Printer) error {
	// Get worktree from filesystem state
	worktree, err := state.GetWorktree(proj, branch)
	if err != nil {
		return eris.Errorf("worktree for branch %s not found", branch)
	}

	// Check if this is the main worktree
	if worktree.IsMain {
		return eris.Errorf("cannot delete main worktree, use --all to delete the entire project")
	}

	if !deleteForce {
		// Ask for confirmation
		disp.Printf(
			"This will delete worktree for branch '%s' and its associated session.\n",
			branch,
		)
		disp.Printf("Worktree path: %s\n", worktree.Path)
		disp.Print("Are you sure? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			disp.Println("Deletion cancelled.")
			return nil
		}
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Generate session name
	sessionName := workspace.GenerateSessionName(proj.Name, branch)

	// Kill session if it exists
	exists, err := sessionMgr.Exists(sessionName)
	if err != nil {
		disp.Printf("Warning: failed to check session existence: %v\n", err)
	} else if exists {
		disp.Printf("Killing %s session: %s\n", sessionMgr.Name(), sessionName)
		if err := sessionMgr.Delete(sessionName); err != nil {
			disp.Printf("Warning: failed to kill session: %v\n", err)
		}
	}

	// Remove worktree
	disp.Printf("Removing worktree: %s\n", worktree.Path)
	if err := git.RemoveWorktree(proj.LocalPath, worktree.Path); err != nil {
		return eris.Wrap(err, "failed to remove worktree")
	}

	disp.Printf("\nSuccessfully deleted worktree for branch: %s\n", branch)
	return nil
}
