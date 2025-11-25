package ui

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	// Success prints green text for successful operations
	Success = color.New(color.FgGreen).SprintFunc()
	// Error prints red text for errors
	Error = color.New(color.FgRed).SprintFunc()
	// Warning prints yellow text for warnings
	Warning = color.New(color.FgYellow).SprintFunc()
	// Info prints cyan text for informational messages
	Info = color.New(color.FgCyan).SprintFunc()
	// Bold prints bold text for emphasis
	Bold = color.New(color.Bold).SprintFunc()
	// Faint prints dim text for less important information
	Faint = color.New(color.Faint).SprintFunc()
)

// SuccessMsg prints a success message with checkmark
func SuccessMsg(msg string) {
	fmt.Printf("%s %s\n", Success("✓"), msg)
}

// ErrorMsg prints an error message with X mark
func ErrorMsg(msg string) {
	fmt.Printf("%s %s\n", Error("✗"), msg)
}

// WarningMsg prints a warning message with exclamation mark
func WarningMsg(msg string) {
	fmt.Printf("%s %s\n", Warning("⚠"), msg)
}

// InfoMsg prints an info message with info icon
func InfoMsg(msg string) {
	fmt.Printf("%s %s\n", Info("ℹ"), msg)
}
