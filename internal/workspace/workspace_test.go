package workspace

import (
	"path/filepath"
	"testing"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch",
			input:    "main",
			expected: "main",
		},
		{
			name:     "branch with forward slash",
			input:    "feature/foo",
			expected: "feature-foo",
		},
		{
			name:     "branch with multiple slashes",
			input:    "release/v1.0.0",
			expected: "release-v1.0.0",
		},
		{
			name:     "branch with colon",
			input:    "fix:bug#123",
			expected: "fix-bug-123",
		},
		{
			name:     "branch with multiple special chars",
			input:    "feature/foo:bar#123",
			expected: "feature-foo-bar-123",
		},
		{
			name:     "branch with spaces",
			input:    "my feature branch",
			expected: "my-feature-branch",
		},
		{
			name:     "branch with consecutive hyphens",
			input:    "feature//foo",
			expected: "feature-foo",
		},
		{
			name:     "branch with leading/trailing hyphens",
			input:    "/feature/",
			expected: "feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateSessionName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		branch      string
		expected    string
	}{
		{
			name:        "simple project and branch",
			projectName: "myrepo",
			branch:      "main",
			expected:    "myrepo-main",
		},
		{
			name:        "full github path",
			projectName: "github.com/user/repo",
			branch:      "main",
			expected:    "repo-main",
		},
		{
			name:        "branch with special chars",
			projectName: "github.com/user/repo",
			branch:      "feature/foo",
			expected:    "repo-feature-foo",
		},
		{
			name:        "nested project path",
			projectName: "gitlab.com/org/team/project",
			branch:      "develop",
			expected:    "project-develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSessionName(tt.projectName, tt.branch)
			if result != tt.expected {
				t.Errorf("GenerateSessionName(%q, %q) = %q, want %q", tt.projectName, tt.branch, result, tt.expected)
			}
		})
	}
}

func TestParseSessionName(t *testing.T) {
	tests := []struct {
		name        string
		sessionName string
		wantRepo    string
		wantBranch  string
		wantErr     bool
	}{
		{
			name:        "simple session name",
			sessionName: "repo-main",
			wantRepo:    "repo",
			wantBranch:  "main",
			wantErr:     false,
		},
		{
			name:        "branch with hyphen",
			sessionName: "myproject-feature-foo",
			wantRepo:    "myproject",
			wantBranch:  "feature-foo",
			wantErr:     false,
		},
		{
			name:        "invalid format no hyphen",
			sessionName: "invalid",
			wantErr:     true,
		},
		{
			name:        "branch with multiple hyphens",
			sessionName: "repo-branch-extra",
			wantRepo:    "repo",
			wantBranch:  "branch-extra",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, branch, err := ParseSessionName(tt.sessionName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSessionName(%q) error = %v, wantErr %v", tt.sessionName, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if repo != tt.wantRepo {
					t.Errorf("ParseSessionName(%q) repo = %q, want %q", tt.sessionName, repo, tt.wantRepo)
				}
				if branch != tt.wantBranch {
					t.Errorf("ParseSessionName(%q) branch = %q, want %q", tt.sessionName, branch, tt.wantBranch)
				}
			}
		})
	}
}

func TestGetRepoNameFromProject(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expected    string
	}{
		{
			name:        "simple repo name",
			projectName: "repo",
			expected:    "repo",
		},
		{
			name:        "github path",
			projectName: "github.com/user/repo",
			expected:    "repo",
		},
		{
			name:        "nested path",
			projectName: "gitlab.com/org/team/project",
			expected:    "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRepoNameFromProject(tt.projectName)
			if result != tt.expected {
				t.Errorf("GetRepoNameFromProject(%q) = %q, want %q", tt.projectName, result, tt.expected)
			}
		})
	}
}

func TestGetProjectPath(t *testing.T) {
	tests := []struct {
		name         string
		workspaceDir string
		projectName  string
		expected     string
	}{
		{
			name:         "simple project",
			workspaceDir: "/home/user/.sesh",
			projectName:  "myrepo",
			expected:     "/home/user/.sesh/myrepo",
		},
		{
			name:         "github project",
			workspaceDir: "/home/user/.sesh",
			projectName:  "github.com/user/repo",
			expected:     "/home/user/.sesh/github.com/user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectPath(tt.workspaceDir, tt.projectName)
			if result != tt.expected {
				t.Errorf("GetProjectPath(%q, %q) = %q, want %q", tt.workspaceDir, tt.projectName, result, tt.expected)
			}
		})
	}
}

func TestGetWorktreePath(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		branch      string
		expected    string
	}{
		{
			name:        "simple branch",
			projectPath: "/home/user/.sesh/myrepo",
			branch:      "main",
			expected:    "/home/user/.sesh/myrepo/main",
		},
		{
			name:        "branch with slash",
			projectPath: "/home/user/.sesh/myrepo",
			branch:      "feature/foo",
			expected:    "/home/user/.sesh/myrepo/feature-foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorktreePath(tt.projectPath, tt.branch)
			if result != tt.expected {
				t.Errorf("GetWorktreePath(%q, %q) = %q, want %q", tt.projectPath, tt.branch, result, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "absolute path",
			path:    "/home/user/.sesh",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "relative/path",
			wantErr: false,
		},
		{
			name:    "home directory",
			path:    "~",
			wantErr: false,
		},
		{
			name:    "home directory with path",
			path:    "~/.sesh",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// For paths without ~, result should be the same
				if tt.path[0] != '~' && result != tt.path {
					t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, result, tt.path)
				}
				// For ~ paths, result should not contain ~
				if tt.path[0] == '~' && result[0] == '~' {
					t.Errorf("ExpandPath(%q) = %q, expected ~ to be expanded", tt.path, result)
				}
			}
		})
	}
}

func TestGetProjectFromFullPath(t *testing.T) {
	tests := []struct {
		name         string
		workspaceDir string
		fullPath     string
		expected     string
		wantErr      bool
	}{
		{
			name:         "valid workspace path",
			workspaceDir: "/home/user/.sesh",
			fullPath:     "/home/user/.sesh/github.com/user/repo/main",
			expected:     filepath.Join("github.com", "user", "repo"),
			wantErr:      false,
		},
		{
			name:         "simple project",
			workspaceDir: "/home/user/.sesh",
			fullPath:     "/home/user/.sesh/myrepo/main",
			expected:     "myrepo",
			wantErr:      false,
		},
		{
			name:         "nested project",
			workspaceDir: "/home/user/.sesh",
			fullPath:     "/home/user/.sesh/gitlab.com/org/project/feature-branch",
			expected:     filepath.Join("gitlab.com", "org", "project"),
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetProjectFromFullPath(tt.workspaceDir, tt.fullPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProjectFromFullPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("GetProjectFromFullPath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCleanPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "path with double slashes",
			path:     "/home//user/.sesh",
			expected: "/home/user/.sesh",
		},
		{
			name:     "path with trailing slash",
			path:     "/home/user/.sesh/",
			expected: "/home/user/.sesh",
		},
		{
			name:     "path with dot segments",
			path:     "/home/user/./.sesh/../.sesh",
			expected: "/home/user/.sesh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanPath(tt.path)
			if result != tt.expected {
				t.Errorf("CleanPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}
