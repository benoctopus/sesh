package project

import (
	"path/filepath"
	"testing"
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
