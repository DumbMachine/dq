package connection

import (
	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured connections",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		rows := make([]map[string]any, 0, len(cfg.Connections))
		for name, conn := range cfg.Connections {
			row := map[string]any{
				"name": name,
				"type": conn.Type,
			}
			if conn.Host != "" {
				row["host"] = conn.Host
			}
			if conn.Port != 0 {
				row["port"] = conn.Port
			}
			if conn.Database != "" {
				row["database"] = conn.Database
			}
			if conn.Path != "" {
				row["path"] = conn.Path
			}
			rows = append(rows, row)
		}

		return output.Print(format, rows)
	},
}
