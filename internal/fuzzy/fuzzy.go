package fuzzy

import (
	"bufio"
	"fmt"
	"io"
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

// SelectBranchStreaming presents a fuzzy finder interface with streaming input
// The producer function is called to write items to the provided channel
// This starts fzf immediately for a more responsive interface
func SelectBranchStreaming(producer func(chan<- string) error) (string, error) {
	finder, err := DetectFuzzyFinder()
	if err != nil {
		// For streaming, we can't easily fall back to prompt since we don't have all items
		return "", eris.Wrap(err, "fuzzy finder required for streaming selection")
	}

	return RunFuzzyFinderStreaming(producer, string(finder))
}

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

	// Capture stdout for the selection
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", eris.Wrap(err, "failed to create stdout pipe")
	}

	// Set stderr to show errors
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Write items to stdin in a goroutine
	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	// Read the output
	scanner := bufio.NewScanner(stdout)
	var selected string
	if scanner.Scan() {
		selected = strings.TrimSpace(scanner.Text())
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		// User might have cancelled (Ctrl+C)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", eris.New("selection cancelled")
		}
		return "", eris.Wrap(err, "fuzzy finder failed")
	}

	if selected == "" {
		return "", eris.New("no selection made")
	}

	return selected, nil
}

// RunFuzzyFinderStreaming runs a fuzzy finder with streaming input
// The producer function is called in a goroutine and should send items to the channel
// This allows fzf to start immediately and display items as they're produced
func RunFuzzyFinderStreaming(producer func(chan<- string) error, finder string) (string, error) {
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

	// Capture stdout for the selection
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", eris.Wrap(err, "failed to create stdout pipe")
	}

	// Set stderr to show errors
	cmd.Stderr = os.Stderr

	// Start the command immediately
	if err := cmd.Start(); err != nil {
		return "", eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Create channel for streaming items
	itemsChan := make(chan string, 100) // Buffer to reduce blocking

	// Start producer in goroutine
	producerErr := make(chan error, 1)
	go func() {
		producerErr <- producer(itemsChan)
		close(itemsChan)
	}()

	// Write items to stdin as they arrive
	go func() {
		defer stdin.Close()
		for item := range itemsChan {
			fmt.Fprintln(stdin, item)
		}
	}()

	// Read the output
	scanner := bufio.NewScanner(stdout)
	var selected string
	if scanner.Scan() {
		selected = strings.TrimSpace(scanner.Text())
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		// User might have cancelled (Ctrl+C)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", eris.New("selection cancelled")
		}
		return "", eris.Wrap(err, "fuzzy finder failed")
	}

	// Check if producer had an error
	if err := <-producerErr; err != nil {
		return "", eris.Wrap(err, "failed to produce items")
	}

	if selected == "" {
		return "", eris.New("no selection made")
	}

	return selected, nil
}

// RunFuzzyFinderFromReader runs a fuzzy finder with input from a reader
// This pipes data directly from the reader to fzf for maximum performance
// The reader is closed when the function returns
func RunFuzzyFinderFromReader(reader io.ReadCloser, finder string) (string, error) {
	defer reader.Close()

	var cmd *exec.Cmd

	switch Finder(finder) {
	case FinderFzf:
		cmd = exec.Command("fzf", "--height", "40%", "--reverse", "--border")
	case FinderPeco:
		cmd = exec.Command("peco")
	default:
		return "", eris.Errorf("unknown fuzzy finder: %s", finder)
	}

	// Pipe the reader directly to fzf's stdin
	cmd.Stdin = reader

	// Capture stdout for the selection
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", eris.Wrap(err, "failed to create stdout pipe")
	}

	// Set stderr to show errors
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Read the selected item
	scanner := bufio.NewScanner(stdout)
	var selected string
	if scanner.Scan() {
		selected = strings.TrimSpace(scanner.Text())
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		// User might have cancelled (Ctrl+C)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", eris.New("selection cancelled")
		}
		return "", eris.Wrap(err, "fuzzy finder failed")
	}

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

	// Capture stdout for the selection
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", eris.Wrap(err, "failed to create stdout pipe")
	}

	// Set stderr to show errors
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", eris.Wrap(err, "failed to start fuzzy finder")
	}

	// Write items to stdin in a goroutine
	go func() {
		defer stdin.Close()
		for _, item := range items {
			fmt.Fprintln(stdin, item)
		}
	}()

	// Read the output
	scanner := bufio.NewScanner(stdout)
	var selected string
	if scanner.Scan() {
		selected = strings.TrimSpace(scanner.Text())
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", eris.New("selection cancelled")
		}
		return "", eris.Wrap(err, "fuzzy finder failed")
	}

	if selected == "" {
		return "", eris.New("no selection made")
	}

	return selected, nil
}
