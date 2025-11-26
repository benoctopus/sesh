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

	// Configure the bare repo to create remote-tracking branches (refs/remotes/origin/*)
	// This is necessary for git status to show ahead/behind tracking information in worktrees
	// By default, bare repos don't have a fetch refspec configured
	cmd = exec.Command("git", "-C", destPath, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to configure remote fetch: %s", string(output))
	}

	// Fetch to populate the remote-tracking branches
	cmd = exec.Command("git", "-C", destPath, "fetch", "origin")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to fetch remote branches: %s", string(output))
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
// For bare repositories (which sesh uses), this checks the symbolic ref HEAD
func GetDefaultBranch(repoPath string) (string, error) {
	// In bare repos, HEAD points directly to refs/heads/<branch>
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "HEAD")
	output, err := cmd.Output()
	if err == nil {
		// Parse the ref (e.g., "refs/heads/main" -> "main")
		ref := strings.TrimSpace(string(output))
		parts := strings.Split(ref, "/")
		if len(parts) >= 3 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: try to detect from common branch names at refs/heads/*
	for _, branch := range []string{"main", "master", "develop"} {
		if exists, _ := doesRefExist(repoPath, fmt.Sprintf("refs/heads/%s", branch)); exists {
			return branch, nil
		}
	}

	return "", eris.New("failed to determine default branch")
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
