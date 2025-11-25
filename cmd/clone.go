package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/ui"
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

	// Get project path in workspace
	projectPath := workspace.GetProjectPath(cfg.WorkspaceDir, projectName)

	// Clone repository as bare repo
	bareRepoPath := filepath.Join(projectPath, ".git")
	fmt.Printf("%s Cloning %s\n", ui.Info("⬇"), ui.Bold(remoteURL))
	fmt.Printf("  %s %s\n", ui.Faint("→"), projectPath)
	if err := git.Clone(remoteURL, bareRepoPath); err != nil {
		return eris.Wrap(err, "failed to clone repository")
	}

	// Get default branch
	defaultBranch, err := git.GetDefaultBranch(bareRepoPath)
	if err != nil {
		return eris.Wrap(err, "failed to get default branch")
	}

	// Create main worktree
	worktreePath := workspace.GetWorktreePath(projectPath, defaultBranch)
	fmt.Printf("%s Creating worktree for branch %s\n", ui.Info("✨"), ui.Bold(defaultBranch))
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
	fmt.Printf("%s Creating %s session %s\n", ui.Info("✨"), sessionMgr.Name(), ui.Bold(sessionName))
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	fmt.Printf("\n%s Successfully cloned %s\n", ui.Success("✓"), ui.Bold(projectName))
	fmt.Printf("  %s %s\n", ui.Faint("Worktree:"), worktreePath)
	fmt.Printf("  %s %s\n", ui.Faint("Session:"), sessionName)
	fmt.Printf("\n%s Attaching to session...\n", ui.Info("→"))

	// Execute startup command if configured
	startupCmd, err := config.GetStartupCommand(worktreePath)
	if err == nil && startupCmd != "" && sessionMgr.Name() == "tmux" {
		fmt.Printf("%s Running startup command: %s\n", ui.Info("⚙"), ui.Faint(startupCmd))
		if tmuxMgr, ok := sessionMgr.(*session.TmuxManager); ok {
			if err := tmuxMgr.SendKeys(sessionName, startupCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to run startup command: %v\n", err)
			}
		}
	}

	// Attach to the new session
	if err := sessionMgr.Attach(sessionName); err != nil {
		return eris.Wrap(err, "failed to attach to session")
	}

	return nil
}
