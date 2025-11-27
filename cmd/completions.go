package cmd

import (
	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/spf13/cobra"
)

// completeProjects returns a completion function that provides project names
func completeProjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, proj := range projects {
		names = append(names, proj.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeSessions returns a completion function that provides session names
func completeSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Discover all projects and their worktrees to generate session names
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var sessionNames []string
	for _, proj := range projects {
		worktrees, err := state.DiscoverWorktrees(proj)
		if err != nil {
			continue
		}

		for _, wt := range worktrees {
			sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)
			sessionNames = append(sessionNames, sessionName)
		}
	}

	return sessionNames, cobra.ShellCompDirectiveNoFileComp
}

// completeActiveSessions returns a completion function that provides only active (running) session names
func completeActiveSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	sessions, err := sessionMgr.List()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return sessions, cobra.ShellCompDirectiveNoFileComp
}

// completeBranches returns a completion function that provides branch names for the current project
// It uses the --project flag if provided, otherwise tries to detect from cwd
func completeBranches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Check if --project flag was provided
	projectName, _ := cmd.Flags().GetString("project")

	// Discover all projects
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// If project flag is set, find that specific project
	if projectName != "" {
		for _, proj := range projects {
			if proj.Name == projectName {
				return getBranchesForProject(proj)
			}
		}
		// Project not found
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// No project flag - return branches from all projects (deduped)
	branchSet := make(map[string]struct{})
	for _, proj := range projects {
		worktrees, err := state.DiscoverWorktrees(proj)
		if err != nil {
			continue
		}
		for _, wt := range worktrees {
			branchSet[wt.Branch] = struct{}{}
		}
	}

	var branches []string
	for branch := range branchSet {
		branches = append(branches, branch)
	}

	return branches, cobra.ShellCompDirectiveNoFileComp
}

// getBranchesForProject returns all branches (worktrees) for a specific project
func getBranchesForProject(proj *models.Project) ([]string, cobra.ShellCompDirective) {
	worktrees, err := state.DiscoverWorktrees(proj)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var branches []string
	for _, wt := range worktrees {
		branches = append(branches, wt.Branch)
	}

	return branches, cobra.ShellCompDirectiveNoFileComp
}
