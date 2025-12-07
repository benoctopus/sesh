package cmd

import (
	"testing"

	"github.com/benoctopus/sesh/internal/config"
)

func TestSwitchToAllWorktrees_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupFlags    func()
		args          []string
		expectedError string
	}{
		{
			name: "cannot use --pr with --all-worktrees",
			setupFlags: func() {
				switchPR = true
				switchAllWorktrees = true
				switchProjectName = ""
			},
			args:          []string{},
			expectedError: "cannot use --pr with --all-worktrees",
		},
		{
			name: "cannot use --project with --all-worktrees",
			setupFlags: func() {
				switchPR = false
				switchAllWorktrees = true
				switchProjectName = "myproject"
			},
			args:          []string{},
			expectedError: "cannot use --project with --all-worktrees",
		},
		{
			name: "cannot specify branch name with --all-worktrees",
			setupFlags: func() {
				switchPR = false
				switchAllWorktrees = true
				switchProjectName = ""
			},
			args:          []string{"main"},
			expectedError: "cannot specify branch name with --all-worktrees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup flags for this test case
			tt.setupFlags()
			defer func() {
				// Reset flags after test
				switchPR = false
				switchAllWorktrees = false
				switchProjectName = ""
			}()

			// Create a minimal config
			cfg := &config.Config{
				WorkspaceDir: t.TempDir(),
			}

			// Call switchToAllWorktrees
			err := switchToAllWorktrees(nil, cfg, tt.args)

			// Verify we got the expected error
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.expectedError)
			}

			// Check if error message contains expected text
			if !containsError(err, tt.expectedError) {
				t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestSwitchToAllWorktrees_InteractiveRequired(t *testing.T) {
	// Reset flags
	switchPR = false
	switchAllWorktrees = true
	switchProjectName = ""
	defer func() {
		switchAllWorktrees = false
	}()

	// Create a minimal config
	cfg := &config.Config{
		WorkspaceDir: t.TempDir(),
	}

	// Call switchToAllWorktrees (will fail because we're not in interactive mode)
	err := switchToAllWorktrees(nil, cfg, []string{})

	// Verify we got an error about interactive mode
	if err == nil {
		t.Fatal("expected error about interactive mode, got nil")
	}

	expectedMsg := "--all-worktrees requires interactive mode"
	if !containsError(err, expectedMsg) {
		t.Errorf("expected error containing %q, got %q", expectedMsg, err.Error())
	}
}

func TestGetStartupCommand(t *testing.T) {
	tests := []struct {
		name             string
		flagValue        string
		globalConfig     string
		projectConfig    string
		expectedResult   string
		setupProjectCfg  bool
		projectCfgExists bool
	}{
		{
			name:           "command-line flag takes precedence",
			flagValue:      "direnv allow",
			globalConfig:   "echo global",
			expectedResult: "direnv allow",
		},
		{
			name:           "global config when no flag",
			flagValue:      "",
			globalConfig:   "echo global",
			expectedResult: "echo global",
		},
		{
			name:           "empty when no config",
			flagValue:      "",
			globalConfig:   "",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup command-line flag
			switchStartupCommand = tt.flagValue
			defer func() {
				switchStartupCommand = ""
			}()

			// Create config with global startup command
			cfg := &config.Config{
				StartupCommand: tt.globalConfig,
			}

			// Create a temp worktree path
			worktreePath := t.TempDir()

			// Get startup command
			result := getStartupCommand(cfg, worktreePath)

			// Verify result
			if result != tt.expectedResult {
				t.Errorf("getStartupCommand() = %q, want %q", result, tt.expectedResult)
			}
		})
	}
}

// Helper function to check if an error contains a specific message
func containsError(err error, msg string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Simple substring check
	return len(errStr) >= len(msg) && contains(errStr, msg)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

// Helper to check substring existence
func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
