package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benoctopus/sesh/internal/display"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

var tmuxCmd = &cobra.Command{
	Use:   "tmux",
	Short: "Tmux integration commands",
	Long: `Commands for integrating sesh with tmux.

These commands help set up and manage sesh integration with tmux,
including installing recommended keybindings.`,
}

var tmuxKeybindingsCmd = &cobra.Command{
	Use:   "keybindings",
	Short: "Show recommended tmux keybindings",
	Long: `Display recommended tmux keybindings for sesh integration.

These keybindings can be manually copied to your ~/.tmux.conf or
automatically installed using 'sesh tmux install'.`,
	RunE: runTmuxKeybindings,
}

var tmuxInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install tmux keybindings to tmux.conf",
	Long: `Automatically install recommended sesh keybindings to your tmux configuration.

This command will:
  1. Detect your tmux.conf location (~/.tmux.conf or ~/.config/tmux/tmux.conf)
  2. Check if sesh keybindings are already installed
  3. Append the keybindings if not present
  4. Prompt to reload tmux configuration

The installed keybindings include:
  - prefix + f: Fuzzy session switcher with preview
  - prefix + L: Switch to last/previous session

Examples:
  sesh tmux install        # Install keybindings
  sesh tmux keybindings    # Show keybindings without installing`,
	RunE: runTmuxInstall,
}

func init() {
	rootCmd.AddCommand(tmuxCmd)
	tmuxCmd.AddCommand(tmuxKeybindingsCmd)
	tmuxCmd.AddCommand(tmuxInstallCmd)
}

const tmuxKeybindingsContent = `# BEGIN sesh tmux integration
# Fuzzy session switcher with preview (prefix + f)
bind-key f display-popup -E -w 80% -h 60% \
  "sesh list --plain | fzf --reverse --preview 'sesh info {}' | xargs -r sesh switch"

# Quick switch to last/previous session (prefix + L)
bind-key L run-shell "sesh last"
# END sesh tmux integration
`

const (
	seshMarkerBegin = "# BEGIN sesh tmux integration"
	seshMarkerEnd   = "# END sesh tmux integration"
)

func runTmuxKeybindings(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()

	disp.Printf("\n%s\n", disp.Bold("Recommended tmux keybindings for sesh:"))
	disp.Println()

	// Print to stdout for easy copying
	fmt.Print(tmuxKeybindingsContent)

	disp.Println()
	disp.Info("To install these keybindings automatically, run:")
	disp.Printf("  %s\n\n", disp.Bold("sesh tmux install"))

	return nil
}

func runTmuxInstall(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()

	// Find tmux.conf location
	tmuxConfPath, err := findTmuxConf()
	if err != nil {
		return err
	}

	disp.Printf("\n%s %s\n", disp.InfoText("→"), disp.Faint(fmt.Sprintf("Using tmux config: %s", tmuxConfPath)))

	// Check if file exists, create if not
	createNew := false
	if _, err := os.Stat(tmuxConfPath); os.IsNotExist(err) {
		createNew = true
		// Ensure directory exists
		dir := filepath.Dir(tmuxConfPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return eris.Wrapf(err, "failed to create config directory: %s", dir)
		}
	}

	// Read existing content if file exists
	var existingContent string
	if !createNew {
		contentBytes, err := os.ReadFile(tmuxConfPath)
		if err != nil {
			return eris.Wrapf(err, "failed to read tmux config: %s", tmuxConfPath)
		}
		existingContent = string(contentBytes)

		// Check if sesh keybindings are already installed
		if strings.Contains(existingContent, seshMarkerBegin) {
			disp.Success("Sesh keybindings are already installed!")
			disp.Println()
			disp.Info("To reload your tmux configuration, run:")
			disp.Printf("  %s\n\n", disp.Bold("tmux source-file ~/.tmux.conf"))
			return nil
		}
	}

	// Append keybindings
	file, err := os.OpenFile(tmuxConfPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return eris.Wrapf(err, "failed to open tmux config for writing: %s", tmuxConfPath)
	}
	defer file.Close() //nolint:errcheck

	// Add a blank line before our section if file has existing content
	contentToWrite := tmuxKeybindingsContent
	if !createNew && len(existingContent) > 0 && !strings.HasSuffix(existingContent, "\n\n") {
		if strings.HasSuffix(existingContent, "\n") {
			contentToWrite = "\n" + contentToWrite
		} else {
			contentToWrite = "\n\n" + contentToWrite
		}
	}

	if _, err := file.WriteString(contentToWrite); err != nil {
		return eris.Wrap(err, "failed to write keybindings to tmux config")
	}

	disp.Success("Successfully installed sesh tmux keybindings!")
	disp.Println()
	disp.Printf("%s\n", disp.Bold("Installed keybindings:"))
	disp.Printf("  %s %s\n", disp.InfoText("prefix + f"), "Fuzzy session switcher with preview")
	disp.Printf("  %s %s\n", disp.InfoText("prefix + L"), "Switch to last/previous session")
	disp.Println()
	disp.Info("To apply the changes, reload your tmux configuration:")
	disp.Printf("  %s\n\n", disp.Bold("tmux source-file "+tmuxConfPath))

	return nil
}

// findTmuxConf locates the tmux configuration file
func findTmuxConf() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", eris.Wrap(err, "failed to get home directory")
	}

	// Check standard locations in order of preference
	candidates := []string{
		filepath.Join(homeDir, ".tmux.conf"),
		filepath.Join(homeDir, ".config", "tmux", "tmux.conf"),
	}

	// If TMUX_CONF env var is set, use that
	if envPath := os.Getenv("TMUX_CONF"); envPath != "" {
		// Expand tilde if present
		if strings.HasPrefix(envPath, "~/") {
			envPath = filepath.Join(homeDir, envPath[2:])
		}
		candidates = append([]string{envPath}, candidates...)
	}

	// Return first existing file, or default to ~/.tmux.conf
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Default to ~/.tmux.conf (will be created)
	return candidates[0], nil
}
