package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var infoProjectName string

var infoCmd = &cobra.Command{
	Use:   "info <session-name-or-branch>",
	Short: "Show detailed information about a session",
	Long: `Display detailed information about a session for use in fzf preview panes.

This command shows:
- Project name and branch
- Session status (running/stopped)
- Git status summary
- Last commit message
- Last used time
- Worktree path

Examples:
  sesh info myproject-main                     # Show info for a session
  sesh info --project myproject feature-branch # Show info for project and branch
  sesh list --plain | fzf --preview 'sesh info {}'  # Use in fzf preview`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&infoProjectName, "project", "p", "", "Project name (when passing branch as argument)")
}

func runInfo(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	var projectName, branchName, sessionName string

	// Handle --project flag mode
	if infoProjectName != "" {
		// In this mode, args[0] is the branch name
		branchName = args[0]
		projectName = infoProjectName

		// Generate session name using the same logic as the switch command
		sessionName = workspace.GenerateSessionName(projectName, branchName)
	} else {
		// Original mode: args[0] is the session name
		sessionName = args[0]

		// Parse session name to get project and branch
		// Session names are in format: project-branch
		parts := strings.SplitN(sessionName, "-", 2)
		if len(parts) != 2 {
			return eris.Errorf("invalid session name format: %s (expected project-branch)", sessionName)
		}

		projectName = parts[0]
		branchName = parts[1]
	}

	// Resolve project
	proj, err := project.ResolveProject(cfg.WorkspaceDir, projectName, "")
	if err != nil {
		return eris.Wrapf(err, "failed to resolve project: %s", projectName)
	}

	// Find the worktree for this branch
	worktrees, err := state.DiscoverWorktrees(proj)
	if err != nil {
		return eris.Wrap(err, "failed to discover worktrees")
	}

	var worktreePath string
	var worktreeExists bool
	for _, wt := range worktrees {
		if wt.Branch == branchName {
			worktreePath = wt.Path
			worktreeExists = true
			break
		}
	}

	// Display using stdout (for fzf preview)
	disp := display.NewStdout()

	disp.Printf("\n")
	disp.Printf("%s %s\n", disp.Bold("Session:"), sessionName)
	disp.Printf("%s %s\n", disp.Bold("Project:"), projectName)
	disp.Printf("%s %s\n", disp.Bold("Branch:"), branchName)

	if worktreeExists {
		// Worktree exists - show full information
		disp.Printf("%s %s\n", disp.Bold("Path:"), worktreePath)

		// Check if session is running
		isRunning, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return eris.Wrap(err, "failed to check session status")
		}

		if isRunning {
			disp.Printf("%s %s\n", disp.Bold("Status:"), disp.SuccessText("● Running"))
		} else {
			disp.Printf("%s %s\n", disp.Bold("Status:"), disp.Faint("○ Stopped"))
		}

		// Get git status
		gitStatus := getGitStatus(worktreePath)
		lastCommit := getLastCommit(worktreePath)

		disp.Printf("\n")
		disp.Printf("%s\n", disp.Bold("Git Status:"))
		if gitStatus != "" {
			disp.Print(gitStatus)
		} else {
			disp.Printf("%s\n", disp.Faint("  (clean)"))
		}

		disp.Printf("\n")
		disp.Printf("%s\n", disp.Bold("Last Commit:"))
		if lastCommit != "" {
			disp.Print(lastCommit)
		} else {
			disp.Printf("%s\n", disp.Faint("  (no commits)"))
		}
	} else {
		// Worktree doesn't exist - show remote branch information
		disp.Printf("%s %s\n", disp.Bold("Status:"), disp.Faint("○ Remote branch (no local worktree)"))

		// Try to get information about the remote branch
		lastCommit := getRemoteBranchLastCommit(proj.LocalPath, branchName)

		disp.Printf("\n")
		disp.Printf("%s\n", disp.Bold("Remote Branch Info:"))
		if lastCommit != "" {
			disp.Print(lastCommit)
		} else {
			disp.Printf("%s\n", disp.Faint("  (no commit information available)"))
		}

		disp.Printf("\n")
		disp.Printf("%s\n", disp.Faint("Run 'sesh switch' to create a worktree for this branch."))
	}

	return nil
}

// getGitStatus returns a formatted git status summary
func getGitStatus(worktreePath string) string {
	cmd := exec.Command("git", "-C", worktreePath, "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	statusStr := strings.TrimSpace(string(output))
	if statusStr == "" {
		return ""
	}

	// Indent each line
	lines := strings.Split(statusStr, "\n")
	var result strings.Builder
	for _, line := range lines {
		result.WriteString("  ")
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// getLastCommit returns the last commit message
func getLastCommit(worktreePath string) string {
	// Check if .git exists (to avoid errors in bare repos or new worktrees)
	gitPath := filepath.Join(worktreePath, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return ""
	}

	cmd := exec.Command("git", "-C", worktreePath, "log", "-1", "--pretty=format:%h %s (%ar)")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	commitStr := strings.TrimSpace(string(output))
	if commitStr == "" {
		return ""
	}

	return "  " + commitStr + "\n"
}

// getRemoteBranchLastCommit returns the last commit message for a remote branch
func getRemoteBranchLastCommit(repoPath, branch string) string {
	// Try to get the last commit from the remote branch
	// Use origin/<branch> format
	remoteBranch := "origin/" + branch

	cmd := exec.Command("git", "-C", repoPath, "log", "-1", remoteBranch, "--pretty=format:%h %s (%ar)")
	output, err := cmd.Output()
	if err != nil {
		// Try without origin/ prefix in case it's a different remote format
		cmd = exec.Command("git", "-C", repoPath, "log", "-1", branch, "--pretty=format:%h %s (%ar)")
		output, err = cmd.Output()
		if err != nil {
			return ""
		}
	}

	commitStr := strings.TrimSpace(string(output))
	if commitStr == "" {
		return ""
	}

	return "  " + commitStr + "\n"
}
