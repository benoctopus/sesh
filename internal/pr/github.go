package pr

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rotisserie/eris"
)

// GitHubProvider implements the Provider interface for GitHub
type GitHubProvider struct{}

// NewGitHubProvider creates a new GitHub provider instance
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{}
}

// Name returns the provider name
func (g *GitHubProvider) Name() string {
	return "github"
}

// ghPullRequest represents the JSON structure returned by gh pr list
type ghPullRequest struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
	State     string    `json:"state"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Body      string    `json:"body"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// ListOpenPRs lists all open pull requests for the repository
func (g *GitHubProvider) ListOpenPRs(ctx context.Context, repoPath string) ([]*PullRequest, error) {
	// Use gh CLI to list open PRs
	// gh pr list --json number,title,headRefName,baseRefName,author,state,url,createdAt,updatedAt,body,labels
	cmd := exec.CommandContext(
		ctx,
		"gh", "pr", "list",
		"--json", "number,title,headRefName,baseRefName,author,state,url,createdAt,updatedAt,body,labels",
		"--state", "open",
	)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, eris.Wrapf(
				err,
				"gh command failed: %s",
				string(exitErr.Stderr),
			)
		}
		return nil, eris.Wrap(err, "failed to execute gh command")
	}

	var ghPRs []ghPullRequest
	if err := json.Unmarshal(output, &ghPRs); err != nil {
		return nil, eris.Wrap(err, "failed to parse gh output")
	}

	// Convert to our PullRequest type
	prs := make([]*PullRequest, len(ghPRs))
	for i, ghPR := range ghPRs {
		labels := make([]string, len(ghPR.Labels))
		for j, label := range ghPR.Labels {
			labels[j] = label.Name
		}

		prs[i] = &PullRequest{
			Number:      ghPR.Number,
			Title:       ghPR.Title,
			Branch:      ghPR.HeadRefName,
			BaseBranch:  ghPR.BaseRefName,
			Author:      ghPR.Author.Login,
			State:       strings.ToLower(ghPR.State),
			URL:         ghPR.URL,
			CreatedAt:   ghPR.CreatedAt,
			UpdatedAt:   ghPR.UpdatedAt,
			Description: ghPR.Body,
			Labels:      labels,
		}
	}

	return prs, nil
}

// GetPR retrieves a specific pull request by number
func (g *GitHubProvider) GetPR(ctx context.Context, repoPath string, number int) (*PullRequest, error) {
	// Use gh CLI to view a specific PR
	cmd := exec.CommandContext(
		ctx,
		"gh", "pr", "view", strconv.Itoa(number),
		"--json", "number,title,headRefName,baseRefName,author,state,url,createdAt,updatedAt,body,labels",
	)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, eris.Wrapf(
				err,
				"gh command failed: %s",
				string(exitErr.Stderr),
			)
		}
		return nil, eris.Wrap(err, "failed to execute gh command")
	}

	var ghPR ghPullRequest
	if err := json.Unmarshal(output, &ghPR); err != nil {
		return nil, eris.Wrap(err, "failed to parse gh output")
	}

	labels := make([]string, len(ghPR.Labels))
	for i, label := range ghPR.Labels {
		labels[i] = label.Name
	}

	return &PullRequest{
		Number:      ghPR.Number,
		Title:       ghPR.Title,
		Branch:      ghPR.HeadRefName,
		BaseBranch:  ghPR.BaseRefName,
		Author:      ghPR.Author.Login,
		State:       strings.ToLower(ghPR.State),
		URL:         ghPR.URL,
		CreatedAt:   ghPR.CreatedAt,
		UpdatedAt:   ghPR.UpdatedAt,
		Description: ghPR.Body,
		Labels:      labels,
	}, nil
}

// GetPRBranch returns the branch name for a given PR number
func (g *GitHubProvider) GetPRBranch(ctx context.Context, repoPath string, number int) (string, error) {
	pr, err := g.GetPR(ctx, repoPath, number)
	if err != nil {
		return "", eris.Wrapf(err, "failed to get PR #%d", number)
	}
	return pr.Branch, nil
}

// CheckGHCLI checks if the gh CLI is installed and authenticated
func CheckGHCLI() error {
	// Check if gh is installed
	cmd := exec.Command("gh", "--version")
	if err := cmd.Run(); err != nil {
		return eris.New("gh CLI not found. Install it from https://cli.github.com/")
	}

	// Check if gh is authenticated
	cmd = exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return eris.New("gh CLI not authenticated. Run 'gh auth login' to authenticate")
	}

	return nil
}

// FormatPRForDisplay formats a pull request for display in the terminal
func FormatPRForDisplay(pr *PullRequest) string {
	// Format: #123 Title (branch) @author
	return fmt.Sprintf("#%d %s (%s) @%s", pr.Number, pr.Title, pr.Branch, pr.Author)
}

// FormatPRForFuzzyFinder formats a pull request for the fuzzy finder
// Returns a string that can be parsed back to extract the PR number
func FormatPRForFuzzyFinder(pr *PullRequest) string {
	// Format: #123│Title│branch│@author
	// Using │ as delimiter to make parsing easier
	return fmt.Sprintf("#%d│%s│%s│@%s", pr.Number, pr.Title, pr.Branch, pr.Author)
}

// ParsePRNumber extracts the PR number from a fuzzy finder selection
func ParsePRNumber(selection string) (int, error) {
	// Extract number from "#123│..." format
	if !strings.HasPrefix(selection, "#") {
		return 0, eris.Errorf("invalid PR selection format: %s", selection)
	}

	// Find the first │ delimiter
	parts := strings.SplitN(selection, "│", 2)
	if len(parts) == 0 {
		return 0, eris.Errorf("invalid PR selection format: %s", selection)
	}

	// Parse the number (remove # prefix)
	numStr := strings.TrimPrefix(parts[0], "#")
	number, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, eris.Wrapf(err, "failed to parse PR number from: %s", selection)
	}

	return number, nil
}
