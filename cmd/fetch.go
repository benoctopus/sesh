package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/project"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var (
	fetchAll         bool
	fetchProjectName string
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [project]",
	Short: "Fetch latest changes from remote for a project",
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

	if fetchAll {
		return fetchAllProjects(database)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return eris.Wrap(err, "failed to get current working directory")
	}

	// Resolve project
	proj, err := project.ResolveProject(database, fetchProjectName, cwd)
	if err != nil {
		return eris.Wrap(err, "failed to resolve project")
	}

	return fetchProject(database, proj)
}

func fetchProject(database *sql.DB, proj *models.Project) error {
	fmt.Printf("Fetching %s...\n", proj.Name)

	// Run git fetch
	if err := git.Fetch(proj.LocalPath); err != nil {
		return eris.Wrap(err, "failed to fetch repository")
	}

	// Update last_fetched timestamp
	if err := db.UpdateProjectFetchTime(database, proj.ID); err != nil {
		return eris.Wrap(err, "failed to update fetch time")
	}

	fmt.Printf("Successfully fetched %s\n", proj.Name)
	return nil
}

func fetchAllProjects(database *sql.DB) error {
	projects, err := db.GetAllProjects(database)
	if err != nil {
		return eris.Wrap(err, "failed to get projects")
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

		if err := db.UpdateProjectFetchTime(database, proj.ID); err != nil {
			fmt.Printf(" warning: failed to update timestamp: %v\n", err)
		} else {
			fmt.Printf(" done\n")
		}

		successCount++
	}

	fmt.Printf("\nFetched %d/%d project(s) successfully", successCount, len(projects))
	if failCount > 0 {
		fmt.Printf(" (%d failed)", failCount)
	}
	fmt.Println()

	return nil
}
