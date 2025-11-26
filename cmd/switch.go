package cmd

import (
	"fmt"
	"os"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/fuzzy"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/tty"
	"github.com/benoctopus/sesh/internal/ui"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	switchProjectName    string
	switchStartupCommand string
)

var switchCmd = &cobra.Command{
	Use:   "switch [branch]",
	Short: "Switch to a branch (create worktree if needed)",
	Long: `Switch to a branch, creating a worktree and session if they don't exist.
If no branch is specified, an interactive fuzzy finder will show all available branches.

The project is automatically detected from the current working directory,
or can be specified explicitly with the --project flag.

If the branch doesn't exist locally or remotely, a new branch will be created automatically.

Examples:
  sesh switch feature-foo                        # Switch to existing branch
  sesh switch new-feature                        # Create new branch automatically
  sesh switch                                    # Interactive fuzzy branch selection
  sesh switch --project myproject feature-bar    # Explicit project
  sesh switch -c "direnv allow" feature-baz      # Run startup command`,
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
	switchCmd.Flags().
		StringVarP(&switchProjectName, "project", "p", "", "Specify project explicitly")
	switchCmd.Flags().
		StringVarP(&switchStartupCommand, "command", "c", "", "Command to run after switching to session")
}

func runSwitch(cmd *cobra.Command, args []string) error {
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
	proj, err := project.ResolveProject(cfg.WorkspaceDir, switchProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	var branch string
	if len(args) > 0 {
		branch = args[0]
	} else {
		// No branch specified
		if !tty.IsInteractive() {
			return eris.New("branch argument required in noninteractive mode (usage: sesh switch <branch>)")
		}

		// Use streaming fuzzy finder in interactive mode
		// Start git fetch in background - don't wait for it
		go func() {
			if err := git.Fetch(proj.LocalPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: git fetch failed: %s\n", eris.ToString(err, true))
			}
		}()

		// Stream branches directly from git to fzf for instant UI
		branchReader, err := git.StreamRemoteBranches(cmd.Context(), proj.LocalPath)
		if err != nil {
			return eris.Wrap(err, "failed to start branch listing")
		}

		selectedBranch, err := fuzzy.SelectBranchFromReader(branchReader)
		if err != nil {
			return eris.Wrap(err, "failed to select branch")
		}

		branch = selectedBranch
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Check if worktree already exists in filesystem
	existingWorktree, err := state.GetWorktree(proj, branch)
	if err == nil && existingWorktree != nil {
		// Worktree exists, attach to existing or create new session
		fmt.Printf("%s %s\n", ui.Info("→"), ui.Bold(fmt.Sprintf("Switching to existing worktree: %s", existingWorktree.Path)))

		// Generate session name
		sessionName := workspace.GenerateSessionName(proj.Name, branch)

		// Check if session is running
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return eris.Wrap(err, "failed to check session existence")
		}

		if exists {
			// In noninteractive mode, don't attach
			if !tty.IsInteractive() {
				fmt.Printf("%s Session %s already exists\n", ui.Success("✓"), ui.Bold(sessionName))
				return nil
			}
			// Attach to existing session
			return sessionMgr.Attach(sessionName)
		}

		// Session doesn't exist, create it
		fmt.Printf("%s Creating %s session %s\n", ui.Info("✨"), sessionMgr.Name(), ui.Bold(sessionName))
		if err := sessionMgr.Create(sessionName, existingWorktree.Path); err != nil {
			return eris.Wrap(err, "failed to create session")
		}

		// Execute startup command if configured
		startupCmd := getStartupCommand(cfg, existingWorktree.Path)
		if startupCmd != "" && sessionMgr.Name() == "tmux" {
			fmt.Printf("%s Running startup command: %s\n", ui.Info("⚙"), ui.Faint(startupCmd))
			if tmuxMgr, ok := sessionMgr.(*session.TmuxManager); ok {
				if err := tmuxMgr.SendKeys(sessionName, startupCmd); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to run startup command: %v\n", err)
				}
			}
		}

		// In noninteractive mode, don't attach
		if !tty.IsInteractive() {
			fmt.Printf("%s Session %s created successfully\n", ui.Success("✓"), ui.Bold(sessionName))
			return nil
		}

		return sessionMgr.Attach(sessionName)
	}

	// Worktree doesn't exist, check branch existence
	exists, remote, err := git.DoesBranchExist(proj.LocalPath, branch)
	if err != nil {
		return eris.Wrap(err, "failed to check branch existence")
	}

	// Get worktree path
	projectPath := workspace.GetProjectPath(cfg.WorkspaceDir, proj.Name)
	worktreePath := workspace.GetWorktreePath(projectPath, branch)

	// Create worktree based on branch state
	if remote {
		// Branch exists remotely, create worktree from remote
		fmt.Printf("%s Creating worktree for remote branch: %s\n", ui.Info("✨"), ui.Bold(branch))
		if err := git.CreateWorktree(proj.LocalPath, branch, worktreePath); err != nil {
			return eris.Wrap(err, "failed to create worktree from remote branch")
		}
	} else if exists {
		// Branch exists locally, create worktree from local
		fmt.Printf("%s Creating worktree for local branch: %s\n", ui.Info("✨"), ui.Bold(branch))
		if err := git.CreateWorktree(proj.LocalPath, branch, worktreePath); err != nil {
			return eris.Wrap(err, "failed to create worktree from local branch")
		}
	} else {
		// Branch doesn't exist, create new branch and worktree
		fmt.Printf("%s Creating new branch and worktree: %s\n", ui.Success("✨"), ui.Bold(branch))
		if err := git.CreateWorktreeNewBranch(proj.LocalPath, branch, worktreePath, "HEAD"); err != nil {
			return eris.Wrap(err, "failed to create worktree with new branch")
		}
	}

	// Create session
	sessionName := workspace.GenerateSessionName(proj.Name, branch)
	fmt.Printf("%s Creating %s session %s\n", ui.Info("✨"), sessionMgr.Name(), ui.Bold(sessionName))
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	fmt.Printf("\n%s Successfully switched to %s\n", ui.Success("✓"), ui.Bold(branch))
	fmt.Printf("  %s %s\n", ui.Faint("Worktree:"), worktreePath)
	fmt.Printf("  %s %s\n", ui.Faint("Session:"), sessionName)

	// Execute startup command if configured
	startupCmd := getStartupCommand(cfg, worktreePath)
	if startupCmd != "" && sessionMgr.Name() == "tmux" {
		fmt.Printf("%s Running startup command: %s\n", ui.Info("⚙"), ui.Faint(startupCmd))
		if tmuxMgr, ok := sessionMgr.(*session.TmuxManager); ok {
			if err := tmuxMgr.SendKeys(sessionName, startupCmd); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to run startup command: %v\n", err)
			}
		}
	}

	// In noninteractive mode, don't attach
	if !tty.IsInteractive() {
		return nil
	}

	// Attach to session
	fmt.Printf("\n%s Attaching to session...\n", ui.Info("→"))
	return sessionMgr.Attach(sessionName)
}

// getStartupCommand returns the startup command following the priority hierarchy:
// 1. Command-line flag (highest priority)
// 2. Per-project config (.sesh.yaml in worktree)
// 3. Global config
// 4. Empty string (no command)
func getStartupCommand(cfg *config.Config, worktreePath string) string {
	// 1. Check command-line flag
	if switchStartupCommand != "" {
		return switchStartupCommand
	}

	// 2. Check per-project config
	startupCmd, err := config.GetStartupCommand(worktreePath)
	if err == nil && startupCmd != "" {
		return startupCmd
	}

	// 3. Return global config (already loaded in cfg)
	return cfg.StartupCommand
}
