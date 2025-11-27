package fuzzy

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/benoctopus/sesh/internal/config"
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
