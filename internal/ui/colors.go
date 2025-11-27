// Package ui provides terminal UI utilities.
//
// Deprecated: This package is deprecated. Use github.com/benoctopus/sesh/internal/display instead.
// The display package provides a cleaner interface with configurable output targets.
package ui

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	// Success prints green text for successful operations.
	// Deprecated: Use display.Printer.SuccessText() instead.
	Success = color.New(color.FgGreen).SprintFunc()
	// Error prints red text for errors.
	// Deprecated: Use display.Printer.ErrorText() instead.
	Error = color.New(color.FgRed).SprintFunc()
	// Warning prints yellow text for warnings.
	// Deprecated: Use display.Printer.WarningText() instead.
	Warning = color.New(color.FgYellow).SprintFunc()
	// Info prints cyan text for informational messages.
	// Deprecated: Use display.Printer.InfoText() instead.
	Info = color.New(color.FgCyan).SprintFunc()
	// Bold prints bold text for emphasis.
	// Deprecated: Use display.Printer.Bold() instead.
	Bold = color.New(color.Bold).SprintFunc()
	// Faint prints dim text for less important information.
	// Deprecated: Use display.Printer.Faint() instead.
	Faint = color.New(color.Faint).SprintFunc()
)

// SuccessMsg prints a success message with checkmark.
// Deprecated: Use display.Printer.Success() instead.
func SuccessMsg(msg string) {
	fmt.Printf("%s %s\n", Success("✓"), msg)
}

// ErrorMsg prints an error message with X mark.
// Deprecated: Use display.Printer.Error() instead.
func ErrorMsg(msg string) {
	fmt.Printf("%s %s\n", Error("✗"), msg)
}

// WarningMsg prints a warning message with exclamation mark.
// Deprecated: Use display.Printer.Warning() instead.
func WarningMsg(msg string) {
	fmt.Printf("%s %s\n", Warning("⚠"), msg)
}

// InfoMsg prints an info message with info icon.
// Deprecated: Use display.Printer.Info() instead.
func InfoMsg(msg string) {
	fmt.Printf("%s %s\n", Info("ℹ"), msg)
}
