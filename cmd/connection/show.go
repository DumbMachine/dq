package connection

import (
	"strings"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var showReveal bool

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show connection details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		conn, err := cfg.GetConnection(name)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		result := map[string]any{
			"name": name,
			"type": conn.Type,
		}
		if conn.Host != "" {
			result["host"] = conn.Host
		}
		if conn.Port != 0 {
			result["port"] = conn.Port
		}
		if conn.Database != "" {
			result["database"] = conn.Database
		}
		if conn.User != "" {
			result["user"] = conn.User
		}
		if conn.Path != "" {
			result["path"] = conn.Path
		}
		if conn.SSLMode != "" {
			result["ssl_mode"] = conn.SSLMode
		}
		if conn.Password != "" {
			if showReveal {
				result["password"] = conn.Password
			} else {
				result["password"] = maskPassword(conn.Password)
			}
		}

		return output.Print(format, result)
	},
}

func maskPassword(pw string) string {
	if strings.HasPrefix(pw, "env:") || strings.HasPrefix(pw, "keyring:") {
		return pw
	}
	return "****"
}

func init() {
	showCmd.Flags().BoolVar(&showReveal, "reveal", false, "Show password in plain text")
}
