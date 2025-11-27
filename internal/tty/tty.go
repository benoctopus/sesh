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
