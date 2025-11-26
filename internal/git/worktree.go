package git

import (
	"bufio"
	"os/exec"
	"strings"

	"github.com/rotisserie/eris"
)

// WorktreeInfo contains information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
}

// CreateWorktree creates a new worktree for a branch that exists in the repository
// For bare repositories (which sesh uses), branches are stored at refs/heads/<branch>
// This sets up tracking to origin/<branch> for pushing
func CreateWorktree(repoPath, branch, worktreePath string) error {
	// Create the worktree
	cmd := exec.Command(
		"git",
		"-C",
		repoPath,
		"worktree",
		"add",
		worktreePath,
		branch,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to create worktree: %s", string(output))
	}

	// Set up tracking to origin/<branch>
	// In bare repos, we need to manually configure the tracking since there are no
	// remote-tracking branches (refs/remotes/origin/*). We set the config directly.
	cmd = exec.Command(
		"git",
		"-C",
		worktreePath,
		"config",
		"branch."+branch+".remote",
		"origin",
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to set branch remote: %s", string(output))
	}

	cmd = exec.Command(
		"git",
		"-C",
		worktreePath,
		"config",
		"branch."+branch+".merge",
		"refs/heads/"+branch,
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to set branch merge: %s", string(output))
	}

	return nil
}

// CreateWorktreeFromLocalBranch creates a new worktree for a branch that already exists locally
func CreateWorktreeFromLocalBranch(repoPath, branch, worktreePath string) error {
	cmd := exec.Command(
		"git",
		"-C",
		repoPath,
		"worktree",
		"add",
		worktreePath,
		branch,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to create worktree from local branch: %s", string(output))
	}
	return nil
}

// CreateWorktreeNewBranch creates a new worktree with a new branch
// This is equivalent to: git worktree add -b <branch> <path> <start-point>
func CreateWorktreeNewBranch(repoPath, branch, worktreePath, startPoint string) error {
	cmd := exec.Command(
		"git",
		"-C",
		repoPath,
		"worktree",
		"add",
		"-b",
		branch,
		worktreePath,
		startPoint,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to create worktree with new branch: %s", string(output))
	}
	return nil
}

// CreateWorktreeFromRef creates a new worktree from a specific ref (commit, tag, etc.)
func CreateWorktreeFromRef(repoPath, ref, worktreePath string) error {
	cmd := exec.Command(
		"git",
		"-C",
		repoPath,
		"worktree",
		"add",
		"--guess-remote",
		"-b",
		ref,
		worktreePath,
		"origin/"+ref,
		"--track",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to create worktree from ref: %s", string(output))
	}
	return nil
}

// ListWorktrees lists all worktrees for a repository
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, eris.Wrap(err, "failed to list worktrees")
	}

	return parseWorktreeList(string(output))
}

// parseWorktreeList parses the output of 'git worktree list --porcelain'
// Format:
// worktree /path/to/worktree
// HEAD <commit>
// branch refs/heads/main
//
// worktree /path/to/another
// HEAD <commit>
// detached
func parseWorktreeList(output string) ([]WorktreeInfo, error) {
	var worktrees []WorktreeInfo
	var current WorktreeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// Empty line marks end of a worktree entry
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = WorktreeInfo{}
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "worktree":
			current.Path = value
		case "HEAD":
			current.Commit = value
		case "branch":
			// Extract branch name from ref (e.g., "refs/heads/main" -> "main")
			branchRef := value
			if strings.HasPrefix(branchRef, "refs/heads/") {
				current.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
			}
		case "detached":
			current.Branch = "(detached)"
		}
	}

	// Add last entry if exists
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, eris.Wrap(err, "failed to parse worktree list")
	}

	return worktrees, nil
}

// RemoveWorktree removes a worktree
func RemoveWorktree(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to remove worktree: %s", string(output))
	}
	return nil
}

// RemoveWorktreeForce forcefully removes a worktree (even if it has uncommitted changes)
func RemoveWorktreeForce(repoPath, worktreePath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "remove", "--force", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to force remove worktree: %s", string(output))
	}
	return nil
}

// GetWorktreeBranch retrieves the current branch name for a worktree
func GetWorktreeBranch(worktreePath string) (string, error) {
	cmd := exec.Command("git", "-C", worktreePath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get worktree branch")
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "(detached)", nil
	}
	return branch, nil
}

// PruneWorktrees removes worktree information for directories that no longer exist
func PruneWorktrees(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return eris.Wrapf(err, "failed to prune worktrees: %s", string(output))
	}
	return nil
}
