package fuzzy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rotisserie/eris"
)

// Finder represents the type of fuzzy finder available
type Finder string

const (
	FinderFzf  Finder = "fzf"
	FinderPeco Finder = "peco"
	FinderNone Finder = "none"
)

// SelectBranch presents a fuzzy finder interface to select a branch from a list
// Returns the selected branch name or an error
func SelectBranch(branches []string) (string, error) {
	if len(branches) == 0 {
		return "", eris.New("no branches available to select")
	}

	finder, err := DetectFuzzyFinder()
	if err != nil {
		// Fall back to simple selection
		return selectWithPrompt(branches)
	}

	return RunFuzzyFinder(branches, string(finder))
}

// Select presents a fuzzy finder interface to select an item from a list
// This is a generic version that can be used for any list of items
func Select(items []string, prompt string) (string, error) {
	if len(items) == 0 {
		return "", eris.New("no items available to select")
	}

	finder, err := DetectFuzzyFinder()
	if err != nil {
		// Fall back to simple selection
		return selectWithPrompt(items)
	}

	return RunFuzzyFinder(items, string(finder))
}

// DetectFuzzyFinder detects which fuzzy finder is available on the system
// Checks in order: fzf, peco
func DetectFuzzyFinder() (Finder, error) {
	// Check for fzf
	if _, err := exec.LookPath("fzf"); err == nil {
		return FinderFzf, nil
	}

	// Check for peco
	if _, err := exec.LookPath("peco"); err == nil {
		return FinderPeco, nil
	}

	return FinderNone, eris.New("no fuzzy finder found (install fzf or peco)")
}

// RunFuzzyFinder runs the specified fuzzy finder with the given items
func RunFuzzyFinder(items []string, finder string) (string, error) {
	var cmd *exec.Cmd

	switch Finder(finder) {
	case FinderFzf:
		cmd = exec.Command("fzf", "--height", "40%", "--reverse", "--border")
	case FinderPeco:
		cmd = exec.Command("peco")
	default:
		return "", eris.Errorf("unknown fuzzy finder: %s", finder)
	}

	// Create pipe to send items to fuzzy finder
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", eris.Wrap(err, "failed to create stdin pipe")
	}

	// Set output to capture selection
	cmd.Stdout = nil // Will capture below
	cmd.Stderr = os.Stderr

	// Connect to terminal for interactive input
	cmd.Stdin = os.Stdin

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Write items to stdin
	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		// User might have cancelled (Ctrl+C)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", eris.New("selection cancelled")
		}
		return "", eris.Wrap(err, "fuzzy finder failed")
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", eris.New("no selection made")
	}

	return selected, nil
}

// selectWithPrompt is a fallback selection method when no fuzzy finder is available
// It displays a numbered list and prompts the user to select by number
func selectWithPrompt(items []string) (string, error) {
	// Display numbered list
	fmt.Println("Select an option:")
	for i, item := range items {
		fmt.Printf("%3d. %s\n", i+1, item)
	}

	// Prompt for selection
	fmt.Print("\nEnter number (1-", len(items), "): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", eris.Wrap(err, "failed to read input")
	}

	// Parse selection
	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil {
		return "", eris.Wrap(err, "invalid selection")
	}

	if selection < 1 || selection > len(items) {
		return "", eris.Errorf("selection out of range (1-%d)", len(items))
	}

	return items[selection-1], nil
}

// IsAvailable checks if a fuzzy finder is available on the system
func IsAvailable() bool {
	_, err := DetectFuzzyFinder()
	return err == nil
}

// GetAvailableFinder returns the name of the available fuzzy finder
func GetAvailableFinder() string {
	finder, err := DetectFuzzyFinder()
	if err != nil {
		return "none"
	}
	return string(finder)
}

// SelectWithPreview presents a fuzzy finder with preview support (fzf only)
// The preview command will be executed for each item, with {} replaced by the item
func SelectWithPreview(items []string, previewCmd string) (string, error) {
	if len(items) == 0 {
		return "", eris.New("no items available to select")
	}

	// Check if fzf is available (peco doesn't support preview)
	if _, err := exec.LookPath("fzf"); err != nil {
		// Fall back to regular selection
		return Select(items, "")
	}

	cmd := exec.Command("fzf",
		"--height", "40%",
		"--reverse",
		"--border",
		"--preview", previewCmd,
		"--preview-window", "right:60%",
	)

	// Create pipe to send items to fuzzy finder
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", eris.Wrap(err, "failed to create stdin pipe")
	}

	// Connect to terminal
	cmd.Stdout = nil // Will capture below
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Write items to stdin
	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", eris.New("selection cancelled")
		}
		return "", eris.Wrap(err, "fuzzy finder failed")
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", eris.New("no selection made")
	}

	return selected, nil
}
