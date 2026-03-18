package connection

import (
	"strings"

	"github.com/dumbmachine/db-cli/internal/cache"
	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var removeYes bool

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a database connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Clean up keyring entry if the connection uses one
		conn, err := cfg.GetConnection(name)
		if err != nil {
			return err
		}
		if strings.HasPrefix(conn.Password, "keyring:") {
			keyringName := strings.TrimPrefix(conn.Password, "keyring:")
			_ = config.DeleteFromKeyring(keyringName)
		}

		if err := cfg.RemoveConnection(name); err != nil {
			return err
		}

		// Also clean up cache
		_ = cache.Invalidate(name)

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		return output.Print(format, map[string]any{
			"status":     "removed",
			"connection": name,
		})
	},
}

func init() {
	removeCmd.Flags().BoolVar(&removeYes, "yes", false, "Skip confirmation")
}
