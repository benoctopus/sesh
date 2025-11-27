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
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <session-name>",
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
  sesh info myproject-main         # Show info for a session
  sesh list --plain | fzf --preview 'sesh info {}'  # Use in fzf preview`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	sessionName := args[0]

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

	// Parse session name to get project and branch
	// Session names are in format: project-branch
	parts := strings.SplitN(sessionName, "-", 2)
	if len(parts) != 2 {
		return eris.Errorf("invalid session name format: %s (expected project-branch)", sessionName)
	}

	projectName := parts[0]
	branchName := parts[1]

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
	for _, wt := range worktrees {
		if wt.Branch == branchName {
			worktreePath = wt.Path
			break
		}
	}

	if worktreePath == "" {
		return eris.Errorf("worktree not found for branch: %s", branchName)
	}

	// Check if session is running
	isRunning, err := sessionMgr.Exists(sessionName)
	if err != nil {
		return eris.Wrap(err, "failed to check session status")
	}

	// Get git status
	gitStatus := getGitStatus(worktreePath)
	lastCommit := getLastCommit(worktreePath)

	// Display using stdout (for fzf preview)
	disp := display.NewStdout()

	disp.Printf("\n")
	disp.Printf("%s %s\n", disp.Bold("Session:"), sessionName)
	disp.Printf("%s %s\n", disp.Bold("Project:"), projectName)
	disp.Printf("%s %s\n", disp.Bold("Branch:"), branchName)
	disp.Printf("%s %s\n", disp.Bold("Path:"), worktreePath)

	if isRunning {
		disp.Printf("%s %s\n", disp.Bold("Status:"), disp.SuccessText("● Running"))
	} else {
		disp.Printf("%s %s\n", disp.Bold("Status:"), disp.Faint("○ Stopped"))
	}

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
