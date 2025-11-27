package cmd

import (
	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var popCmd = &cobra.Command{
	Use:     "pop",
	Aliases: []string{"p", "back", "last"},
	Short:   "Switch to the previous session in history",
	Long: `Switch to the previous session you were working on.

The pop command uses session history to switch back to the last session
you switched to. This is useful for quickly switching between two sessions
you're actively working on.

Examples:
  sesh pop         # Switch to previous session
  sesh p           # Short alias
  sesh back        # Alternative alias
  sesh last        # Another alias`,
	RunE: runPop,
}

func init() {
	rootCmd.AddCommand(popCmd)
}

func runPop(cmd *cobra.Command, args []string) error {
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

	// Get current session name (if inside a session)
	currentSessionName := ""
	if sessionMgr.IsInsideSession() {
		currentSessionName, err = sessionMgr.GetCurrentSessionName()
		if err != nil {
			// Not critical if we can't get current session name
			currentSessionName = ""
		}
	}

	// Get database path
	dbPath, err := config.GetDBPath()
	if err != nil {
		return eris.Wrap(err, "failed to get database path")
	}

	// Initialize database
	database, err := db.InitDB(dbPath)
	if err != nil {
		return eris.Wrap(err, "failed to initialize database")
	}
	defer database.Close()

	// Get previous session from history
	previousSession, err := db.GetPreviousSession(database, currentSessionName)
	if err != nil {
		return eris.Wrap(err, "no previous session found in history")
	}

	// Check if the previous session still exists
	exists, err := sessionMgr.Exists(previousSession.SessionName)
	if err != nil {
		return eris.Wrap(err, "failed to check session existence")
	}

	if !exists {
		return eris.Errorf(
			"previous session '%s' no longer exists (from %s - %s)",
			previousSession.SessionName,
			previousSession.ProjectName,
			previousSession.Branch,
		)
	}

	// Display info about where we're switching to
	disp := display.NewStderr()
	disp.Printf(
		"%s Switching to previous session: %s (%s - %s)\n",
		disp.InfoText("→"),
		disp.Bold(previousSession.SessionName),
		previousSession.ProjectName,
		previousSession.Branch,
	)

	// Record this as a session access (so we can pop back)
	recordSessionHistory(previousSession.SessionName, previousSession.ProjectName, previousSession.Branch)

	// Attach to the previous session
	return sessionMgr.Attach(previousSession.SessionName)
}
