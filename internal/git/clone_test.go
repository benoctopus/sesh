package git

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		wantHost  string
		wantOrg   string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "SSH URL simple",
			remoteURL: "git@github.com:user/repo.git",
			wantHost:  "github.com",
			wantOrg:   "user",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "SSH URL without .git",
			remoteURL: "git@github.com:user/repo",
			wantHost:  "github.com",
			wantOrg:   "user",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "HTTPS URL simple",
			remoteURL: "https://github.com/user/repo.git",
			wantHost:  "github.com",
			wantOrg:   "user",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "HTTPS URL without .git",
			remoteURL: "https://github.com/user/repo",
			wantHost:  "github.com",
			wantOrg:   "user",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "GitLab nested organization",
			remoteURL: "https://gitlab.com/org/subgroup/project.git",
			wantHost:  "gitlab.com",
			wantOrg:   "org/subgroup",
			wantRepo:  "project",
			wantErr:   false,
		},
		{
			name:      "SSH GitLab nested",
			remoteURL: "git@gitlab.com:org/subgroup/project.git",
			wantHost:  "gitlab.com",
			wantOrg:   "org/subgroup",
			wantRepo:  "project",
			wantErr:   false,
		},
		{
			name:      "Bitbucket URL",
			remoteURL: "https://bitbucket.org/team/repo.git",
			wantHost:  "bitbucket.org",
			wantOrg:   "team",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "invalid SSH format",
			remoteURL: "git@github.com",
			wantErr:   true,
		},
		{
			name:      "invalid HTTPS path",
			remoteURL: "https://github.com/repo",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, org, repo, err := ParseRemoteURL(tt.remoteURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRemoteURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if host != tt.wantHost {
					t.Errorf("ParseRemoteURL() host = %q, want %q", host, tt.wantHost)
				}
				if org != tt.wantOrg {
					t.Errorf("ParseRemoteURL() org = %q, want %q", org, tt.wantOrg)
				}
				if repo != tt.wantRepo {
					t.Errorf("ParseRemoteURL() repo = %q, want %q", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestGenerateProjectName(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		want      string
		wantErr   bool
	}{
		{
			name:      "GitHub SSH",
			remoteURL: "git@github.com:user/repo.git",
			want:      "github.com/user/repo",
			wantErr:   false,
		},
		{
			name:      "GitHub HTTPS",
			remoteURL: "https://github.com/user/repo.git",
			want:      "github.com/user/repo",
			wantErr:   false,
		},
		{
			name:      "GitLab nested",
			remoteURL: "https://gitlab.com/org/subgroup/project.git",
			want:      "gitlab.com/org/subgroup/project",
			wantErr:   false,
		},
		{
			name:      "Bitbucket",
			remoteURL: "https://bitbucket.org/team/repo.git",
			want:      "bitbucket.org/team/repo",
			wantErr:   false,
		},
		{
			name:      "invalid URL",
			remoteURL: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateProjectName(tt.remoteURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateProjectName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("GenerateProjectName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseGitBranchList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple branch list",
			input:    "main\nfeature\ndevelop\n",
			expected: []string{"main", "feature", "develop"},
		},
		{
			name:     "empty output",
			input:    "",
			expected: nil,
		},
		{
			name:     "single branch",
			input:    "main\n",
			expected: []string{"main"},
		},
		{
			name:     "branches with spaces",
			input:    "  main  \n  feature  \n",
			expected: []string{"main", "feature"},
		},
		{
			name:     "branches with empty lines",
			input:    "main\n\nfeature\n\n",
			expected: []string{"main", "feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGitBranchList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf(
					"parseGitBranchList() returned %d items, want %d",
					len(result),
					len(tt.expected),
				)
				return
			}
			for i, branch := range result {
				if branch != tt.expected[i] {
					t.Errorf("parseGitBranchList()[%d] = %q, want %q", i, branch, tt.expected[i])
				}
			}
		})
	}
}
