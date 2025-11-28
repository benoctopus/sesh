package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/pr"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	infoProjectName string
	infoPRMode      bool
)

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
  sesh info --pr "#123│Title│..."              # Show info for a pull request
  sesh list --plain | fzf --preview 'sesh info {}'  # Use in fzf preview`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().StringVarP(&infoProjectName, "project", "p", "", "Project name (when passing branch as argument)")
	infoCmd.Flags().BoolVar(&infoPRMode, "pr", false, "Show pull request info instead of session info")
}

func runInfo(cmd *cobra.Command, args []string) error {
	// Handle PR mode
	if infoPRMode {
		return runPRInfo(cmd, args)
	}

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
	disp.Printf("%s %s\n", disp.InfoText("Session:"), disp.Bold(sessionName))
	disp.Printf("%s %s\n", disp.InfoText("Project:"), projectName)
	disp.Printf("%s %s\n", disp.InfoText("Branch:"), disp.Bold(branchName))

	if worktreeExists {
		// Worktree exists - show full information
		disp.Printf("%s %s\n", disp.InfoText("Path:"), disp.Faint(worktreePath))

		// Check if session is running
		isRunning, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return eris.Wrap(err, "failed to check session status")
		}

		if isRunning {
			disp.Printf("%s %s\n", disp.InfoText("Status:"), disp.SuccessText("● Running"))
		} else {
			disp.Printf("%s %s\n", disp.InfoText("Status:"), disp.Faint("○ Stopped"))
		}

		// Get git status
		gitStatus := getGitStatus(worktreePath)
		lastCommit := getLastCommit(worktreePath)

		disp.Printf("\n")
		disp.Printf("%s\n", disp.Bold("Git Status:"))
		if gitStatus != "" {
			disp.Print(gitStatus)
		} else {
			disp.Printf("%s\n", disp.SuccessText("  ✓ Clean"))
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
		disp.Printf("%s %s\n", disp.InfoText("Status:"), disp.WarningText("○ Remote branch (no local worktree)"))

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
		disp.Printf("%s\n", disp.InfoText("→ Run 'sesh switch' to create a worktree for this branch."))
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

// runPRInfo displays detailed information about a pull request
func runPRInfo(cmd *cobra.Command, args []string) error {
	prSelection := args[0]

	// Parse PR number from selection
	prNum, err := pr.ParsePRNumber(prSelection)
	if err != nil {
		return eris.Wrap(err, "failed to parse PR number")
	}

	// Load configuration to get workspace
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Resolve project
	proj, err := project.ResolveProject(cfg.WorkspaceDir, "", cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	// Get remote URL
	remoteURL, err := git.GetRemoteURL(proj.LocalPath)
	if err != nil {
		return eris.Wrap(err, "failed to get remote URL")
	}

	// Create PR provider
	provider, err := pr.NewProvider(remoteURL)
	if err != nil {
		return eris.Wrap(err, "failed to create PR provider")
	}

	// Get PR details
	pullRequest, err := provider.GetPR(cmd.Context(), proj.LocalPath, prNum)
	if err != nil {
		return eris.Wrap(err, "failed to get pull request details")
	}

	// Display PR information
	disp := display.NewStdout()

	disp.Printf("\n")
	disp.Printf("%s %s\n", disp.InfoText("PR:"), disp.Bold(fmt.Sprintf("#%d", pullRequest.Number)))
	disp.Printf("%s %s\n", disp.InfoText("Title:"), disp.Bold(pullRequest.Title))
	disp.Printf("%s %s\n", disp.InfoText("Author:"), pullRequest.Author)
	disp.Printf("%s %s → %s\n", disp.InfoText("Branch:"), disp.Bold(pullRequest.Branch), pullRequest.BaseBranch)
	disp.Printf("%s %s\n", disp.InfoText("State:"), getPRStateDisplay(pullRequest.State, disp))

	if len(pullRequest.Labels) > 0 {
		disp.Printf("%s %s\n", disp.InfoText("Labels:"), strings.Join(pullRequest.Labels, ", "))
	}

	disp.Printf("\n")
	if pullRequest.Description != "" {
		disp.Printf("%s\n", disp.Bold("Description:"))
		// Show first few lines of description
		lines := strings.Split(pullRequest.Description, "\n")
		maxLines := 5
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			for _, line := range lines {
				disp.Printf("  %s\n", line)
			}
			disp.Printf("  %s\n", disp.Faint("..."))
		} else {
			for _, line := range lines {
				disp.Printf("  %s\n", line)
			}
		}
	}

	disp.Printf("\n")
	disp.Printf("%s\n", disp.Faint(pullRequest.URL))

	return nil
}

// getPRStateDisplay returns a colorized state display
func getPRStateDisplay(state string, disp display.Printer) string {
	switch strings.ToLower(state) {
	case "open":
		return disp.SuccessText("● Open")
	case "closed":
		return disp.ErrorText("● Closed")
	case "merged":
		return disp.InfoText("● Merged")
	default:
		return disp.Faint("○ " + state)
	}
}
