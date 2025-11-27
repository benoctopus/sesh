package display

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// Printer provides formatted output with styling capabilities.
// It follows the standard fmt.Print* interface while adding semantic
// methods for common message types (Success, Error, Warning, Info).
// All output methods ignore errors internally for simplicity.
type Printer interface {
	// Standard fmt-like methods
	Print(a ...interface{})
	Println(a ...interface{})
	Printf(format string, a ...interface{})

	// Styled output methods with icons
	Success(msg string)
	Error(msg string)
	Warning(msg string)
	Info(msg string)

	// Styled formatting methods with icons
	Successf(format string, a ...interface{})
	Errorf(format string, a ...interface{})
	Warningf(format string, a ...interface{})
	Infof(format string, a ...interface{})

	// Text styling methods
	Bold(text string) string
	Faint(text string) string
	SuccessText(text string) string
	ErrorText(text string) string
	WarningText(text string) string
	InfoText(text string) string
}

// writer implements the Printer interface
type writer struct {
	out io.Writer
	// Color formatters
	successColor func(a ...interface{}) string
	errorColor   func(a ...interface{}) string
	warningColor func(a ...interface{}) string
	infoColor    func(a ...interface{}) string
	boldStyle    func(a ...interface{}) string
	faintStyle   func(a ...interface{}) string
}

// New creates a new Printer that writes to the given io.Writer.
func New(w io.Writer) Printer {
	return &writer{
		out:          w,
		successColor: color.New(color.FgGreen).SprintFunc(),
		errorColor:   color.New(color.FgRed).SprintFunc(),
		warningColor: color.New(color.FgYellow).SprintFunc(),
		infoColor:    color.New(color.FgCyan).SprintFunc(),
		boldStyle:    color.New(color.Bold).SprintFunc(),
		faintStyle:   color.New(color.Faint).SprintFunc(),
	}
}

// NewStderr creates a new Printer that writes to stderr.
// This is the recommended default for user-facing messages.
func NewStderr() Printer {
	return New(os.Stderr)
}

// NewStdout creates a new Printer that writes to stdout.
// Use this only for results that may be piped to other commands.
func NewStdout() Printer {
	return New(os.Stdout)
}

// Print formats using the default formats for its operands and writes to the output.
func (w *writer) Print(a ...interface{}) {
	_, _ = fmt.Fprint(w.out, a...)
}

// Println formats using the default formats for its operands and writes to the output.
func (w *writer) Println(a ...interface{}) {
	_, _ = fmt.Fprintln(w.out, a...)
}

// Printf formats according to a format specifier and writes to the output.
func (w *writer) Printf(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(w.out, format, a...)
}

// Success prints a success message with a green checkmark icon.
func (w *writer) Success(msg string) {
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.successColor("✓"), msg)
}

// Error prints an error message with a red X icon.
func (w *writer) Error(msg string) {
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.errorColor("✗"), msg)
}

// Warning prints a warning message with a yellow warning icon.
func (w *writer) Warning(msg string) {
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.warningColor("⚠"), msg)
}

// Info prints an info message with a cyan info icon.
func (w *writer) Info(msg string) {
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.infoColor("ℹ"), msg)
}

// Successf prints a formatted success message with a green checkmark icon.
func (w *writer) Successf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.successColor("✓"), msg)
}

// Errorf prints a formatted error message with a red X icon.
func (w *writer) Errorf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.errorColor("✗"), msg)
}

// Warningf prints a formatted warning message with a yellow warning icon.
func (w *writer) Warningf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.warningColor("⚠"), msg)
}

// Infof prints a formatted info message with a cyan info icon.
func (w *writer) Infof(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	_, _ = fmt.Fprintf(w.out, "%s %s\n", w.infoColor("ℹ"), msg)
}

// Bold returns the text formatted in bold.
func (w *writer) Bold(text string) string {
	return w.boldStyle(text)
}

// Faint returns the text formatted as faint/dim.
func (w *writer) Faint(text string) string {
	return w.faintStyle(text)
}

// SuccessText returns the text formatted in green (success color).
func (w *writer) SuccessText(text string) string {
	return w.successColor(text)
}

// ErrorText returns the text formatted in red (error color).
func (w *writer) ErrorText(text string) string {
	return w.errorColor(text)
}

// WarningText returns the text formatted in yellow (warning color).
func (w *writer) WarningText(text string) string {
	return w.warningColor(text)
}

// InfoText returns the text formatted in cyan (info color).
func (w *writer) InfoText(text string) string {
	return w.infoColor(text)
}
