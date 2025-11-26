package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/fuzzy"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/tty"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	cleanOrphaned      bool
	cleanRemoteDeleted bool
	cleanForce         bool
	cleanProjectName   string
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up stale worktrees and sessions",
	Long: `Clean up stale worktrees and sessions interactively.

By default, presents a multi-select interface to choose which worktrees/sessions to delete.

Options:
  --orphaned         Delete worktrees that don't have active sessions
  --remote-deleted   Delete local worktrees for branches that have been deleted on the remote
  --force            Skip confirmation prompts

The project is automatically detected from the current working directory,
or can be specified explicitly with the --project flag.

Examples:
  sesh clean                           # Interactive multi-select to delete worktrees
  sesh clean --orphaned                # Delete worktrees without active sessions
  sesh clean --remote-deleted          # Delete local worktrees for remote-deleted branches
  sesh clean --orphaned --force        # Delete orphaned worktrees without confirmation
  sesh clean --project myproject       # Clean specific project`,
	RunE: runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanOrphaned, "orphaned", false, "Delete worktrees without active sessions")
	cleanCmd.Flags().
		BoolVar(&cleanRemoteDeleted, "remote-deleted", false, "Delete local worktrees for remote-deleted branches")
	cleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip confirmation prompts")
	cleanCmd.Flags().StringVarP(&cleanProjectName, "project", "p", "", "Specify project explicitly")
}

func runClean(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()

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
	proj, err := project.ResolveProject(cfg.WorkspaceDir, cleanProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	// Initialize session manager
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return eris.Wrap(err, "failed to initialize session manager")
	}

	// Handle different clean modes
	if cleanOrphaned {
		return cleanOrphanedWorktrees(cfg, proj, sessionMgr, disp)
	}

	if cleanRemoteDeleted {
		return cleanRemoteDeletedBranches(cfg, proj, sessionMgr, disp)
	}

	// Default: interactive multi-select
	return cleanInteractive(cfg, proj, sessionMgr, disp)
}

// cleanInteractive presents a multi-select interface to choose worktrees to delete
func cleanInteractive(
	cfg *config.Config,
	proj *models.Project,
	sessionMgr session.SessionManager,
	disp display.Printer,
) error {
	// Get all worktrees for this project
	worktrees, err := state.DiscoverWorktrees(proj)
	if err != nil {
		return eris.Wrap(err, "failed to discover worktrees")
	}

	if len(worktrees) == 0 {
		disp.Println("No worktrees found for this project.")
		return nil
	}

	// Build list of worktrees to present (exclude main worktree)
	var items []string
	var selectableWorktrees []*models.Worktree

	for _, wt := range worktrees {
		if wt.IsMain {
			continue // Skip main worktree
		}

		// Check if session exists
		sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)
		hasSession, _ := sessionMgr.Exists(sessionName)

		var label string
		if hasSession {
			label = fmt.Sprintf("%s (active session)", wt.Branch)
		} else {
			label = fmt.Sprintf("%s (no session)", wt.Branch)
		}

		items = append(items, label)
		selectableWorktrees = append(selectableWorktrees, wt)
	}

	if len(items) == 0 {
		disp.Println("No worktrees available to clean (main worktree cannot be deleted).")
		return nil
	}

	// In noninteractive mode, fuzzy finder won't work - require specific flags
	if !tty.IsInteractive() {
		return eris.New("interactive mode required for default clean (use --orphaned or --remote-deleted in noninteractive mode)")
	}

	// Present multi-select interface
	disp.Println("Select worktrees to delete (TAB to select, ENTER to confirm):")
	selected, err := fuzzy.MultiSelect(items, "Select worktrees> ")
	if err != nil {
		// User cancelled or error occurred
		if strings.Contains(err.Error(), "cancelled") {
			disp.Println("Cleanup cancelled.")
			return nil
		}
		return eris.Wrap(err, "failed to select worktrees")
	}

	if len(selected) == 0 {
		disp.Println("No worktrees selected.")
		return nil
	}

	// Map selected labels back to worktrees
	var toDelete []*models.Worktree
	for _, label := range selected {
		// Extract branch name from label (remove status suffix)
		branch := strings.TrimSuffix(label, " (active session)")
		branch = strings.TrimSuffix(branch, " (no session)")

		// Find the corresponding worktree
		for _, wt := range selectableWorktrees {
			if wt.Branch == branch {
				toDelete = append(toDelete, wt)
				break
			}
		}
	}

	// Confirm deletion
	if !cleanForce {
		disp.Printf("\nThis will delete %d worktree(s) and their associated sessions:\n", len(toDelete))
		for _, wt := range toDelete {
			disp.Printf("  - %s (%s)\n", wt.Branch, wt.Path)
		}
		disp.Print("\nAre you sure? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			disp.Println("Cleanup cancelled.")
			return nil
		}
	}

	// Delete selected worktrees
	for _, wt := range toDelete {
		if err := deleteWorktreeAndSession(cfg, proj, wt, sessionMgr, disp); err != nil {
			disp.Printf("Warning: failed to delete worktree %s: %v\n", wt.Branch, err)
		}
	}

	disp.Printf("\nSuccessfully deleted %d worktree(s).\n", len(toDelete))
	return nil
}

// cleanOrphanedWorktrees deletes worktrees that don't have active sessions
func cleanOrphanedWorktrees(
	cfg *config.Config,
	proj *models.Project,
	sessionMgr session.SessionManager,
	disp display.Printer,
) error {
	// Get all worktrees for this project
	worktrees, err := state.DiscoverWorktrees(proj)
	if err != nil {
		return eris.Wrap(err, "failed to discover worktrees")
	}

	// Find orphaned worktrees (no active session)
	var orphaned []*models.Worktree
	for _, wt := range worktrees {
		if wt.IsMain {
			continue // Skip main worktree
		}

		sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)
		hasSession, err := sessionMgr.Exists(sessionName)
		if err != nil {
			disp.Printf("Warning: failed to check session for %s: %v\n", wt.Branch, err)
			continue
		}

		if !hasSession {
			orphaned = append(orphaned, wt)
		}
	}

	if len(orphaned) == 0 {
		disp.Println("No orphaned worktrees found.")
		return nil
	}

	// Show orphaned worktrees
	disp.Printf("Found %d orphaned worktree(s) without active sessions:\n", len(orphaned))
	for _, wt := range orphaned {
		disp.Printf("  - %s (%s)\n", wt.Branch, wt.Path)
	}

	// In noninteractive mode, require --force flag
	if !cleanForce {
		if !tty.IsInteractive() {
			return eris.New("--force flag required for deletion in noninteractive mode")
		}

		// Ask for confirmation in interactive mode
		disp.Print("\nDelete these worktrees? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			disp.Println("Cleanup cancelled.")
			return nil
		}
	}

	// Delete orphaned worktrees
	for _, wt := range orphaned {
		if err := deleteWorktreeAndSession(cfg, proj, wt, sessionMgr, disp); err != nil {
			disp.Printf("Warning: failed to delete worktree %s: %v\n", wt.Branch, err)
		}
	}

	disp.Printf("\nSuccessfully deleted %d orphaned worktree(s).\n", len(orphaned))
	return nil
}

// cleanRemoteDeletedBranches deletes local worktrees for branches that have been deleted on the remote
func cleanRemoteDeletedBranches(
	cfg *config.Config,
	proj *models.Project,
	sessionMgr session.SessionManager,
	disp display.Printer,
) error {
	// Get all local worktrees
	worktrees, err := state.DiscoverWorktrees(proj)
	if err != nil {
		return eris.Wrap(err, "failed to discover worktrees")
	}

	// Get all remote branches
	remoteBranches, err := git.ListRemoteBranches(proj.LocalPath)
	if err != nil {
		return eris.Wrap(err, "failed to list remote branches")
	}

	// Build a set of remote branches for fast lookup
	remoteBranchSet := make(map[string]bool)
	for _, branch := range remoteBranches {
		remoteBranchSet[branch] = true
	}

	// Find worktrees for branches that no longer exist on remote
	var deleted []*models.Worktree
	for _, wt := range worktrees {
		if wt.IsMain {
			continue // Skip main worktree
		}

		// Check if this branch exists on remote
		if !remoteBranchSet[wt.Branch] {
			deleted = append(deleted, wt)
		}
	}

	if len(deleted) == 0 {
		disp.Println("No worktrees found for remote-deleted branches.")
		return nil
	}

	// Show deleted branches
	disp.Printf("Found %d worktree(s) for branches deleted on remote:\n", len(deleted))
	for _, wt := range deleted {
		disp.Printf("  - %s (%s)\n", wt.Branch, wt.Path)
	}

	// In noninteractive mode, require --force flag
	if !cleanForce {
		if !tty.IsInteractive() {
			return eris.New("--force flag required for deletion in noninteractive mode")
		}

		// Ask for confirmation in interactive mode
		disp.Print("\nDelete these worktrees? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return eris.Wrap(err, "failed to read confirmation")
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			disp.Println("Cleanup cancelled.")
			return nil
		}
	}

	// Delete worktrees for remote-deleted branches
	for _, wt := range deleted {
		if err := deleteWorktreeAndSession(cfg, proj, wt, sessionMgr, disp); err != nil {
			disp.Printf("Warning: failed to delete worktree %s: %v\n", wt.Branch, err)
		}
	}

	disp.Printf("\nSuccessfully deleted %d worktree(s) for remote-deleted branches.\n", len(deleted))
	return nil
}

// deleteWorktreeAndSession deletes a worktree and its associated session
func deleteWorktreeAndSession(
	cfg *config.Config,
	proj *models.Project,
	wt *models.Worktree,
	sessionMgr session.SessionManager,
	disp display.Printer,
) error {
	// Generate session name
	sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)

	// Kill session if it exists
	exists, err := sessionMgr.Exists(sessionName)
	if err != nil {
		disp.Printf("Warning: failed to check session existence for %s: %v\n", wt.Branch, err)
	} else if exists {
		disp.Printf("Killing %s session: %s\n", sessionMgr.Name(), sessionName)
		if err := sessionMgr.Delete(sessionName); err != nil {
			disp.Printf("Warning: failed to kill session: %v\n", err)
		}
	}

	// Remove worktree
	disp.Printf("Removing worktree: %s\n", wt.Path)
	if err := git.RemoveWorktree(proj.LocalPath, wt.Path); err != nil {
		return eris.Wrap(err, "failed to remove worktree")
	}

	return nil
}
