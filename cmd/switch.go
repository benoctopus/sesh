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
	switchCmd.Flags().
		BoolVarP(&switchCreateBranch, "create", "b", false, "Create a new branch (like git checkout -b)")
	switchCmd.Flags().
		StringVarP(&switchProjectName, "project", "p", "", "Specify project explicitly")
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
		// No branch specified, use streaming fuzzy finder
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
		fmt.Printf("Switching to existing worktree: %s\n", existingWorktree.Path)

		// Generate session name
		sessionName := workspace.GenerateSessionName(proj.Name, branch)

		// Check if session is running
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return eris.Wrap(err, "failed to check session existence")
		}

		if exists {
			// Attach to existing session
			return sessionMgr.Attach(sessionName)
		}

		// Session doesn't exist, create it
		fmt.Printf("Creating %s session %s...\n", sessionMgr.Name(), sessionName)
		if err := sessionMgr.Create(sessionName, existingWorktree.Path); err != nil {
			return eris.Wrap(err, "failed to create session")
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
		exists, err := git.DoesBranchExist(proj.LocalPath, branch)
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

	// Create session
	sessionName := workspace.GenerateSessionName(proj.Name, branch)
	fmt.Printf("Creating %s session %s...\n", sessionMgr.Name(), sessionName)
	if err := sessionMgr.Create(sessionName, worktreePath); err != nil {
		return eris.Wrap(err, "failed to create session")
	}

	fmt.Printf("\nSuccessfully switched to %s\n", branch)
	fmt.Printf("Worktree: %s\n", worktreePath)
	fmt.Printf("Session: %s\n", sessionName)
	fmt.Printf("\nAttaching to session...\n")

	// Attach to session
	return sessionMgr.Attach(sessionName)
}
