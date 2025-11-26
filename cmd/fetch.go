package cmd

import (
	"fmt"
	"os"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	fetchAll         bool
	fetchProjectName string
)

var fetchCmd = &cobra.Command{
	Use:     "fetch [project]",
	Aliases: []string{"fc"},
	Short:   "Fetch latest changes from remote for a project",
	Long: `Fetch the latest changes from the remote repository for a project.

By default, fetches changes for the current project (detected from working directory).
Use --all to fetch changes for all projects in the workspace.

Examples:
  sesh fetch                       # Fetch current project
  sesh fetch --project myproject   # Fetch specific project
  sesh fetch --all                 # Fetch all projects`,
	RunE: runFetch,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().BoolVar(&fetchAll, "all", false, "Fetch all projects")
	fetchCmd.Flags().StringVarP(&fetchProjectName, "project", "p", "", "Specify project explicitly")
}

func runFetch(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	if fetchAll {
		return fetchAllProjects(cfg)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Resolve project from filesystem state
	proj, err := project.ResolveProject(cfg.WorkspaceDir, fetchProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	return fetchProject(proj)
}

func fetchProject(proj *models.Project) error {
	fmt.Printf("Fetching %s...\n", proj.Name)

	// Run git fetch
	if err := git.Fetch(proj.LocalPath); err != nil {
		return eris.Wrap(err, "failed to fetch repository")
	}

	fmt.Printf("Successfully fetched %s\n", proj.Name)
	return nil
}

func fetchAllProjects(cfg *config.Config) error {
	// Discover all projects from filesystem
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return eris.Wrap(err, "failed to discover projects")
	}

	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	fmt.Printf("Fetching %d project(s)...\n\n", len(projects))

	successCount := 0
	failCount := 0

	for _, proj := range projects {
		fmt.Printf("Fetching %s...", proj.Name)

		if err := git.Fetch(proj.LocalPath); err != nil {
			fmt.Printf(" failed: %v\n", err)
			failCount++
			continue
		}

		fmt.Printf(" done\n")
		successCount++
	}

	fmt.Printf("\nFetched %d/%d project(s) successfully", successCount, len(projects))
	if failCount > 0 {
		fmt.Printf(" (%d failed)", failCount)
	}
	fmt.Println()

	return nil
}
