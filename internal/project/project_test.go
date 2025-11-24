package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/benoctopus/sesh/internal/db"
	"github.com/benoctopus/sesh/internal/models"
)

func TestNormalizeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "myproject",
			expected: "myproject",
		},
		{
			name:     "name with trailing slash",
			input:    "myproject/",
			expected: "myproject",
		},
		{
			name:     "name with spaces",
			input:    "  myproject  ",
			expected: "myproject",
		},
		{
			name:     "path with multiple slashes",
			input:    "github.com//user//repo",
			expected: filepath.Join("github.com", "user", "repo"),
		},
		{
			name:     "path with dots",
			input:    "github.com/user/./repo",
			expected: filepath.Join("github.com", "user", "repo"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeProjectName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeProjectName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractProjectFromRemote(t *testing.T) {
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
			name:      "GitLab nested groups",
			remoteURL: "https://gitlab.com/org/team/project.git",
			want:      "gitlab.com/org/team/project",
			wantErr:   false,
		},
		{
			name:      "invalid URL",
			remoteURL: "invalid-url",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractProjectFromRemote(tt.remoteURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractProjectFromRemote() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ExtractProjectFromRemote() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveByShortName(t *testing.T) {
	// Create a temporary database for testing
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer database.Close()

	// Create test projects
	projects := []*models.Project{
		{
			Name:      "github.com/user/sesh",
			RemoteURL: "git@github.com:user/sesh.git",
			LocalPath: "/test/path1/.git",
		},
		{
			Name:      "gitlab.com/org/sesh",
			RemoteURL: "git@gitlab.com:org/sesh.git",
			LocalPath: "/test/path2/.git",
		},
		{
			Name:      "github.com/user/myproject",
			RemoteURL: "git@github.com:user/myproject.git",
			LocalPath: "/test/path3/.git",
		},
	}

	for _, proj := range projects {
		if err := db.CreateProject(database, proj); err != nil {
			t.Fatalf("Failed to create test project: %v", err)
		}
	}

	tests := []struct {
		name      string
		shortName string
		wantErr   bool
		wantName  string // Expected full project name, or empty if error
		errMsg    string // Expected error message substring
	}{
		{
			name:      "unique short name",
			shortName: "myproject",
			wantErr:   false,
			wantName:  "github.com/user/myproject",
		},
		{
			name:      "duplicate short name",
			shortName: "sesh",
			wantErr:   true,
			errMsg:    "multiple projects found",
		},
		{
			name:      "non-existent short name",
			shortName: "nonexistent",
			wantErr:   true,
			errMsg:    "no project found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj, err := resolveByShortName(database, tt.shortName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveByShortName() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("resolveByShortName() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("resolveByShortName() unexpected error: %v", err)
				return
			}

			if proj.Name != tt.wantName {
				t.Errorf("resolveByShortName() project name = %q, want %q", proj.Name, tt.wantName)
			}
		})
	}
}
