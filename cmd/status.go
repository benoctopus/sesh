package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current session and project information",
	Long: `Display information about the current session and project.

Shows details about:
- Current session (if inside one)
- Current project and branch
- Worktree path
- Git status summary
- Other available sessions for this project

Examples:
  sesh status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Get current session
	currentSessionName, err := sessionMgr.GetCurrentSessionName()
	if err != nil && !eris.Is(err, session.ErrNotInSession) {
		return eris.Wrap(err, "failed to get current session name")
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Try to resolve project from CWD
	proj, err := project.ResolveProject(database, "", cwd)
	if err != nil {
		// Not in a project directory
		if currentSessionName != "" {
			fmt.Printf("Current Session: %s\n", currentSessionName)
			fmt.Println("\nNot currently in a sesh-managed project directory.")
			return nil
		}
		fmt.Println("Not currently in a sesh-managed project directory.")
		fmt.Println("Not inside a session.")
		fmt.Println("\nUse 'sesh list' to see all available sessions.")
		return nil
	}

	// Display project information
	fmt.Printf("Project: %s\n", proj.Name)
	fmt.Printf("Remote: %s\n", proj.RemoteURL)

	// Get current branch
	gitRoot, err := project.FindGitRoot(cwd)
	if err == nil {
		branch, err := git.GetCurrentBranch(gitRoot)
		if err == nil {
			fmt.Printf("Branch: %s\n", branch)
		}

		// Get worktree info from database
		worktree, err := db.GetWorktree(database, proj.ID, branch)
		if err == nil {
			fmt.Printf("Worktree: %s\n", worktree.Path)
			fmt.Printf("Last Used: %s\n", formatTimeAgo(worktree.LastUsed))

			// Get session info
			sess, err := db.GetSessionByWorktree(database, worktree.ID)
			if err == nil && sess != nil {
				fmt.Printf("Session: %s\n", sess.TmuxSessionName)

				// Check if session is running
				exists, err := sessionMgr.Exists(sess.TmuxSessionName)
				if err == nil {
					if exists {
						fmt.Printf("Session Status: Running\n")
					} else {
						fmt.Printf("Session Status: Not Running\n")
					}
				}

				if currentSessionName != "" && currentSessionName == sess.TmuxSessionName {
					fmt.Printf("(You are currently in this session)\n")
				}
			}
		}

		// Get git status summary
		fmt.Println()
		gitStatus, err := getGitStatusSummary(gitRoot)
		if err == nil {
			fmt.Printf("Git Status: %s\n", gitStatus)
		}
	}

	// List other sessions for this project
	worktrees, err := db.GetWorktreesByProject(database, proj.ID)
	if err != nil {
		return eris.Wrap(err, "failed to get worktrees")
	}

	if len(worktrees) > 1 {
		fmt.Println("\nOther Sessions:")
		for _, wt := range worktrees {
			// Skip current worktree
			if gitRoot != "" {
				currentBranch, err := git.GetCurrentBranch(gitRoot)
				if err == nil && wt.Branch == currentBranch {
					continue
				}
			}

			sess, err := db.GetSessionByWorktree(database, wt.ID)
			if err != nil || sess == nil {
				continue
			}

			// Check if session is running
			status := ""
			exists, err := sessionMgr.Exists(sess.TmuxSessionName)
			if err == nil {
				if exists {
					status = " (running)"
				} else {
					status = " (not running)"
				}
			}

			fmt.Printf("  %s%s - last used %s\n",
				sess.TmuxSessionName,
				status,
				formatTimeAgo(wt.LastUsed),
			)
		}
	}

	return nil
}

// getGitStatusSummary returns a summary of the git status
func getGitStatusSummary(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get git status")
	}

	if len(output) == 0 {
		return "clean", nil
	}

	// Count different types of changes
	modified := 0
	untracked := 0
	staged := 0

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		status := line[:2]
		switch {
		case status == "??":
			untracked++
		case status[0] != ' ' && status[0] != '?':
			staged++
		case status[1] != ' ' && status[1] != '?':
			modified++
		}
	}

	var parts []string
	if staged > 0 {
		parts = append(parts, fmt.Sprintf("%d staged", staged))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}

	if len(parts) == 0 {
		return "clean", nil
	}

	return strings.Join(parts, ", "), nil
}
