package git

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
)

// Clone clones a git repository as a bare repository to the specified destination path
func Clone(remoteURL, destPath string) error {
	cmd := exec.Command("git", "clone", "--bare", remoteURL, destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to clone repository: %s", string(output))
	}
	return nil
}

// GetRemoteURL retrieves the remote URL from a git repository
func GetRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get remote URL")
	}
	return strings.TrimSpace(string(output)), nil
}

// ParseRemoteURL parses a git remote URL and extracts the host, organization, and repository name
// Supports both SSH and HTTPS URLs
// Examples:
//   - git@github.com:user/repo.git -> github.com, user, repo
//   - https://github.com/user/repo.git -> github.com, user, repo
//   - https://gitlab.com/org/subgroup/project.git -> gitlab.com, org/subgroup, project
func ParseRemoteURL(remoteURL string) (host, org, repo string, err error) {
	// Handle SSH URLs (git@host:path)
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) != 2 {
			return "", "", "", eris.Errorf("invalid SSH URL format: %s", remoteURL)
		}
		host = strings.TrimPrefix(parts[0], "git@")
		path := strings.TrimSuffix(parts[1], ".git")

		// Split path into org and repo
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return "", "", "", eris.Errorf("invalid repository path: %s", path)
		}
		repo = pathParts[len(pathParts)-1]
		org = strings.Join(pathParts[:len(pathParts)-1], "/")

		return host, org, repo, nil
	}

	// Handle HTTPS URLs
	parsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", "", eris.Wrap(err, "failed to parse remote URL")
	}

	host = parsedURL.Host
	path := strings.TrimPrefix(parsedURL.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return "", "", "", eris.Errorf("invalid repository path: %s", path)
	}

	repo = pathParts[len(pathParts)-1]
	org = strings.Join(pathParts[:len(pathParts)-1], "/")

	return host, org, repo, nil
}

// GenerateProjectName generates a project name from a remote URL
// Format: host/org/repo (e.g., "github.com/user/repo")
func GenerateProjectName(remoteURL string) (string, error) {
	host, org, repo, err := ParseRemoteURL(remoteURL)
	if err != nil {
		return "", err
	}
	return filepath.Join(host, org, repo), nil
}

// Fetch fetches the latest changes from the remote repository
func Fetch(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "fetch", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to fetch from remote: %s", string(output))
	}
	return nil
}

// GetDefaultBranch retrieves the default branch name from a repository
func GetDefaultBranch(repoPath string) (string, error) {
	// Query the symbolic ref for the remote HEAD
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try to detect from common branch names
		for _, branch := range []string{"main", "master", "develop"} {
			if exists, _ := doesRefExist(repoPath, fmt.Sprintf("refs/remotes/origin/%s", branch)); exists {
				return branch, nil
			}
		}
		return "", eris.Wrap(err, "failed to determine default branch")
	}

	// Parse the ref (e.g., "refs/remotes/origin/main" -> "main")
	ref := strings.TrimSpace(string(output))
	parts := strings.Split(ref, "/")
	if len(parts) < 1 {
		return "", eris.Errorf("invalid default branch ref: %s", ref)
	}

	return parts[len(parts)-1], nil
}

// doesRefExist checks if a git ref exists in the repository
func doesRefExist(repoPath, ref string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", ref)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, eris.Wrap(err, "failed to check ref existence")
	}
	return true, nil
}
