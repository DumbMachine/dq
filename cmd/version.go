package cmd

import (
	"runtime"

	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		info := map[string]any{
			"version":    Version,
			"commit":     Commit,
			"build_date": BuildDate,
			"go_version": runtime.Version(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
		}
		return output.Print(GetOutputFormat(), info)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
