package pr

import (
	"context"
	"time"

	"github.com/rotisserie/eris"
)

// PullRequest represents a pull request from any provider
type PullRequest struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Branch      string    `json:"branch"`      // Head branch name
	BaseBranch  string    `json:"base_branch"` // Base/target branch
	Author      string    `json:"author"`
	State       string    `json:"state"` // open, closed, merged
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Description string    `json:"description,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
}

// Provider defines the interface for pull request providers (GitHub, GitLab, etc.)
type Provider interface {
	// Name returns the provider name (e.g., "github", "gitlab")
	Name() string

	// ListOpenPRs lists all open pull requests for the repository
	ListOpenPRs(ctx context.Context, repoPath string) ([]*PullRequest, error)

	// GetPR retrieves a specific pull request by number
	GetPR(ctx context.Context, repoPath string, number int) (*PullRequest, error)

	// GetPRBranch returns the branch name for a given PR number
	GetPRBranch(ctx context.Context, repoPath string, number int) (string, error)
}

// ProviderType represents the type of git hosting provider
type ProviderType string

const (
	ProviderTypeGitHub  ProviderType = "github"
	ProviderTypeGitLab  ProviderType = "gitlab"
	ProviderTypeUnknown ProviderType = "unknown"
)

// DetectProvider detects the provider type from a git remote URL
func DetectProvider(remoteURL string) ProviderType {
	// Check for GitHub
	if contains(remoteURL, "github.com") {
		return ProviderTypeGitHub
	}

	// Check for GitLab
	if contains(remoteURL, "gitlab.com") {
		return ProviderTypeGitLab
	}

	return ProviderTypeUnknown
}

// NewProvider creates a new provider instance based on the remote URL
func NewProvider(remoteURL string) (Provider, error) {
	providerType := DetectProvider(remoteURL)

	switch providerType {
	case ProviderTypeGitHub:
		return NewGitHubProvider(), nil
	case ProviderTypeGitLab:
		return nil, eris.New("GitLab provider not yet implemented")
	default:
		return nil, eris.Errorf("unsupported git provider for URL: %s", remoteURL)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
