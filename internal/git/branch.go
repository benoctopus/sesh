package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
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
// For bare repositories, lists branches from refs/heads/
func ListRemoteBranches(repoPath string) ([]string, error) {
	// Try listing branches using for-each-ref which works for both bare and normal repos
	cmd := exec.Command(
		"git",
		"-C",
		repoPath,
		"for-each-ref",
		"--format=%(refname:short)",
		"refs/heads/",
	)
	output, err := cmd.Output()
	if err != nil {
		// Fallback to branch -r for normal repos
		cmd = exec.Command("git", "-C", repoPath, "branch", "-r", "--format=%(refname:short)")
		output, err = cmd.Output()
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

	// For bare repos, branches are directly under refs/heads/
	return parseGitBranchList(string(output)), nil
}

// StreamRemoteBranches returns a reader that streams branch names and the cleanup function
// The reader will output one branch name per line as git produces them
// The caller must call cleanup() when done to ensure the process terminates
func StreamRemoteBranches(ctx context.Context, repoPath string) (io.ReadCloser, error) {
	// Use for-each-ref which works for both bare and normal repos
	cmd := exec.CommandContext(
		ctx,
		"git",
		"-C",
		repoPath,
		"for-each-ref",
		"--format=%(refname:short)",
		"refs/heads/",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, eris.Wrap(err, "failed to create stdout pipe")
	}

	if err := cmd.Start(); err != nil {
		return nil, eris.Wrap(err, "failed to start git command")
	}

	// Create a pipe that will transform the output
	reader, writer := io.Pipe()

	// Transform output in a goroutine
	go func() {
		ctx, cancel := context.WithCancelCause(ctx)
		defer cancel(nil)

		defer func() {
			if err := writer.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing writer: %v\n", err)
			}
			if err := stdout.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing stdout: %v\n", err)
			}
		}()

		defer writer.Close() //nolint:errcheck
		defer stdout.Close() //nolint:errcheck

		// Wait for command to finish when done reading
		defer func() {
			var err error
			defer cancel(err)

			if err = cmd.Wait(); err != nil {
				fmt.Fprintf(
					os.Stderr,
					"Git command error: %s\n",
					eris.ToString(err, true),
				)
			}
		}()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if ctx.Err() != nil {
				return // Context cancelled
			}

			branch := strings.TrimSpace(scanner.Text())
			if branch != "" && !strings.Contains(branch, "HEAD") {
				// Remove "origin/" prefix if present
				if after, ok := strings.CutPrefix(branch, "origin/"); ok {
					branch = after
				}
				fmt.Fprintln(writer, branch) //nolint:errcheck
			}
		}
	}()

	return reader, nil
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
	cmd = exec.Command(
		"git",
		"-C",
		repoPath,
		"rev-parse",
		"--verify",
		"refs/remotes/origin/"+branch,
	)
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
// For bare repositories, branches are at refs/heads/ not refs/remotes/origin/
func DoesRemoteBranchExist(repoPath, branch string) (bool, error) {
	// First try refs/heads/ (for bare repos)
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "refs/heads/"+branch)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	// Then try refs/remotes/origin/ (for normal repos)
	cmd = exec.Command(
		"git",
		"-C",
		repoPath,
		"rev-parse",
		"--verify",
		"refs/remotes/origin/"+branch,
	)
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
