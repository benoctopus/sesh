package git

import (
	"bufio"
	"os/exec"
	"strings"

	"github.com/rotisserie/eris"
)

// BranchInfo contains information about a branch
type BranchInfo struct {
	Name      string
	IsRemote  bool
	IsCurrent bool
}

// ListLocalBranches lists all local branches in a repository
func ListLocalBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, eris.Wrap(err, "failed to list local branches")
	}

	return parseGitBranchList(string(output)), nil
}

// ListRemoteBranches lists all remote branches in a repository
// Returns branch names without the "origin/" prefix (e.g., "main" instead of "origin/main")
func ListRemoteBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "-r", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, eris.Wrap(err, "failed to list remote branches")
	}

	branches := parseGitBranchList(string(output))

	// Remove "origin/" prefix and filter out HEAD
	var result []string
	for _, branch := range branches {
		if strings.Contains(branch, "HEAD") {
			continue
		}
		// Remove "origin/" prefix
		if strings.HasPrefix(branch, "origin/") {
			result = append(result, strings.TrimPrefix(branch, "origin/"))
		} else {
			result = append(result, branch)
		}
	}

	return result, nil
}

// ListAllBranches lists both local and remote branches
// Returns a list of BranchInfo with details about each branch
func ListAllBranches(repoPath string) ([]BranchInfo, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "-a", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, eris.Wrap(err, "failed to list all branches")
	}

	// Get current branch
	currentBranch, _ := GetCurrentBranch(repoPath)

	branches := parseGitBranchList(string(output))
	var result []BranchInfo

	seen := make(map[string]bool)

	for _, branch := range branches {
		if strings.Contains(branch, "HEAD") {
			continue
		}

		isRemote := strings.HasPrefix(branch, "origin/")
		branchName := branch
		if isRemote {
			branchName = strings.TrimPrefix(branch, "origin/")
		}

		// Skip duplicates (local and remote with same name)
		if seen[branchName] {
			continue
		}
		seen[branchName] = true

		result = append(result, BranchInfo{
			Name:      branchName,
			IsRemote:  isRemote,
			IsCurrent: branchName == currentBranch,
		})
	}

	return result, nil
}

// DoesBranchExist checks if a branch exists (local or remote)
func DoesBranchExist(repoPath, branch string) (bool, error) {
	// Check local branch first
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "refs/heads/"+branch)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	// Check remote branch
	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "refs/remotes/origin/"+branch)
	err = cmd.Run()
	if err == nil {
		return true, nil
	}

	// Check if it's just a ref that doesn't exist or an actual error
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
		return false, nil
	}

	return false, nil
}

// GetCurrentBranch retrieves the current branch name in a git repository
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get current branch")
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "(detached)", nil
	}
	return branch, nil
}

// DoesLocalBranchExist checks if a local branch exists
func DoesLocalBranchExist(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "refs/heads/"+branch)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return false, nil
		}
		return false, eris.Wrap(err, "failed to check local branch existence")
	}
	return true, nil
}

// DoesRemoteBranchExist checks if a remote branch exists
func DoesRemoteBranchExist(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "refs/remotes/origin/"+branch)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return false, nil
		}
		return false, eris.Wrap(err, "failed to check remote branch existence")
	}
	return true, nil
}

// parseGitBranchList parses the output of git branch commands
func parseGitBranchList(output string) []string {
	var branches []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches
}
