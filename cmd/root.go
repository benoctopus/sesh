package cmd

import (
	"fmt"
	"os"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sesh",
	Short: "Git workspace and tmux session manager",
	Long: `sesh is a tool for managing git workspaces and tmux sessions.

It helps developers quickly switch between different project contexts by
managing both git repositories and tmux sessions in a unified way.

Examples:
  sesh list                    # List all sessions
  sesh attach <name>          # Attach to a session
  sesh create <name> <path>   # Create new session`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", eris.ToString(err, true))
		os.Exit(1)
	}
}

func init() {
	// Global flags can be defined here
}
