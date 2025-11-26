package fuzzy

import (
	"os/exec"
	"testing"
)

func TestDetectFuzzyFinder(t *testing.T) {
	// This test checks if any fuzzy finder is available
	// The actual result will depend on the test environment
	finder, err := DetectFuzzyFinder()

	// If fzf or peco is installed, we should get no error
	// If neither is installed, we should get an error
	if err == nil {
		// A fuzzy finder was found
		if finder != FinderFzf && finder != FinderPeco {
			t.Errorf("DetectFuzzyFinder() returned unexpected finder: %s", finder)
		}
	} else {
		// No fuzzy finder found
		if finder != FinderNone {
			t.Errorf("DetectFuzzyFinder() should return FinderNone when error occurs, got: %s", finder)
		}
	}
}

// TestFuzzyFinderInPath tests if the fuzzy finder detection matches exec.LookPath
func TestFuzzyFinderInPath(t *testing.T) {
	// Check fzf
	_, fzfErr := exec.LookPath("fzf")

	// Check peco
	_, pecoErr := exec.LookPath("peco")

	// Detect fuzzy finder
	finder, detectErr := DetectFuzzyFinder()

	if fzfErr == nil {
		// fzf is available, should be detected
		if detectErr != nil {
			t.Error("DetectFuzzyFinder() returned error when fzf is available")
		}
		if finder != FinderFzf {
			t.Errorf("DetectFuzzyFinder() = %s, want %s when fzf is available", finder, FinderFzf)
		}
	} else if pecoErr == nil {
		// peco is available (and fzf is not), should be detected
		if detectErr != nil {
			t.Error("DetectFuzzyFinder() returned error when peco is available")
		}
		if finder != FinderPeco {
			t.Errorf("DetectFuzzyFinder() = %s, want %s when peco is available", finder, FinderPeco)
		}
	} else {
		// Neither is available, should return error
		if detectErr == nil {
			t.Error("DetectFuzzyFinder() should return error when no fuzzy finder is available")
		}
		if finder != FinderNone {
			t.Errorf("DetectFuzzyFinder() = %s, want %s when no fuzzy finder is available", finder, FinderNone)
		}
	}
}

// TestFinderConstants verifies the finder constant values
func TestFinderConstants(t *testing.T) {
	tests := []struct {
		finder Finder
		want   string
	}{
		{FinderFzf, "fzf"},
		{FinderPeco, "peco"},
		{FinderNone, "none"},
	}

	for _, tt := range tests {
		if string(tt.finder) != tt.want {
			t.Errorf("Finder constant %s = %q, want %q", tt.want, string(tt.finder), tt.want)
		}
	}
}

// TestCreateFinderCommand tests the helper function that creates finder commands
func TestCreateFinderCommand(t *testing.T) {
	tests := []struct {
		name      string
		finder    string
		wantCmd   string
		wantError bool
	}{
		{
			name:      "fzf command",
			finder:    "fzf",
			wantCmd:   "fzf",
			wantError: false,
		},
		{
			name:      "peco command",
			finder:    "peco",
			wantCmd:   "peco",
			wantError: false,
		},
		{
			name:      "unknown finder",
			finder:    "unknown",
			wantCmd:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := createFinderCommand(tt.finder)

			if tt.wantError {
				if err == nil {
					t.Errorf("createFinderCommand(%q) expected error, got nil", tt.finder)
				}
			} else {
				if err != nil {
					t.Errorf("createFinderCommand(%q) unexpected error: %v", tt.finder, err)
				}
				if cmd == nil {
					t.Errorf("createFinderCommand(%q) returned nil command", tt.finder)
				} else if cmd.Path != "" && cmd.Args[0] != tt.wantCmd {
					t.Errorf("createFinderCommand(%q) command = %q, want %q", tt.finder, cmd.Args[0], tt.wantCmd)
				}
			}
		})
	}
}
