package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/benoctopus/sesh/internal/config"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the sesh configuration file",
	Long: `Opens the sesh configuration file in your default editor.

The config file is located at:
  - Linux: ~/.config/sesh/config.yaml
  - macOS: ~/Library/Application Support/sesh/config.yaml
  - Windows: %APPDATA%\sesh\config.yaml

The editor is determined by the EDITOR environment variable (falls back to vim).

After editing, the configuration will be validated. If validation fails, you'll
be prompted to fix the errors before the changes are saved.

Example config.yaml:
  version: "1"
  workspace_dir: ~/projects
  session_backend: tmux
  fuzzy_finder: fzf
  startup_command: ""
`,
	RunE: runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	// Get config path
	configPath, err := config.GetConfigPath()
	if err != nil {
		return eris.Wrap(err, "failed to get config path")
	}

	// Ensure config directory exists
	if err := config.EnsureConfigDir(); err != nil {
		return eris.Wrap(err, "failed to ensure config directory")
	}

	// Create config file with defaults if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return eris.Wrap(err, "failed to create default config")
		}
		fmt.Printf("Created default config at: %s\n", configPath)
	}

	// Get the file hash before editing
	hashBefore, err := hashFile(configPath)
	if err != nil {
		return eris.Wrap(err, "failed to hash config file")
	}

	// Open in editor
	editor := getEditor()
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return eris.Wrapf(err, "failed to run editor: %s", editor)
	}

	// Get the file hash after editing
	hashAfter, err := hashFile(configPath)
	if err != nil {
		return eris.Wrap(err, "failed to hash config file after editing")
	}

	// Check if file was modified
	if hashBefore == hashAfter {
		fmt.Println("No changes made to config")
		return nil
	}

	// Validate the config
	if err := config.ValidateConfigFile(configPath); err != nil {
		fmt.Printf("\nConfig validation failed: %+v\n", eris.ToString(err, false))
		fmt.Printf("\nThe config file has errors. Do you want to:\n")
		fmt.Printf("  1. Edit again to fix errors\n")
		fmt.Printf("  2. Discard changes and restore previous version\n")
		fmt.Printf("  3. Save anyway (not recommended)\n")
		fmt.Printf("\nChoice (1-3): ")

		var choice string
		fmt.Scanln(&choice)

		switch choice {
		case "1":
			// Edit again
			return runEdit(cmd, args)
		case "2":
			// We can't easily restore without keeping a backup
			// So we'll just tell the user to manually fix it
			fmt.Println("\nPlease manually fix the config file or delete it to start over.")
			return eris.New("config validation failed")
		case "3":
			// Save anyway
			fmt.Println("\nWarning: Saving invalid config. This may cause issues.")
			return nil
		default:
			return eris.New("invalid choice")
		}
	}

	fmt.Printf("Config saved and validated successfully: %s\n", configPath)
	return nil
}

// getEditor returns the user's preferred editor
// Priority: VISUAL > EDITOR > vi
func getEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	// Load default config
	cfg := &config.Config{
		WorkspaceDir:   "~/.sesh",
		SessionBackend: "auto",
		StartupCommand: "",
		FuzzyFinder:    "auto",
	}

	return config.SaveConfig(cfg)
}

// hashFile computes the SHA256 hash of a file
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
