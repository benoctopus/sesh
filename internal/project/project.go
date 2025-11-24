package project

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/rotisserie/eris"
)

// ResolveProject resolves a project from a project name or current working directory
// If projectName is empty, it will attempt to detect the project from CWD
// Priority:
// 1. If projectName is provided, look it up in the database
// 2. If projectName is empty, detect project from CWD
// 3. Return error if not found
func ResolveProject(database *sql.DB, projectName string, cwd string) (*models.Project, error) {
	// If project name is explicitly provided, look it up
	if projectName != "" {
		project, err := db.GetProject(database, projectName)
		if err != nil {
			return nil, eris.Wrapf(err, "project '%s' not found", projectName)
		}
		return project, nil
	}

	// Try to detect project from CWD
	detectedName, err := DetectProjectFromCWD(cwd)
	if err != nil {
		return nil, eris.Wrap(err, "could not detect project from current directory")
	}

	// Look up detected project in database
	project, err := db.GetProject(database, detectedName)
	if err != nil {
		return nil, eris.Wrapf(err, "detected project '%s' not found in database", detectedName)
	}

	return project, nil
}

// DetectProjectFromCWD detects the project name from the current working directory
// It finds the git repository root and extracts the project name from the remote URL
func DetectProjectFromCWD(cwd string) (string, error) {
	// Find git root from current directory
	gitRoot, err := FindGitRoot(cwd)
	if err != nil {
		return "", err
	}

	// Get remote URL from git repository
	remoteURL, err := git.GetRemoteURL(gitRoot)
	if err != nil {
		return "", eris.Wrap(err, "failed to get remote URL from git repository")
	}

	// Extract project name from remote URL
	projectName, err := git.GenerateProjectName(remoteURL)
	if err != nil {
		return "", eris.Wrap(err, "failed to extract project name from remote URL")
	}

	return projectName, nil
}

// FindGitRoot finds the git repository root starting from the given path
// It traverses up the directory tree until it finds a .git directory
func FindGitRoot(startPath string) (string, error) {
	path, err := filepath.Abs(startPath)
	if err != nil {
		return "", eris.Wrap(err, "failed to get absolute path")
	}

	for {
		gitPath := filepath.Join(path, ".git")

		// Check if .git exists (either as directory or file for worktrees)
		if _, err := os.Stat(gitPath); err == nil {
			return path, nil
		}

		// Move to parent directory
		parent := filepath.Dir(path)

		// If we've reached the root, stop
		if parent == path {
			return "", eris.New("not in a git repository")
		}

		path = parent
	}
}

// ExtractProjectFromRemote extracts the project name from a remote URL
// This is a convenience wrapper around git.GenerateProjectName
func ExtractProjectFromRemote(remoteURL string) (string, error) {
	return git.GenerateProjectName(remoteURL)
}

// IsInGitRepository checks if the given path is inside a git repository
func IsInGitRepository(path string) bool {
	_, err := FindGitRoot(path)
	return err == nil
}

// GetProjectRemoteURL gets the remote URL for a project by finding its git root
func GetProjectRemoteURL(projectPath string) (string, error) {
	gitRoot, err := FindGitRoot(projectPath)
	if err != nil {
		return "", err
	}
	return git.GetRemoteURL(gitRoot)
}

// ResolveProjectOrPrompt resolves a project with helpful error messages
// If no project is found, it returns a user-friendly error with suggestions
func ResolveProjectOrPrompt(database *sql.DB, projectName string, cwd string) (*models.Project, error) {
	project, err := ResolveProject(database, projectName, cwd)
	if err != nil {
		// Provide helpful error message
		if projectName == "" {
			// No project specified and CWD detection failed
			if !IsInGitRepository(cwd) {
				return nil, eris.New("not in a git repository and no project specified\n" +
					"Please either:\n" +
					"  1. Run this command from within a git repository\n" +
					"  2. Specify a project name explicitly\n" +
					"  3. Use 'sesh clone <url>' to clone a new project")
			}
			return nil, eris.Wrap(err, "project not found in sesh workspace\n"+
				"Use 'sesh clone <url>' to add this project to sesh")
		}

		// Project was specified but not found
		return nil, eris.Wrapf(err, "project '%s' not found\n"+
			"Use 'sesh list --projects' to see all available projects", projectName)
	}
	return project, nil
}

// DetectWorktreeFromCWD detects which worktree the current directory is in
// Returns the worktree path if found, otherwise returns empty string
func DetectWorktreeFromCWD(database *sql.DB, cwd string) (*models.Worktree, error) {
	// First, ensure we're in a git repository
	gitRoot, err := FindGitRoot(cwd)
	if err != nil {
		return nil, eris.Wrap(err, "not in a git repository")
	}

	// Get the worktree from database by path
	worktree, err := db.GetWorktreeByPath(database, gitRoot)
	if err != nil {
		return nil, eris.Wrap(err, "current directory is not a sesh-managed worktree")
	}

	return worktree, nil
}

// NormalizeProjectName normalizes a project name for consistent lookup
// Removes any trailing slashes and ensures consistent path separators
func NormalizeProjectName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "/")
	name = filepath.Clean(name)
	return name
}
