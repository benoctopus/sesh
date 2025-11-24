package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rotisserie/eris"
)

// GetProjectPath returns the full path to a project within the workspace directory
// Format: <workspaceDir>/<projectName>
// Example: ~/.sesh/github.com/user/repo
func GetProjectPath(workspaceDir, projectName string) string {
	return filepath.Join(workspaceDir, projectName)
}

// GetBareRepoPath returns the path to the bare git repository for a project
// Format: <workspaceDir>/<projectName>/.git
func GetBareRepoPath(workspaceDir, projectName string) string {
	return filepath.Join(workspaceDir, projectName, ".git")
}

// GetWorktreePath returns the full path to a worktree within a project
// Format: <projectPath>/<sanitizedBranch>
// Example: ~/.sesh/github.com/user/repo/main
func GetWorktreePath(projectPath, branch string) string {
	sanitizedBranch := SanitizeBranchName(branch)
	return filepath.Join(projectPath, sanitizedBranch)
}

// EnsureProjectDir creates the project directory if it doesn't exist
func EnsureProjectDir(projectPath string) error {
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		return eris.Wrapf(err, "failed to create project directory: %s", projectPath)
	}
	return nil
}

// GenerateSessionName generates a unique session name from project name and branch
// Format: <repoName>-<branch>
// Example: "repo-main", "myproject-feature-foo"
// Note: We use "-" instead of ":" because ":" is a special character in tmux
// that separates session names from window names (e.g., "session:window")
func GenerateSessionName(projectName, branch string) string {
	// Extract repository name from project path (last component)
	repoName := filepath.Base(projectName)

	// Sanitize branch name for session compatibility
	sanitizedBranch := SanitizeBranchName(branch)

	return fmt.Sprintf("%s-%s", repoName, sanitizedBranch)
}

// SanitizeBranchName sanitizes a branch name for use in filesystem paths and session names
// Replaces special characters with safe alternatives
// Examples:
//   - "feature/foo" -> "feature-foo"
//   - "release/v1.0.0" -> "release-v1.0.0"
//   - "fix:bug#123" -> "fix-bug-123"
func SanitizeBranchName(branch string) string {
	// Replace common separators with hyphens
	sanitized := branch
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "*", "-")
	sanitized = strings.ReplaceAll(sanitized, "?", "-")
	sanitized = strings.ReplaceAll(sanitized, "\"", "-")
	sanitized = strings.ReplaceAll(sanitized, "<", "-")
	sanitized = strings.ReplaceAll(sanitized, ">", "-")
	sanitized = strings.ReplaceAll(sanitized, "|", "-")
	sanitized = strings.ReplaceAll(sanitized, "#", "-")
	sanitized = strings.ReplaceAll(sanitized, "%", "-")
	sanitized = strings.ReplaceAll(sanitized, "&", "-")
	sanitized = strings.ReplaceAll(sanitized, "{", "-")
	sanitized = strings.ReplaceAll(sanitized, "}", "-")
	sanitized = strings.ReplaceAll(sanitized, "$", "-")
	sanitized = strings.ReplaceAll(sanitized, "!", "-")
	sanitized = strings.ReplaceAll(sanitized, "'", "-")
	sanitized = strings.ReplaceAll(sanitized, "`", "-")
	sanitized = strings.ReplaceAll(sanitized, "=", "-")
	sanitized = strings.ReplaceAll(sanitized, "@", "-")
	sanitized = strings.ReplaceAll(sanitized, " ", "-")

	// Remove multiple consecutive hyphens
	re := regexp.MustCompile(`-+`)
	sanitized = re.ReplaceAllString(sanitized, "-")

	// Remove leading/trailing hyphens
	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}

// ParseSessionName parses a session name back into repository and branch
// Format: <repoName>-<branch>
// Returns: repoName, branch, error
func ParseSessionName(sessionName string) (string, string, error) {
	// Find the first hyphen to split repo name and branch
	// We need to be careful because branch names can also contain hyphens
	// For now, we'll use a simple split on the first hyphen
	idx := strings.Index(sessionName, "-")
	if idx == -1 {
		return "", "", eris.Errorf("invalid session name format: %s", sessionName)
	}
	return sessionName[:idx], sessionName[idx+1:], nil
}

// GetRepoNameFromProject extracts the repository name from a full project path
// Example: "github.com/user/repo" -> "repo"
func GetRepoNameFromProject(projectName string) string {
	return filepath.Base(projectName)
}

// GetProjectFromFullPath extracts the project name from a full workspace path
// Example: "/home/user/.sesh/github.com/user/repo/main" -> "github.com/user/repo"
func GetProjectFromFullPath(workspaceDir, fullPath string) (string, error) {
	// Get relative path from workspace directory
	relPath, err := filepath.Rel(workspaceDir, fullPath)
	if err != nil {
		return "", eris.Wrap(err, "path is not within workspace directory")
	}

	// Project name is all components except the last one (which is the worktree/branch)
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) < 2 {
		return "", eris.Errorf("invalid workspace path structure: %s", fullPath)
	}

	// Join all parts except the last one
	projectName := filepath.Join(parts[:len(parts)-1]...)
	return projectName, nil
}

// WorkspaceExists checks if the workspace directory exists
func WorkspaceExists(workspaceDir string) bool {
	info, err := os.Stat(workspaceDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ProjectExists checks if a project directory exists in the workspace
func ProjectExists(workspaceDir, projectName string) bool {
	projectPath := GetProjectPath(workspaceDir, projectName)
	info, err := os.Stat(projectPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// WorktreeExists checks if a worktree directory exists
func WorktreeExists(worktreePath string) bool {
	info, err := os.Stat(worktreePath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ListProjects lists all projects in the workspace directory
// Returns a list of project names (e.g., ["github.com/user/repo1", "github.com/user/repo2"])
func ListProjects(workspaceDir string) ([]string, error) {
	var projects []string

	// Walk the workspace directory looking for .git directories
	err := filepath.Walk(workspaceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check if this is a .git directory (bare repo)
		if info.Name() == ".git" {
			// Get the project path (parent of .git)
			projectPath := filepath.Dir(path)

			// Get relative path from workspace
			relPath, err := filepath.Rel(workspaceDir, projectPath)
			if err != nil {
				return err
			}

			projects = append(projects, relPath)

			// Don't descend into .git directory
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to list projects in workspace")
	}

	return projects, nil
}

// CleanPath cleans and normalizes a path
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// ExpandPath expands ~ to home directory in a path
func ExpandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", eris.Wrap(err, "failed to get home directory")
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}
