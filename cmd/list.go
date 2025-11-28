package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	listProjects       bool
	listSessions       bool
	listPRs            bool
	listJSON           bool
	listPlain          bool
	listCurrentProject bool
	listRunning        bool
	listAll            bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "status"},
	Short:   "List projects, worktrees, sessions, or pull requests",
	Long: `Display all projects, worktrees, sessions, or pull requests.

By default, shows all sessions with their project and branch information.

Examples:
  sesh list                        # List all sessions
  sesh list --projects             # List only projects
  sesh list --sessions             # List only sessions
  sesh list --pr                   # List open pull requests
  sesh list --json                 # Output in JSON format
  sesh list --plain                # Output session names only (for piping to fzf)
  sesh list --current-project      # List sessions for current project only
  sesh list --running              # List only running sessions
  sesh list --all                  # List all sessions (running and stopped)`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listProjects, "projects", false, "Show only projects")
	listCmd.Flags().BoolVar(&listSessions, "sessions", false, "Show only sessions (default)")
	listCmd.Flags().BoolVar(&listPRs, "pr", false, "Show open pull requests")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVar(&listPlain, "plain", false, "Output session names only (for piping)")
	listCmd.Flags().BoolVar(&listCurrentProject, "current-project", false, "Filter to sessions for current project")
	listCmd.Flags().BoolVar(&listRunning, "running", false, "Show only running sessions")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Show all sessions (running and stopped)")
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

	if listPRs {
		return listAllPRs(cmd.Context(), cfg)
	}

	// Default: list sessions
	return listAllSessions(cfg)
}

func listAllProjects(cfg *config.Config) error {
	disp := display.NewStderr()

	// Discover all projects from filesystem
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return eris.Wrap(err, "failed to discover projects")
	}

	if len(projects) == 0 {
		disp.Info("No projects found.")
		disp.Printf(
			"  %s Clone a repository with: %s\n",
			disp.Faint("→"),
			disp.Bold("sesh clone <remote-url>"),
		)
		return nil
	}

	if listJSON {
		data, err := json.MarshalIndent(projects, "", "  ")
		if err != nil {
			return eris.Wrap(err, "failed to marshal projects to JSON")
		}
		// JSON output is pipeable, so use stdout
		fmt.Println(string(data))
		return nil
	}

	// Print tree header
	disp.Printf("\n%s\n", disp.Bold("Projects"))
	disp.Println()

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

		disp.Printf(
			"%s %s %s\n",
			disp.Faint(prefix),
			disp.Bold(proj.Name),
			disp.Faint(
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
			disp.Printf("%s%s %s %s\n",
				disp.Faint(childPrefix),
				disp.Faint(wtPrefix),
				disp.InfoText(wt.Branch),
				disp.Faint(fmt.Sprintf("(last used %s)", lastUsed)),
			)
		}
	}
	disp.Println()

	return nil
}

func listAllSessions(cfg *config.Config) error {
	disp := display.NewStderr()

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

	// Detect current project if --current-project flag is set
	var currentProjectName string
	if listCurrentProject {
		cwd, err := os.Getwd()
		if err != nil {
			return eris.Wrap(err, "failed to get current working directory")
		}

		// Resolve project from current directory
		currentProj, err := project.ResolveProject(cfg.WorkspaceDir, "", cwd)
		if err != nil {
			return eris.Wrap(err, "failed to resolve current project - are you in a sesh workspace?")
		}
		currentProjectName = currentProj.Name
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
		// Filter by current project if requested
		if listCurrentProject && proj.Name != currentProjectName {
			continue
		}

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

			// Filter by running state if requested
			if listRunning && !isRunning {
				continue
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
		// For plain output, just return empty (no sessions to list)
		if listPlain {
			return nil
		}
		disp.Info("No worktrees found.")
		disp.Printf(
			"  %s Clone a repository with: %s\n",
			disp.Faint("→"),
			disp.Bold("sesh clone <remote-url>"),
		)
		disp.Printf(
			"  %s Or switch to a branch with: %s\n",
			disp.Faint("→"),
			disp.Bold("sesh switch <branch>"),
		)
		return nil
	}

	if listJSON {
		data, err := json.MarshalIndent(sessions, "", "  ")
		if err != nil {
			return eris.Wrap(err, "failed to marshal sessions to JSON")
		}
		// JSON output is pipeable, so use stdout
		fmt.Println(string(data))
		return nil
	}

	if listPlain {
		// Plain output: just session names, one per line (to stdout for piping)
		for _, sess := range sessions {
			fmt.Println(sess.SessionName)
		}
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
	disp.Printf("\n%s\n", disp.Bold("Sessions"))
	disp.Println()

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

		disp.Printf("%s %s\n",
			disp.Faint(prefix),
			disp.Bold(projName),
		)

		// Print sessions/branches as children
		for j, sess := range projSessions {
			isLastSession := j == len(projSessions)-1
			sessPrefix := "├──"
			if isLastSession {
				sessPrefix = "└──"
			}

			statusIcon := disp.Faint("○")
			statusText := disp.Faint("stopped")
			if sess.IsRunning {
				statusIcon = disp.SuccessText("●")
				statusText = disp.SuccessText("running")
			}

			disp.Printf("%s%s %s %s %s\n",
				disp.Faint(childPrefix),
				disp.Faint(sessPrefix),
				disp.InfoText(sess.Branch),
				statusIcon,
				statusText,
			)
		}
	}
	disp.Println()

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

// listAllPRs lists all open pull requests for the current project
func listAllPRs(ctx context.Context, cfg *config.Config) error {
	disp := display.NewStderr()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Resolve project from current directory
	proj, err := project.ResolveProject(cfg.WorkspaceDir, "", cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project from current directory")
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

	// Check if gh CLI is installed and authenticated (for GitHub)
	if provider.Name() == "github" {
		if err := pr.CheckGHCLI(); err != nil {
			return err
		}
	}

	// List open PRs
	prs, err := provider.ListOpenPRs(ctx, proj.LocalPath)
	if err != nil {
		return eris.Wrap(err, "failed to list pull requests")
	}

	if len(prs) == 0 {
		disp.Info("No open pull requests found.")
		return nil
	}

	if listJSON {
		data, err := json.MarshalIndent(prs, "", "  ")
		if err != nil {
			return eris.Wrap(err, "failed to marshal PRs to JSON")
		}
		// JSON output is pipeable, so use stdout
		fmt.Println(string(data))
		return nil
	}

	// Print tree header
	disp.Printf("\n%s\n", disp.Bold(fmt.Sprintf("Open Pull Requests (%s)", proj.Name)))
	disp.Println()

	for i, pullRequest := range prs {
		isLast := i == len(prs)-1
		prefix := "├──"
		if isLast {
			prefix = "└──"
		}

		// Format PR info
		prNum := disp.InfoText(fmt.Sprintf("#%d", pullRequest.Number))
		title := disp.Bold(pullRequest.Title)
		branch := disp.Faint(fmt.Sprintf("(%s → %s)", pullRequest.Branch, pullRequest.BaseBranch))
		author := disp.Faint(fmt.Sprintf("@%s", pullRequest.Author))
		updated := disp.Faint(fmt.Sprintf("updated %s", formatTimeAgo(pullRequest.UpdatedAt)))

		disp.Printf("%s %s %s %s %s %s\n",
			disp.Faint(prefix),
			prNum,
			title,
			branch,
			author,
			updated,
		)

		// Print labels if any
		if len(pullRequest.Labels) > 0 {
			childPrefix := "    "
			if !isLast {
				childPrefix = "│   "
			}
			labelStr := strings.Join(pullRequest.Labels, ", ")
			disp.Printf("%s%s %s\n",
				disp.Faint(childPrefix),
				disp.Faint("└──"),
				disp.Faint(fmt.Sprintf("Labels: %s", labelStr)),
			)
		}
	}
	disp.Println()

	return nil
}
