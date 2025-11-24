package cmd

import (
	"testing"
)

func TestGetGitStatusSummary(t *testing.T) {
	// Note: This test requires mocking or a real git repository
	// For now, we'll test the status parsing logic separately

	t.Run("clean status", func(t *testing.T) {
		// This would need a temp git repo to test properly
		// Skipping for now as it requires integration testing
		t.Skip("Requires integration test setup")
	})
}
