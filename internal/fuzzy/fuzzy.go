package fuzzy

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/tty"
	"github.com/rotisserie/eris"
)

// Finder represents the type of fuzzy finder available
type Finder string

const (
	FinderFzf  Finder = "fzf"
	FinderPeco Finder = "peco"
	FinderNone Finder = "none"
)

// SelectBranchFromReader presents a fuzzy finder interface with streaming input from a reader
// The reader should output one item per line
// This starts fzf immediately and pipes data directly for maximum responsiveness
func SelectBranchFromReader(reader io.ReadCloser) (string, error) {
	if !tty.IsInteractive() {
		reader.Close() //nolint:errcheck // Error not critical in early return
		return "", eris.New("interactive selection not available in noninteractive mode")
	}

	finder, err := DetectFuzzyFinder()
	if err != nil {
		return "", eris.Wrap(err, "fuzzy finder required for streaming selection")
	}

	return RunFuzzyFinderFromReader(reader, string(finder))
}

// DetectFuzzyFinder detects which fuzzy finder is available on the system
// Checks config first, then auto-detects in order: fzf, peco
func DetectFuzzyFinder() (Finder, error) {
	// 1. Check config for user preference
	configuredFinder, err := config.GetFuzzyFinder()
	if err == nil && configuredFinder != "" && configuredFinder != "auto" {
		// Verify the configured finder is actually available
		if _, err := exec.LookPath(configuredFinder); err == nil {
			return Finder(configuredFinder), nil
		}
		// If configured finder not found, fall back to auto-detect
	}

	// 2. Auto-detect: Check for fzf
	if _, err := exec.LookPath("fzf"); err == nil {
		return FinderFzf, nil
	}

	// 3. Auto-detect: Check for peco
	if _, err := exec.LookPath("peco"); err == nil {
		return FinderPeco, nil
	}

	return FinderNone, eris.New("no fuzzy finder found (install fzf or peco)")
}

// createFinderCommand creates the appropriate command for the given fuzzy finder
func createFinderCommand(finder string) (*exec.Cmd, error) {
	switch Finder(finder) {
	case FinderFzf:
		return exec.Command("fzf", "--height", "40%", "--reverse", "--border"), nil
	case FinderPeco:
		return exec.Command("peco"), nil
	default:
		return nil, eris.Errorf("unknown fuzzy finder: %s", finder)
	}
}

// RunFuzzyFinderFromReader runs a fuzzy finder with input from a reader
// This pipes data directly from the reader to fzf for maximum performance
// The reader is closed when the function returns
func RunFuzzyFinderFromReader(reader io.ReadCloser, finder string) (string, error) {
	defer reader.Close() //nolint:errcheck

	cmd, err := createFinderCommand(finder)
	if err != nil {
		return "", err
	}

	// Pipe the reader directly to fzf's stdin
	cmd.Stdin = reader
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "", eris.Wrap(err, "failed to get fuzzy finder output")
	}

	selected := strings.TrimSpace(string(out))
	if selected == "" {
		return "", eris.New("no selection made")
	}

	return selected, nil
}

// MultiSelect presents a fuzzy finder with multi-select support (fzf only)
// Returns a list of selected items, or an error
// Users can select multiple items using TAB, and confirm with ENTER
func MultiSelect(items []string, prompt string) ([]string, error) {
	if len(items) == 0 {
		return nil, eris.New("no items available to select")
	}

	// Check if fzf is available (peco doesn't support multi-select)
	if _, err := exec.LookPath("fzf"); err != nil {
		return nil, eris.New("fzf required for multi-select (install fzf)")
	}

	args := []string{
		"--multi",
		"--height", "40%",
		"--reverse",
		"--border",
		"--header", "TAB to select/deselect, ENTER to confirm",
	}
	if prompt != "" {
		args = append(args, "--prompt", prompt)
	}

	cmd := exec.Command("fzf", args...)

	// Create pipe to send items to fuzzy finder
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, eris.Wrap(err, "failed to create stdin pipe")
	}

	// Capture stdout for the selections
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, eris.Wrap(err, "failed to create stdout pipe")
	}

	// Set stderr to show errors
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Write items to stdin in a goroutine
	go func() {
		defer stdin.Close() //nolint:errcheck // Defer close in cleanup
		for _, item := range items {
			fmt.Fprintln(stdin, item) //nolint:errcheck // Write errors will be caught by command execution
		}
	}()

	// Read all selected items
	var selected []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			selected = append(selected, line)
		}
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		// User might have cancelled (Ctrl+C)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil, eris.New("selection cancelled")
		}
		return nil, eris.Wrap(err, "fuzzy finder failed")
	}

	if len(selected) == 0 {
		return nil, eris.New("no selection made")
	}

	return selected, nil
}
