package tty

import (
	"os"

	"golang.org/x/term"
)

// IsInteractive returns true if the current environment is interactive.
// It checks if stdin is a terminal (TTY).
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// IsOutputTerminal returns true if stdout is a terminal.
// This is useful for determining if output formatting should be used.
func IsOutputTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// IsFullyInteractive returns true if both stdin and stdout are terminals.
// This indicates a fully interactive environment.
func IsFullyInteractive() bool {
	return IsInteractive() && IsOutputTerminal()
}
