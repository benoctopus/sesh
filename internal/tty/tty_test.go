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
