package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	// Get the actual home directory for testing
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "tilde only",
			path:     "~",
			wantPath: home,
			wantErr:  false,
		},
		{
			name:     "tilde with path",
			path:     "~/.sesh",
			wantPath: filepath.Join(home, ".sesh"),
			wantErr:  false,
		},
		{
			name:     "absolute path",
			path:     "/absolute/path",
			wantPath: "/absolute/path",
			wantErr:  false,
		},
		{
			name:     "relative path",
			path:     "relative/path",
			wantPath: "relative/path",
			wantErr:  false,
		},
		{
			name:     "empty path",
			path:     "",
			wantPath: "",
			wantErr:  false,
		},
		{
			name:     "tilde in middle",
			path:     "path/~/file",
			wantPath: "path/~/file",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandHome(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandHome(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.wantPath {
				t.Errorf("expandHome(%q) = %q, want %q", tt.path, result, tt.wantPath)
			}
		})
	}
}

func TestGetConfigDir(t *testing.T) {
	configDir, err := GetConfigDir()
	if err != nil {
		t.Fatalf("GetConfigDir() returned error: %v", err)
	}

	if configDir == "" {
		t.Error("GetConfigDir() returned empty string")
	}

	// Verify the path ends with "sesh"
	if filepath.Base(configDir) != "sesh" {
		t.Errorf("GetConfigDir() path doesn't end with 'sesh': %s", configDir)
	}
}

func TestGetDBPath(t *testing.T) {
	dbPath, err := GetDBPath()
	if err != nil {
		t.Fatalf("GetDBPath() returned error: %v", err)
	}

	if dbPath == "" {
		t.Error("GetDBPath() returned empty string")
	}

	// Verify the path ends with "sesh.db"
	if filepath.Base(dbPath) != "sesh.db" {
		t.Errorf("GetDBPath() path doesn't end with 'sesh.db': %s", dbPath)
	}

	// Verify parent directory is named "sesh"
	parentDir := filepath.Dir(dbPath)
	if filepath.Base(parentDir) != "sesh" {
		t.Errorf("GetDBPath() parent directory is not 'sesh': %s", parentDir)
	}
}

func TestGetWorkspaceDir(t *testing.T) {
	// Save and restore original environment
	originalEnv := os.Getenv("SESH_WORKSPACE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("SESH_WORKSPACE", originalEnv)
		} else {
			os.Unsetenv("SESH_WORKSPACE")
		}
	}()

	t.Run("with environment variable", func(t *testing.T) {
		testPath := "/tmp/test-workspace"
		os.Setenv("SESH_WORKSPACE", testPath)

		workspaceDir, err := GetWorkspaceDir()
		if err != nil {
			t.Fatalf("GetWorkspaceDir() returned error: %v", err)
		}

		if workspaceDir != testPath {
			t.Errorf("GetWorkspaceDir() = %q, want %q", workspaceDir, testPath)
		}
	})

	t.Run("with tilde in environment variable", func(t *testing.T) {
		os.Setenv("SESH_WORKSPACE", "~/test-workspace")

		workspaceDir, err := GetWorkspaceDir()
		if err != nil {
			t.Fatalf("GetWorkspaceDir() returned error: %v", err)
		}

		if workspaceDir[0] == '~' {
			t.Errorf("GetWorkspaceDir() didn't expand tilde: %s", workspaceDir)
		}
	})

	t.Run("default workspace", func(t *testing.T) {
		os.Unsetenv("SESH_WORKSPACE")

		workspaceDir, err := GetWorkspaceDir()
		if err != nil {
			t.Fatalf("GetWorkspaceDir() returned error: %v", err)
		}

		if workspaceDir == "" {
			t.Error("GetWorkspaceDir() returned empty string")
		}

		// Should end with .sesh
		if filepath.Base(workspaceDir) != ".sesh" {
			t.Errorf("GetWorkspaceDir() default doesn't end with '.sesh': %s", workspaceDir)
		}
	})
}

func TestGetSessionBackend(t *testing.T) {
	// Save and restore original environment
	originalEnv := os.Getenv("SESH_SESSION_BACKEND")
	defer func() {
		if originalEnv != "" {
			os.Setenv("SESH_SESSION_BACKEND", originalEnv)
		} else {
			os.Unsetenv("SESH_SESSION_BACKEND")
		}
	}()

	t.Run("with environment variable", func(t *testing.T) {
		os.Setenv("SESH_SESSION_BACKEND", "tmux")

		backend, err := GetSessionBackend()
		if err != nil {
			t.Fatalf("GetSessionBackend() returned error: %v", err)
		}

		if backend != "tmux" {
			t.Errorf("GetSessionBackend() = %q, want %q", backend, "tmux")
		}
	})

	t.Run("default backend", func(t *testing.T) {
		os.Unsetenv("SESH_SESSION_BACKEND")

		backend, err := GetSessionBackend()
		if err != nil {
			t.Fatalf("GetSessionBackend() returned error: %v", err)
		}

		if backend != "auto" {
			t.Errorf("GetSessionBackend() = %q, want %q", backend, "auto")
		}
	})
}

func TestLoadConfig(t *testing.T) {
	// Save and restore original environment
	originalWorkspace := os.Getenv("SESH_WORKSPACE")
	originalBackend := os.Getenv("SESH_SESSION_BACKEND")
	defer func() {
		if originalWorkspace != "" {
			os.Setenv("SESH_WORKSPACE", originalWorkspace)
		} else {
			os.Unsetenv("SESH_WORKSPACE")
		}
		if originalBackend != "" {
			os.Setenv("SESH_SESSION_BACKEND", originalBackend)
		} else {
			os.Unsetenv("SESH_SESSION_BACKEND")
		}
	}()

	t.Run("load default config", func(t *testing.T) {
		os.Unsetenv("SESH_WORKSPACE")
		os.Unsetenv("SESH_SESSION_BACKEND")

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() returned error: %v", err)
		}

		if config == nil {
			t.Fatal("LoadConfig() returned nil config")
		}

		if config.WorkspaceDir == "" {
			t.Error("LoadConfig() returned empty WorkspaceDir")
		}

		if config.SessionBackend == "" {
			t.Error("LoadConfig() returned empty SessionBackend")
		}
	})

	t.Run("load config with environment variables", func(t *testing.T) {
		os.Setenv("SESH_WORKSPACE", "/tmp/test-workspace")
		os.Setenv("SESH_SESSION_BACKEND", "tmux")

		config, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() returned error: %v", err)
		}

		if config.WorkspaceDir != "/tmp/test-workspace" {
			t.Errorf("LoadConfig() WorkspaceDir = %q, want %q", config.WorkspaceDir, "/tmp/test-workspace")
		}

		if config.SessionBackend != "tmux" {
			t.Errorf("LoadConfig() SessionBackend = %q, want %q", config.SessionBackend, "tmux")
		}
	})
}
