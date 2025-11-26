package cmd

import (
	"encoding/json"
	"fmt"
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

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "status"},
	Short:   "List projects, worktrees, and sessions",
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
		fmt.Printf(
			"  %s Clone a repository with: %s\n",
			ui.Faint("→"),
			ui.Bold("sesh clone <remote-url>"),
		)
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

	// Print tree header
	fmt.Printf("\n%s\n", ui.Bold("Projects"))
	fmt.Println()

	for i, proj := range projects {
		// Get worktree count
		worktrees, err := state.DiscoverWorktrees(proj)
		if err != nil {
			// Skip projects with errors
			continue
		}

		isLast := i == len(projects)-1
		created := formatTimeAgo(proj.CreatedAt)

		// Print project node
		prefix := "├──"
		childPrefix := "│   "
		if isLast {
			prefix = "└──"
			childPrefix = "    "
		}

		fmt.Printf(
			"%s %s %s\n",
			ui.Faint(prefix),
			ui.Bold(proj.Name),
			ui.Faint(
				fmt.Sprintf(
					"(%d worktree%s, created %s)",
					len(worktrees),
					pluralize(len(worktrees)),
					created,
				),
			),
		)

		// Print worktrees as children
		for j, wt := range worktrees {
			isLastWorktree := j == len(worktrees)-1
			wtPrefix := "├──"
			if isLastWorktree {
				wtPrefix = "└──"
			}

			lastUsed := formatTimeAgo(wt.LastUsed)
			fmt.Printf("%s%s %s %s\n",
				ui.Faint(childPrefix),
				ui.Faint(wtPrefix),
				ui.Info(wt.Branch),
				ui.Faint(fmt.Sprintf("(last used %s)", lastUsed)),
			)
		}
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
		fmt.Printf(
			"  %s Clone a repository with: %s\n",
			ui.Faint("→"),
			ui.Bold("sesh clone <remote-url>"),
		)
		fmt.Printf(
			"  %s Or switch to a branch with: %s\n",
			ui.Faint("→"),
			ui.Bold("sesh switch <branch>"),
		)
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

	// Group sessions by project for tree rendering
	projectMap := make(map[string][]SessionDetail)
	var projectOrder []string
	projectSeen := make(map[string]bool)

	for _, sess := range sessions {
		if !projectSeen[sess.ProjectName] {
			projectOrder = append(projectOrder, sess.ProjectName)
			projectSeen[sess.ProjectName] = true
		}
		projectMap[sess.ProjectName] = append(projectMap[sess.ProjectName], sess)
	}

	// Print tree header
	fmt.Printf("\n%s\n", ui.Bold("Sessions"))
	fmt.Println()

	for i, projName := range projectOrder {
		isLastProject := i == len(projectOrder)-1
		projSessions := projectMap[projName]

		// Print project node
		prefix := "├──"
		childPrefix := "│   "
		if isLastProject {
			prefix = "└──"
			childPrefix = "    "
		}

		fmt.Printf("%s %s\n",
			ui.Faint(prefix),
			ui.Bold(projName),
		)

		// Print sessions/branches as children
		for j, sess := range projSessions {
			isLastSession := j == len(projSessions)-1
			sessPrefix := "├──"
			if isLastSession {
				sessPrefix = "└──"
			}

			statusIcon := ui.Faint("○")
			statusText := ui.Faint("stopped")
			if sess.IsRunning {
				statusIcon = ui.Success("●")
				statusText = ui.Success("running")
			}

			fmt.Printf("%s%s %s %s %s\n",
				ui.Faint(childPrefix),
				ui.Faint(sessPrefix),
				ui.Info(sess.Branch),
				statusIcon,
				statusText,
			)
		}
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

// pluralize returns "s" if count is not 1, otherwise empty string
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
