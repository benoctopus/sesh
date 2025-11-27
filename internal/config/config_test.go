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
	// Create a temporary directory for isolated testing
	tempHome := t.TempDir()

	// Save and restore original HOME
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp directory to isolate from real config
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)

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
	// Create a temporary directory for isolated testing
	tempHome := t.TempDir()

	// Save and restore original HOME
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp directory to isolate from real config
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)

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
	// Create a temporary directory for isolated testing
	tempHome := t.TempDir()

	// Save and restore original environment
	originalEnv := os.Getenv("SESH_WORKSPACE")
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalEnv != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("SESH_WORKSPACE", originalEnv)
		} else {
			//nolint:errcheck // Test cleanup
			os.Unsetenv("SESH_WORKSPACE")
		}
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp directory to isolate from real config
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)

	t.Run("with environment variable", func(t *testing.T) {
		testPath := "/tmp/test-workspace"
		//nolint:errcheck // Test setup
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
		//nolint:errcheck // Test setup
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
		//nolint:errcheck // Test setup
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
	// Create a temporary directory for isolated testing
	tempHome := t.TempDir()

	// Save and restore original environment
	originalEnv := os.Getenv("SESH_SESSION_BACKEND")
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalEnv != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("SESH_SESSION_BACKEND", originalEnv)
		} else {
			//nolint:errcheck // Test cleanup
			os.Unsetenv("SESH_SESSION_BACKEND")
		}
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp directory to isolate from real config
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)

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
	// Create a temporary directory for isolated testing
	tempHome := t.TempDir()

	// Save and restore original environment
	originalWorkspace := os.Getenv("SESH_WORKSPACE")
	originalBackend := os.Getenv("SESH_SESSION_BACKEND")
	originalHome := os.Getenv("HOME")
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
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp directory to isolate from real config
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)

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

func TestGetFuzzyFinder(t *testing.T) {
	// Create a temporary directory for isolated testing
	tempHome := t.TempDir()

	// Save and restore original environment
	originalEnv := os.Getenv("SESH_FUZZY_FINDER")
	originalHome := os.Getenv("HOME")
	defer func() {
		if originalEnv != "" {
			os.Setenv("SESH_FUZZY_FINDER", originalEnv)
		} else {
			os.Unsetenv("SESH_FUZZY_FINDER")
		}
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Set HOME to temp directory to isolate from real config
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)

	t.Run("with environment variable", func(t *testing.T) {
		os.Setenv("SESH_FUZZY_FINDER", "fzf")

		finder, err := GetFuzzyFinder()
		if err != nil {
			t.Fatalf("GetFuzzyFinder() returned error: %v", err)
		}

		if finder != "fzf" {
			t.Errorf("GetFuzzyFinder() = %q, want %q", finder, "fzf")
		}
	})

	t.Run("default finder", func(t *testing.T) {
		os.Unsetenv("SESH_FUZZY_FINDER")

		finder, err := GetFuzzyFinder()
		if err != nil {
			t.Fatalf("GetFuzzyFinder() returned error: %v", err)
		}

		if finder != "auto" {
			t.Errorf("GetFuzzyFinder() = %q, want %q", finder, "auto")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  configFile
		wantErr bool
	}{
		{
			name: "valid config with all fields",
			config: configFile{
				Version:        "1",
				WorkspaceDir:   "~/projects",
				SessionBackend: "tmux",
				FuzzyFinder:    "fzf",
				StartupCommand: "echo hello",
			},
			wantErr: false,
		},
		{
			name: "valid config with auto values",
			config: configFile{
				Version:        "1",
				WorkspaceDir:   "~/.sesh",
				SessionBackend: "auto",
				FuzzyFinder:    "auto",
			},
			wantErr: false,
		},
		{
			name: "invalid fuzzy finder",
			config: configFile{
				Version:     "1",
				FuzzyFinder: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid session backend",
			config: configFile{
				Version:        "1",
				SessionBackend: "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid empty config",
			config: configFile{
				Version: "1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempHome := t.TempDir()

	// Set HOME to temp directory to isolate from real config
	originalHome := os.Getenv("HOME")
	//nolint:errcheck // Test setup
	os.Setenv("HOME", tempHome)
	defer func() {
		if originalHome != "" {
			//nolint:errcheck // Test cleanup
			os.Setenv("HOME", originalHome)
		}
	}()

	// Create test config
	testConfig := &Config{
		WorkspaceDir:   "~/test-projects",
		SessionBackend: "tmux",
		FuzzyFinder:    "fzf",
		StartupCommand: "echo test",
	}

	// Save config
	err := SaveConfig(testConfig)
	if err != nil {
		t.Fatalf("SaveConfig() returned error: %v", err)
	}

	// Verify file exists
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() returned error: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at: %s", configPath)
	}

	// Load and verify config
	loadedConfig, err := loadConfigFile()
	if err != nil {
		t.Fatalf("loadConfigFile() returned error: %v", err)
	}

	if loadedConfig.Version != CurrentConfigVersion {
		t.Errorf("Version = %q, want %q", loadedConfig.Version, CurrentConfigVersion)
	}

	if loadedConfig.WorkspaceDir != testConfig.WorkspaceDir {
		t.Errorf("WorkspaceDir = %q, want %q", loadedConfig.WorkspaceDir, testConfig.WorkspaceDir)
	}

	if loadedConfig.SessionBackend != testConfig.SessionBackend {
		t.Errorf("SessionBackend = %q, want %q", loadedConfig.SessionBackend, testConfig.SessionBackend)
	}

	if loadedConfig.FuzzyFinder != testConfig.FuzzyFinder {
		t.Errorf("FuzzyFinder = %q, want %q", loadedConfig.FuzzyFinder, testConfig.FuzzyFinder)
	}

	if loadedConfig.StartupCommand != testConfig.StartupCommand {
		t.Errorf("StartupCommand = %q, want %q", loadedConfig.StartupCommand, testConfig.StartupCommand)
	}
}
