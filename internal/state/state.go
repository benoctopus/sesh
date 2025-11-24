package state

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benoctopus/sesh/internal/git"
	"github.com/benoctopus/sesh/internal/models"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/workspace"
	"github.com/rotisserie/eris"
)

// DiscoverProjects scans the workspace directory and discovers all projects
// A project is identified by a .git directory (bare repo) in the workspace structure
func DiscoverProjects(workspaceDir string) ([]*models.Project, error) {
	var projects []*models.Project

	// Walk the workspace directory looking for .git directories
	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory named .git
		if !info.IsDir() || info.Name() != ".git" {
			return nil
		}

		// Check if it's a bare repository
		configPath := filepath.Join(path, "config")
		if _, err := os.Stat(configPath); err != nil {
			return nil // Not a valid git repo
		}

		// Get the project path (parent of .git)
		projectPath := filepath.Dir(path)

		// Extract project name from path relative to workspace
		relPath, err := filepath.Rel(workspaceDir, projectPath)
		if err != nil {
			return nil
		}

		// Get remote URL
		remoteURL, err := git.GetRemoteURL(path)
		if err != nil {
			// If we can't get remote URL, skip this project
			return nil
		}

		// Get creation time from .git directory
		gitInfo, _ := os.Stat(path)
		createdAt := gitInfo.ModTime()

		project := &models.Project{
			Name:      relPath,
			RemoteURL: remoteURL,
			LocalPath: path,
			CreatedAt: createdAt,
		}

		projects = append(projects, project)

		// Don't recurse into .git directories
		return filepath.SkipDir
	})

	if err != nil {
		return nil, eris.Wrap(err, "failed to discover projects")
	}

	return projects, nil
}

// DiscoverWorktrees discovers all worktrees for a given project
func DiscoverWorktrees(project *models.Project) ([]*models.Worktree, error) {
	// Use git worktree list to get all worktrees
	worktrees, err := git.ListWorktrees(project.LocalPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to list worktrees")
	}

	var result []*models.Worktree
	for _, wt := range worktrees {
		// Branch is already provided by ListWorktrees
		branch := wt.Branch

		// Check if it's the main worktree (first one, or matches default branch)
		isMain := len(result) == 0

		// Get last modified time
		info, err := os.Stat(wt.Path)
		lastUsed := time.Now()
		if err == nil {
			lastUsed = info.ModTime()
		}

		worktree := &models.Worktree{
			Branch:    branch,
			Path:      wt.Path,
			IsMain:    isMain,
			CreatedAt: lastUsed, // Best approximation
			LastUsed:  lastUsed,
		}

		result = append(result, worktree)
	}

	return result, nil
}

// DiscoverSessions discovers all active sessions using the session manager
func DiscoverSessions(sessionMgr session.SessionManager) ([]string, error) {
	return sessionMgr.List()
}

// GetProject finds a project by name from the workspace
func GetProject(workspaceDir, projectName string) (*models.Project, error) {
	projects, err := DiscoverProjects(workspaceDir)
	if err != nil {
		return nil, err
	}

	for _, proj := range projects {
		if proj.Name == projectName {
			return proj, nil
		}
	}

	return nil, eris.Errorf("project not found: %s", projectName)
}

// GetProjectByShortName finds a project by its short name (repo name)
func GetProjectByShortName(workspaceDir, shortName string) (*models.Project, error) {
	projects, err := DiscoverProjects(workspaceDir)
	if err != nil {
		return nil, err
	}

	var matches []*models.Project
	for _, proj := range projects {
		repoName := filepath.Base(proj.Name)
		if repoName == shortName {
			matches = append(matches, proj)
		}
	}

	if len(matches) == 0 {
		return nil, eris.Errorf("no project found matching '%s'", shortName)
	}

	if len(matches) == 1 {
		return matches[0], nil
	}

	// Multiple matches - return error with list
	var matchNames []string
	for _, match := range matches {
		matchNames = append(matchNames, match.Name)
	}
	return nil, eris.Errorf(
		"multiple projects found with name '%s': %s\nPlease specify the full project name",
		shortName,
		strings.Join(matchNames, ", "),
	)
}

// GetWorktree finds a worktree by project and branch
func GetWorktree(project *models.Project, branch string) (*models.Worktree, error) {
	worktrees, err := DiscoverWorktrees(project)
	if err != nil {
		return nil, err
	}

	for _, wt := range worktrees {
		if wt.Branch == branch {
			return wt, nil
		}
	}

	return nil, eris.Errorf("worktree not found for branch: %s", branch)
}

// GetWorktreeByPath finds a worktree by its filesystem path
func GetWorktreeByPath(workspaceDir, path string) (*models.Worktree, error) {
	// Find the project that contains this path
	projects, err := DiscoverProjects(workspaceDir)
	if err != nil {
		return nil, err
	}

	for _, proj := range projects {
		worktrees, err := DiscoverWorktrees(proj)
		if err != nil {
			continue
		}

		for _, wt := range worktrees {
			if wt.Path == path {
				return wt, nil
			}
		}
	}

	return nil, eris.Errorf("worktree not found at path: %s", path)
}

// SessionExists checks if a session exists with the given name
func SessionExists(sessionMgr session.SessionManager, sessionName string) (bool, error) {
	return sessionMgr.Exists(sessionName)
}

// MatchSessionToWorktree attempts to match a session name to a worktree
// Returns the worktree if found, nil otherwise
func MatchSessionToWorktree(workspaceDir, sessionName string) (*models.Worktree, error) {
	// Parse session name to get repo and branch
	repoName, branch, err := workspace.ParseSessionName(sessionName)
	if err != nil {
		return nil, err
	}

	// Find project by short name
	projects, err := DiscoverProjects(workspaceDir)
	if err != nil {
		return nil, err
	}

	for _, proj := range projects {
		if filepath.Base(proj.Name) == repoName {
			// Found matching project, look for worktree
			return GetWorktree(proj, branch)
		}
	}

	return nil, eris.Errorf("no worktree found for session: %s", sessionName)
}

// CreateProjectRecord creates a project record (for compatibility)
// In the stateless model, this just ensures the directory structure exists
func CreateProjectRecord(workspaceDir string, project *models.Project) error {
	projectPath := filepath.Join(workspaceDir, project.Name)
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		return eris.Wrap(err, "failed to create project directory")
	}
	return nil
}

// CreateWorktreeRecord creates a worktree record (for compatibility)
// In the stateless model, this is a no-op since worktrees are discovered
func CreateWorktreeRecord(worktree *models.Worktree) error {
	// No-op: worktree already exists in filesystem via git worktree add
	return nil
}

// CreateSessionRecord creates a session record (for compatibility)
// In the stateless model, this is a no-op since sessions are discovered
func CreateSessionRecord(session *models.Session) error {
	// No-op: session already exists via session manager
	return nil
}
