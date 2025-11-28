package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/benoctopus/sesh/internal/display"
	"github.com/benoctopus/sesh/internal/session"
	"github.com/benoctopus/sesh/internal/state"
	"github.com/benoctopus/sesh/internal/workspace"
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

var tmuxMenuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Generate tmux display-menu for session switching",
	Long: `Generate a tmux display-menu command for native session switching.

This command outputs a tmux display-menu command that can be used in keybindings
for a keyboard-driven session switcher without requiring fzf.

Example usage in tmux.conf:
  bind-key m run-shell "sesh tmux menu"

The menu provides:
  - Single-key selection (numbered 0-9, a-z)
  - Native tmux look and feel
  - No external dependencies (fzf not required)

Note: Best for projects with ~20 or fewer sessions. For larger projects,
use the fzf-based popup switcher (prefix + f) instead.`,
	RunE: runTmuxMenu,
}

func init() {
	rootCmd.AddCommand(tmuxCmd)
	tmuxCmd.AddCommand(tmuxKeybindingsCmd)
	tmuxCmd.AddCommand(tmuxInstallCmd)
	tmuxCmd.AddCommand(tmuxMenuCmd)
}

var bin, _ = os.Executable()

const tmuxKeybindingsContent = `# BEGIN sesh tmux integration
# Fuzzy session switcher with preview (prefix + f)
bind-key f display-popup -E -w 80% -h 60% \
  "{{ .Bin }} switch"

# Fuzzy pull request switcher with preview (prefix + F)
bind-key F display-popup -E -w 80% -h 60% \
  "{{ .Bin }} switch --pr"

# Quick switch to last/previous session (prefix + L)
bind-key L run-shell "{{ .Bin }} last"
# END sesh tmux integration
`

const (
	seshMarkerBegin = "# BEGIN sesh tmux integration"
	seshMarkerEnd   = "# END sesh tmux integration"
)

// renderKeybindings executes the keybindings template with the binary path
func renderKeybindings() (string, error) {
	tmpl, err := template.New("keybindings").Parse(tmuxKeybindingsContent)
	if err != nil {
		return "", eris.Wrap(err, "failed to parse keybindings template")
	}

	var buf bytes.Buffer
	data := struct {
		Bin string
	}{
		Bin: bin,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", eris.Wrap(err, "failed to execute keybindings template")
	}

	return buf.String(), nil
}

// removeSeshBlock removes the existing sesh keybindings block from the content
func removeSeshBlock(content string) string {
	// Find the start of the sesh block
	startIdx := strings.Index(content, seshMarkerBegin)
	if startIdx == -1 {
		return content
	}

	// Find the end of the sesh block
	endIdx := strings.Index(content, seshMarkerEnd)
	if endIdx == -1 {
		return content
	}

	// Find the newline after the end marker
	endIdx = strings.Index(content[endIdx:], "\n")
	if endIdx == -1 {
		// End marker is at the end of the file
		endIdx = len(content)
	} else {
		endIdx = endIdx + strings.Index(content, seshMarkerEnd) + 1
	}

	// Remove any blank lines before the block
	beforeBlock := content[:startIdx]
	beforeBlock = strings.TrimRight(beforeBlock, "\n")
	if len(beforeBlock) > 0 {
		beforeBlock += "\n"
	}

	// Combine the content before and after the block
	return beforeBlock + content[endIdx:]
}

func runTmuxKeybindings(cmd *cobra.Command, args []string) error {
	disp := display.NewStderr()

	disp.Printf("\n%s\n", disp.Bold("Recommended tmux keybindings for sesh:"))
	disp.Println()

	// Render keybindings with actual binary path
	keybindings, err := renderKeybindings()
	if err != nil {
		return err
	}

	// Print to stdout for easy copying
	fmt.Print(keybindings)

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

	disp.Printf(
		"\n%s %s\n",
		disp.InfoText("→"),
		disp.Faint(fmt.Sprintf("Using tmux config: %s", tmuxConfPath)),
	)

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
	replaceExisting := false
	if !createNew {
		contentBytes, err := os.ReadFile(tmuxConfPath)
		if err != nil {
			return eris.Wrapf(err, "failed to read tmux config: %s", tmuxConfPath)
		}
		existingContent = string(contentBytes)

		// Check if sesh keybindings are already installed
		if strings.Contains(existingContent, seshMarkerBegin) {
			replaceExisting = true
			// Remove the existing sesh block
			existingContent = removeSeshBlock(existingContent)
		}
	}

	// Render keybindings with actual binary path
	keybindings, err := renderKeybindings()
	if err != nil {
		return err
	}

	var finalContent string
	if replaceExisting {
		// When replacing, write the entire updated file content
		finalContent = existingContent
		// Add blank line before keybindings if content doesn't end with double newline
		if len(finalContent) > 0 && !strings.HasSuffix(finalContent, "\n\n") {
			if strings.HasSuffix(finalContent, "\n") {
				finalContent += "\n"
			} else {
				finalContent += "\n\n"
			}
		}
		finalContent += keybindings

		// Write entire file
		if err := os.WriteFile(tmuxConfPath, []byte(finalContent), 0o644); err != nil {
			return eris.Wrap(err, "failed to write keybindings to tmux config")
		}
	} else {
		// When appending, use append mode
		file, err := os.OpenFile(tmuxConfPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return eris.Wrapf(err, "failed to open tmux config for writing: %s", tmuxConfPath)
		}
		defer file.Close() //nolint:errcheck

		// Add a blank line before our section if file has existing content
		contentToWrite := keybindings
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
	}

	// Display success message
	if replaceExisting {
		disp.Success("Successfully updated sesh tmux keybindings!")
	} else {
		disp.Success("Successfully installed sesh tmux keybindings!")
	}
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

func runTmuxMenu(cmd *cobra.Command, args []string) error {
	// Import required packages (will be added to import block)
	// We need to get the list of sessions similar to the list command

	cfg, err := config.LoadConfig()
	if err != nil {
		return eris.Wrap(err, "failed to load configuration")
	}

	// Get session list using internal packages
	sessions, err := getSessionsForMenu(cfg)
	if err != nil {
		return eris.Wrap(err, "failed to get sessions")
	}

	if len(sessions) == 0 {
		// If no sessions, show a message menu
		fmt.Println("tmux display-menu -T 'No Sessions' 'No sessions found' '' ''")
		return nil
	}

	// Generate tmux display-menu command
	menuCmd := generateTmuxMenuCommand(sessions)
	fmt.Println(menuCmd)

	return nil
}

type menuSession struct {
	Name    string
	Project string
	Branch  string
}

func getSessionsForMenu(cfg *config.Config) ([]menuSession, error) {
	// Import required internal packages
	sessionMgr, err := session.NewSessionManager(cfg.SessionBackend)
	if err != nil {
		return nil, eris.Wrap(err, "failed to initialize session manager")
	}

	// Get all running sessions
	runningSessions, err := state.DiscoverSessions(sessionMgr)
	if err != nil {
		return nil, eris.Wrap(err, "failed to discover sessions")
	}

	// Discover all projects and worktrees
	projects, err := state.DiscoverProjects(cfg.WorkspaceDir)
	if err != nil {
		return nil, eris.Wrap(err, "failed to discover projects")
	}

	var sessions []menuSession

	// Build session list
	for _, proj := range projects {
		worktrees, err := state.DiscoverWorktrees(proj)
		if err != nil {
			continue
		}

		for _, wt := range worktrees {
			// Generate expected session name
			sessionName := workspace.GenerateSessionName(proj.Name, wt.Branch)

			// Only include running sessions in the menu
			isRunning := false
			for _, runningSess := range runningSessions {
				if runningSess == sessionName {
					isRunning = true
					break
				}
			}

			if isRunning {
				sessions = append(sessions, menuSession{
					Name:    sessionName,
					Project: proj.Name,
					Branch:  wt.Branch,
				})
			}
		}
	}

	return sessions, nil
}

func generateTmuxMenuCommand(sessions []menuSession) string {
	// Keys for menu items: 0-9, then a-z
	keys := "0123456789abcdefghijklmnopqrstuvwxyz"

	var menuItems []string
	menuItems = append(menuItems, "tmux display-menu -T 'Switch Session'")

	for i, sess := range sessions {
		if i >= len(keys) {
			// Limit to number of available keys
			break
		}

		key := string(keys[i])
		label := fmt.Sprintf("%s (%s)", sess.Branch, sess.Project)

		// Escape single quotes in the label and command
		label = strings.ReplaceAll(label, "'", "'\\''")
		sessionName := strings.ReplaceAll(sess.Name, "'", "'\\''")

		// Format: "label" "key" "command"
		menuItem := fmt.Sprintf("  '%s' '%s' 'run-shell \"sesh switch %s\"'",
			label, key, sessionName)
		menuItems = append(menuItems, menuItem)
	}

	// Add separator and cancel option
	menuItems = append(menuItems, "  '' '' ''") // separator
	menuItems = append(menuItems, "  'Cancel' 'q' ''")

	return strings.Join(menuItems, " \\\n")
}
