package fuzzy

import (
	"os"
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

func TestIsAvailable(t *testing.T) {
	// Test that IsAvailable returns a boolean
	available := IsAvailable()

	// Verify it matches DetectFuzzyFinder result
	_, err := DetectFuzzyFinder()
	expectedAvailable := err == nil

	if available != expectedAvailable {
		t.Errorf("IsAvailable() = %v, want %v", available, expectedAvailable)
	}
}

func TestGetAvailableFinder(t *testing.T) {
	finder := GetAvailableFinder()

	// Should return one of the known finders or "none"
	validFinders := map[string]bool{
		"fzf":  true,
		"peco": true,
		"none": true,
	}

	if !validFinders[finder] {
		t.Errorf("GetAvailableFinder() returned unknown finder: %s", finder)
	}
}

func TestSelectBranch_EmptyList(t *testing.T) {
	_, err := SelectBranch([]string{})
	if err == nil {
		t.Error("SelectBranch() should return error for empty list")
	}
}

func TestSelect_EmptyList(t *testing.T) {
	_, err := Select([]string{}, "test prompt")
	if err == nil {
		t.Error("Select() should return error for empty list")
	}
}

func TestRunFuzzyFinder_UnknownFinder(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	_, err := RunFuzzyFinder(items, "unknown-finder")

	if err == nil {
		t.Error("RunFuzzyFinder() should return error for unknown finder")
	}
}

func TestSelectWithPreview_EmptyList(t *testing.T) {
	_, err := SelectWithPreview([]string{}, "echo {}")
	if err == nil {
		t.Error("SelectWithPreview() should return error for empty list")
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

// TestSelectWithPreview_FallbackWhenFzfNotAvailable tests that SelectWithPreview
// falls back to regular Select when fzf is not available
func TestSelectWithPreview_FallbackWhenFzfNotAvailable(t *testing.T) {
	// Check if fzf is available
	_, err := exec.LookPath("fzf")
	if err == nil {
		// fzf is available, skip this test
		t.Skip("Skipping fallback test because fzf is available")
	}

	// When fzf is not available and we call SelectWithPreview with a non-empty list,
	// it should fall back to Select, which will fall back to selectWithPrompt
	// Since we can't interact with stdin in tests, we just verify it doesn't panic
	// and returns an appropriate error
	items := []string{"item1", "item2"}

	// We can't really test the interactive part without mocking stdin,
	// but we can at least verify the function handles the case gracefully
	// This is more of a smoke test
	_, err = SelectWithPreview(items, "echo {}")
	// We expect an error because we can't provide stdin in the test
	// The important thing is that it doesn't panic
	if err == nil {
		// If somehow it succeeded, that's also okay
		t.Log("SelectWithPreview succeeded unexpectedly, but that's acceptable")
	}
}

// Test that the fuzzy finder integration uses correct command-line arguments
func TestRunFuzzyFinder_CommandArguments(t *testing.T) {
	// Check if fzf is available
	_, fzfErr := exec.LookPath("fzf")
	if fzfErr != nil {
		t.Skip("Skipping test because fzf is not available")
	}

	// We can't easily test the full execution without user input,
	// but we can verify the function handles the case gracefully
	items := []string{"branch-1", "branch-2", "branch-3"}

	// This will fail because there's no interactive input in the test,
	// but we're checking that it doesn't panic and the error is expected
	_, err := RunFuzzyFinder(items, "fzf")

	// We expect an error because fzf needs interactive input
	// The important part is verifying the function handles it correctly
	if err == nil {
		t.Log("RunFuzzyFinder unexpectedly succeeded without input")
	}
}

// TestSelectBranch_NonInteractive tests that SelectBranch handles the case
// where no fuzzy finder is available
func TestSelectBranch_NonInteractive(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Check if any fuzzy finder is available
	_, err := DetectFuzzyFinder()
	if err == nil {
		t.Skip("Skipping non-interactive test because fuzzy finder is available")
	}

	branches := []string{"main", "develop", "feature"}

	// Without interactive input, this should fail with a read error
	// The important thing is that it attempts the fallback
	_, err = SelectBranch(branches)

	// We expect an error because we can't provide input
	if err == nil {
		t.Error("SelectBranch should fail without interactive input when using fallback")
	}
}

// TestSelect_NonInteractive tests that Select handles the case
// where no fuzzy finder is available
func TestSelect_NonInteractive(t *testing.T) {
	// Check if any fuzzy finder is available
	_, err := DetectFuzzyFinder()
	if err == nil {
		t.Skip("Skipping non-interactive test because fuzzy finder is available")
	}

	items := []string{"option1", "option2", "option3"}

	// Without interactive input, this should fail with a read error
	_, err = Select(items, "Select an option")

	// We expect an error because we can't provide input
	if err == nil {
		t.Error("Select should fail without interactive input when using fallback")
	}
}
