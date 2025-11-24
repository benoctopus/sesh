package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	listProjects bool
	listSessions bool
	listJSON     bool
)

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
	// Ensure config directory exists (needed for database)
	if err := config.EnsureConfigDir(); err != nil {
		return eris.Wrap(err, "failed to ensure config directory")
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

	if listProjects {
		return listAllProjects(database)
	}

	// Default: list sessions
	return listAllSessions(database)
}

func listAllProjects(database *sql.DB) error {
	projects, err := db.GetAllProjects(database)
	if err != nil {
		return eris.Wrap(err, "failed to get projects")
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		fmt.Println("Clone a repository with: sesh clone <remote-url>")
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
	fmt.Printf("%-50s %-12s %-20s\n", "PROJECT", "WORKTREES", "LAST FETCHED")
	fmt.Println(strings.Repeat("-", 85))

	for _, proj := range projects {
		// Get worktree count
		worktrees, err := db.GetWorktreesByProject(database, proj.ID)
		if err != nil {
			return eris.Wrap(err, "failed to get worktrees for project")
		}

		lastFetched := "never"
		if proj.LastFetched != nil {
			lastFetched = formatTimeAgo(*proj.LastFetched)
		}

		fmt.Printf("%-50s %-12d %-20s\n",
			truncate(proj.Name, 50),
			len(worktrees),
			lastFetched,
		)
	}

	return nil
}

func listAllSessions(database *sql.DB) error {
	// Get all sessions with their worktree and project info
	query := `
		SELECT
			s.id, s.worktree_id, s.tmux_session_name, s.created_at, s.last_attached,
			w.id, w.project_id, w.branch, w.path, w.is_main, w.created_at, w.last_used,
			p.id, p.name, p.remote_url, p.local_path, p.created_at, p.last_fetched
		FROM sessions s
		JOIN worktrees w ON s.worktree_id = w.id
		JOIN projects p ON w.project_id = p.id
		ORDER BY s.last_attached DESC
	`

	rows, err := database.Query(query)
	if err != nil {
		return eris.Wrap(err, "failed to query sessions")
	}
	defer rows.Close()

	type SessionDetail struct {
		SessionID    int
		SessionName  string
		ProjectName  string
		Branch       string
		WorktreePath string
		LastAttached time.Time
		LastUsed     time.Time
		LastFetched  *time.Time
	}

	var sessions []SessionDetail

	for rows.Next() {
		var (
			sessID, wtID, projID                                      int
			sessName, wtBranch, wtPath, projName, projURL, projPath   string
			wtIsMain                                                  bool
			sessCreated, sessAttached, wtCreated, wtUsed, projCreated time.Time
			projFetched                                               sql.NullTime
		)

		err := rows.Scan(
			&sessID, &wtID, &sessName, &sessCreated, &sessAttached,
			&wtID, &projID, &wtBranch, &wtPath, &wtIsMain, &wtCreated, &wtUsed,
			&projID, &projName, &projURL, &projPath, &projCreated, &projFetched,
		)
		if err != nil {
			return eris.Wrap(err, "failed to scan session row")
		}

		var lastFetched *time.Time
		if projFetched.Valid {
			lastFetched = &projFetched.Time
		}

		sessions = append(sessions, SessionDetail{
			SessionID:    sessID,
			SessionName:  sessName,
			ProjectName:  projName,
			Branch:       wtBranch,
			WorktreePath: wtPath,
			LastAttached: sessAttached,
			LastUsed:     wtUsed,
			LastFetched:  lastFetched,
		})
	}

	if err := rows.Err(); err != nil {
		return eris.Wrap(err, "error iterating session rows")
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		fmt.Println("Clone a repository with: sesh clone <remote-url>")
		fmt.Println("Or switch to a branch with: sesh switch <branch>")
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
	fmt.Printf("%-30s %-20s %-30s %-15s\n", "PROJECT", "BRANCH", "SESSION NAME", "LAST USED")
	fmt.Println(strings.Repeat("-", 100))

	for _, sess := range sessions {
		fmt.Printf("%-30s %-20s %-30s %-15s\n",
			truncate(sess.ProjectName, 30),
			truncate(sess.Branch, 20),
			truncate(sess.SessionName, 30),
			formatTimeAgo(sess.LastAttached),
		)
	}

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
