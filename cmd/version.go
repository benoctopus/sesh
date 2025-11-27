package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version information set via ldflags during build
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
	builtBy   = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long: `Print the version, commit hash, and build date for sesh.

This information is injected at build time via ldflags.`,
	Run: runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("sesh %s\n", version)
	fmt.Printf("  commit: %s\n", commit)
	fmt.Printf("  built: %s\n", buildDate)
	fmt.Printf("  by: %s\n", builtBy)
	fmt.Printf("  go: %s\n", runtime.Version())
	fmt.Printf("  platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
