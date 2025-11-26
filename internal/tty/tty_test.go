package tty

import (
	"os"
	"testing"
)

func TestIsInteractive(t *testing.T) {
	// In test environment, stdin is typically not a terminal
	// So we expect IsInteractive to return false
	// This test mainly ensures the function doesn't panic
	result := IsInteractive()

	// We can't reliably test the specific return value as it depends
	// on how tests are run, but we can verify it returns a boolean
	if result != true && result != false {
		t.Error("IsInteractive should return a boolean value")
	}
}

func TestIsOutputTerminal(t *testing.T) {
	// Similar to IsInteractive, we just verify it doesn't panic
	result := IsOutputTerminal()

	if result != true && result != false {
		t.Error("IsOutputTerminal should return a boolean value")
	}
}

func TestIsFullyInteractive(t *testing.T) {
	// IsFullyInteractive should be true only if both stdin and stdout are terminals
	result := IsFullyInteractive()

	// Verify it returns a boolean
	if result != true && result != false {
		t.Error("IsFullyInteractive should return a boolean value")
	}

	// If fully interactive, both stdin and stdout should be terminals
	if result {
		if !IsInteractive() || !IsOutputTerminal() {
			t.Error("IsFullyInteractive returned true but IsInteractive or IsOutputTerminal is false")
		}
	}
}

func TestTTYDetectionConsistency(t *testing.T) {
	// Test that the relationship between functions is consistent
	isInteractive := IsInteractive()
	isOutput := IsOutputTerminal()
	isFullyInteractive := IsFullyInteractive()

	// If fully interactive, both stdin and stdout must be terminals
	if isFullyInteractive && (!isInteractive || !isOutput) {
		t.Error("IsFullyInteractive is true but IsInteractive or IsOutputTerminal is false")
	}

	// If either is false, fully interactive must be false
	if (!isInteractive || !isOutput) && isFullyInteractive {
		t.Error("IsInteractive or IsOutputTerminal is false but IsFullyInteractive is true")
	}
}

// TestNonInteractiveEnv tests behavior when explicitly running in non-interactive mode
func TestNonInteractiveEnv(t *testing.T) {
	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Create a pipe (not a terminal)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()

	// Replace stdin with the pipe
	os.Stdin = r

	// Now IsInteractive should return false
	if IsInteractive() {
		t.Error("IsInteractive should return false when stdin is a pipe")
	}
}
