package git

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/rotisserie/eris"
)

// BranchInfo contains information about a branch
type BranchInfo struct {
	Name      string
	IsRemote  bool
	IsCurrent bool
}

var branchListSpecialChars = regexp.MustCompile(`[+*]`)

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
			if trimmed, ok := strings.CutPrefix(branch, "origin/"); ok {
				result = append(result, trimmed)
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
		"ls-remote",
		"--branches",
		"--tags",
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

		local, err := ListLocalBranches(repoPath)
		if err != nil {
			cancel(eris.Wrap(err, "failed to list local branches"))
		}

		localSet := make(map[string]struct{})
		for _, branch := range local {
			localSet[branch] = struct{}{}
			fmt.Fprintln(writer, branch) //nolint:errcheck
		}

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
		skip := true
		for scanner.Scan() {
			if ctx.Err() != nil {
				return // Context cancelled
			}

			if skip {
				skip = false
				continue
			}

			fields := strings.Fields(strings.TrimSpace(scanner.Text()))
			if len(fields) < 2 {
				continue
			}

			branch := strings.TrimPrefix(strings.TrimSpace(fields[1]), "refs/heads/")
			if branch != "" && !strings.Contains(branch, "HEAD") {
				// Remove "origin/" prefix if present
				if after, ok := strings.CutPrefix(branch, "origin/"); ok {
					branch = after
				}

				// this should not pose a concurrent access issue as all other writes are done before starting the goroutine.
				if _, exists := localSet[branch]; exists {
					continue // Skip local branches
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

// DoesBranchExist checks if a branch exists in the repository
// For bare repositories (which sesh uses), all branches are at refs/heads/*
// Returns (exists, isRemote, error) - isRemote is always false for bare repos
func DoesBranchExist(repoPath, branch string) (bool, bool, error) {
	// Check if branch exists at refs/heads/<branch>
	// This works for both bare repos and regular repos with local branches
	cmd := exec.Command(
		"git",
		"-C",
		repoPath,
		"show-ref",
		"--verify",
		"--quiet",
		"refs/heads/"+branch,
	)

	err := cmd.Run()
	if err == nil {
		// Branch exists at refs/heads/<branch>
		return true, false, nil
	}

	// Check if it's a non-existent ref (exit code 1) or an actual error
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 1 {
			// Branch doesn't exist
			return false, false, nil
		}
	}

	// Some other error occurred
	return false, false, eris.Wrap(err, "failed to check branch existence")
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
