package cmd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/pr"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/spf13/cobra"
)

// completionTimeout is the maximum time to wait for PR completions
// Shell completions should be fast, so we use a short timeout
const completionTimeout = 5 * time.Second

// ghCLICheck caches the result of the gh CLI check so it only runs once per process
var (
	ghCLICheckOnce   sync.Once
	ghCLICheckResult error
)

// checkGHCLICached checks if gh CLI is installed and authenticated, caching the result.
// This ensures the check only runs once per process, even if called multiple times.
func checkGHCLICached() error {
	ghCLICheckOnce.Do(func() {
		ghCLICheckResult = pr.CheckGHCLI()
	})
	return ghCLICheckResult
}

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
// If --pr flag is set, returns PR completions instead of branches
func completeBranches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Check if --pr flag is set - provide PR completions instead
	if prFlag, _ := cmd.Flags().GetBool("pr"); prFlag {
		return completePRs(cmd, args, toComplete)
	}

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

// completePRs returns a completion function that provides PR numbers with titles
// This fetches open PRs from GitHub and formats them for shell completion
func completePRs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Check if --project flag was provided
	projectName, _ := cmd.Flags().GetString("project")

	// Resolve project
	proj, err := project.ResolveProject(cfg.WorkspaceDir, projectName, cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get remote URL
	remoteURL, err := git.GetRemoteURL(proj.LocalPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Create PR provider
	provider, err := pr.NewProvider(remoteURL)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Check if gh CLI is installed and authenticated (for GitHub)
	if provider.Name() == "github" {
		if err := checkGHCLICached(); err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// List open PRs (with a short timeout for completions)
	ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
	defer cancel()

	prs, err := provider.ListOpenPRs(ctx, proj.LocalPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Format PRs for completion: "number\ttitle"
	// Cobra uses tab separator for completion descriptions
	var completions []string
	for _, pullRequest := range prs {
		completion := fmt.Sprintf("%d\t#%d: %s (@%s)",
			pullRequest.Number,
			pullRequest.Number,
			pullRequest.Title,
			pullRequest.Author,
		)
		completions = append(completions, completion)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
