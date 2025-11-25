package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/ui"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	listProjects bool
	listSessions bool
	listJSON     bool
)

// BUG: Fix list output such that it does not contain an entry with an empty branch, presumably representing
// the bare project with no checkout.

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects, worktrees, and sessions",
	Long: `Display all projects, worktrees, and sessions in the workspace.

By default, shows all sessions with their project and branch information.

Examples:
  sesh list                    # List all sessions
  sesh list --projects         # List only projects
  sesh list --sessions         # List only sessions
  sesh list --json             # Output in JSON format`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listProjects, "projects", false, "Show only projects")
	listCmd.Flags().BoolVar(&listSessions, "sessions", false, "Show only sessions (default)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	if listProjects {
		return listAllProjects(cfg)
	}

	// Default: list sessions
	return listAllSessions(cfg)
}

func listAllProjects(cfg *config.Config) error {
	// Discover all projects from filesystem
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return eris.Wrap(err, "failed to discover projects")
	}

	if len(projects) == 0 {
		fmt.Printf("%s No projects found.\n", ui.Info("ℹ"))
		fmt.Printf("  %s Clone a repository with: %s\n", ui.Faint("→"), ui.Bold("sesh clone <remote-url>"))
		return nil
	}

	if listJSON {
		data, err := json.MarshalIndent(projects, "", "  ")
		if err != nil {
			return eris.Wrap(err, "failed to marshal projects to JSON")
		}
		fmt.Println(string(data))
		return nil
	}

	// Print table header
	fmt.Printf("\n%s\n", ui.Bold("Projects"))
	fmt.Printf("%-50s %-12s %-20s\n", ui.Faint("PROJECT"), ui.Faint("WORKTREES"), ui.Faint("CREATED"))
	fmt.Println(strings.Repeat("─", 85))

	for _, proj := range projects {
		// Get worktree count
		worktrees, err := state.DiscoverWorktrees(proj)
		if err != nil {
			// Skip projects with errors
			continue
		}

		created := formatTimeAgo(proj.CreatedAt)

		fmt.Printf("%-50s %s%-12d%s %s\n",
			truncate(proj.Name, 50),
			ui.Info(""),
			len(worktrees),
			ui.Info(""),
			ui.Faint(created),
		)
	}
	fmt.Println()

	return nil
}

func listAllSessions(cfg *config.Config) error {
	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Get all running sessions
	runningSessions, err := state.DiscoverSessions(sessionMgr)
	if err != nil {
		return eris.Wrap(err, "failed to discover sessions")
	}

	// Discover all projects and worktrees
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return eris.Wrap(err, "failed to discover projects")
	}

	type SessionDetail struct {
		SessionName  string
		ProjectName  string
		Branch       string
		WorktreePath string
		LastUsed     time.Time
		IsRunning    bool
	}

	var sessions []SessionDetail

	// Build session details by matching worktrees to running sessions
	for _, proj := range projects {
		worktrees, err := state.DiscoverWorktrees(proj)
		if err != nil {
			continue
		}

		for _, wt := range worktrees {
			// Generate expected session name
			sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)

			// Check if this session is running
			isRunning := false
			for _, runningSess := range runningSessions {
				if runningSess == sessionName {
					isRunning = true
					break
				}
			}

			sessions = append(sessions, SessionDetail{
				SessionName:  sessionName,
				ProjectName:  proj.Name,
				Branch:       wt.Branch,
				WorktreePath: wt.Path,
				LastUsed:     wt.LastUsed,
				IsRunning:    isRunning,
			})
		}
	}

	if len(sessions) == 0 {
		fmt.Printf("%s No worktrees found.\n", ui.Info("ℹ"))
		fmt.Printf("  %s Clone a repository with: %s\n", ui.Faint("→"), ui.Bold("sesh clone <remote-url>"))
		fmt.Printf("  %s Or switch to a branch with: %s\n", ui.Faint("→"), ui.Bold("sesh switch <branch>"))
		return nil
	}

	if listJSON {
		data, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			return eris.Wrap(err, "failed to marshal sessions to JSON")
		}
		fmt.Println(string(data))
		return nil
	}

	// Print table header
	fmt.Printf("\n%s\n", ui.Bold("Sessions"))
	fmt.Printf("%-30s %-20s %-30s %-10s\n", ui.Faint("PROJECT"), ui.Faint("BRANCH"), ui.Faint("SESSION NAME"), ui.Faint("STATUS"))
	fmt.Println(strings.Repeat("─", 95))

	for _, sess := range sessions {
		statusText := ui.Faint("stopped")
		statusIcon := "○"
		if sess.IsRunning {
			statusText = ui.Success("running")
			statusIcon = ui.Success("●")
		}

		fmt.Printf("%-30s %-20s %-30s %s %s\n",
			truncate(sess.ProjectName, 30),
			ui.Info(truncate(sess.Branch, 20)),
			ui.Faint(truncate(sess.SessionName, 30)),
			statusIcon,
			statusText,
		)
	}
	fmt.Println()

	return nil
}

// formatTimeAgo formats a time as a human-readable "time ago" string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	years := int(duration.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
