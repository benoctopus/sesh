package cmd

import (
	"fmt"
	"os"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var cloneDetach bool

var cloneCmd = &cobra.Command{
	Use:     "clone <remote-url>",
	Aliases: []string{"cl"},
	Short:   "Clone a git repository into the workspace folder",
	Long: `Clone a git repository into the workspace folder as a bare repo,
create the main worktree, and set up a session.

Examples:
  sesh clone git@github.com:user/repo.git
  sesh clone https://github.com/user/repo.git
  sesh clone -d https://github.com/user/repo.git     # Clone without attaching`,
	Args: cobra.ExactArgs(1),
	RunE: runClone,
}

func init() {
	rootCmd.AddCommand(cloneCmd)
	cloneCmd.Flags().
		BoolVarP(&cloneDetach, "detach", "d", false, "Create session without attaching to it")
}

func runClone(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()
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

	// Generate project name from remote URL
	projectName, err := git.GenerateProjectName(remoteURL)
	if err != nil {
		return eris.Wrap(err, "failed to generate project name from remote URL")
	}

	// Check if project already exists by checking filesystem
	existingProject, err := state.GetProject(cfg.WorkspaceDir, projectName)
	if err == nil && existingProject != nil {
		return eris.Errorf("project %s already exists in workspace", projectName)
	}

	// Get paths for bare repo and worktrees
	bareRepoPath := workspace.GetBareRepoPath(cfg.WorkspaceDir, projectName)
	worktreeBasePath := workspace.GetWorktreeBasePath(cfg.WorkspaceDir, projectName)

	// Clone repository as bare repo
	disp.Infof("Cloning %s", disp.Bold(remoteURL))
	disp.Printf("  %s %s\n", disp.Faint("→"), bareRepoPath)
	if err := git.Clone(remoteURL, bareRepoPath); err != nil {
		return eris.Wrap(err, "failed to clone repository")
	}

	// Get default branch
	defaultBranch, err := git.GetDefaultBranch(bareRepoPath)
	if err != nil {
		return eris.Wrap(err, "failed to get default branch")
	}

	// Create main worktree
	worktreePath := workspace.GetWorktreePath(worktreeBasePath, defaultBranch)
	disp.Infof("Creating worktree for branch %s", disp.Bold(defaultBranch))
	if err := git.CreateWorktree(bareRepoPath, defaultBranch, worktreePath); err != nil {
		return eris.Wrap(err, "failed to clone worktree")
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Generate session name
	sessionName := workspace.GenerateSessionName(projectName, defaultBranch)

	// Create session
	disp.Infof("Creating %s session %s", sessionMgr.Name(), disp.Bold(sessionName))
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	disp.Successf("Successfully cloned %s", disp.Bold(projectName))
	disp.Printf("  %s %s\n", disp.Faint("Worktree:"), worktreePath)
	disp.Printf("  %s %s\n", disp.Faint("Session:"), sessionName)

	// Execute startup command if configured
	startupCmd, err := config.GetStartupCommand(worktreePath)
	if err == nil && startupCmd != "" && sessionMgr.Name() == "tmux" {
		disp.Infof("Running startup command: %s", disp.Faint(startupCmd))
		if tmuxMgr, ok := sessionMgr.(*session.TmuxManager); ok {
			if err := tmuxMgr.SendKeys(sessionName, startupCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to run startup command: %v\n", err)
			}
		}
	}

	// Attach to the new session if not detached
	if !cloneDetach {
		disp.Infof("Attaching to session...")
		if err := sessionMgr.Attach(sessionName); err != nil {
			return eris.Wrap(err, "failed to attach to session")
		}
	}

	return nil
}
