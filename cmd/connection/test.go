package connection

import (
	"time"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"

	// Register drivers
	_ "github.com/dumbmachine/db-cli/internal/database/mysql"
	_ "github.com/dumbmachine/db-cli/internal/database/postgres"
	_ "github.com/dumbmachine/db-cli/internal/database/sqlite"
)

var testCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test a database connection",
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

		drv, err := database.Get(conn.Type)
		if err != nil {
			return err
		}

		start := time.Now()
		db, err := drv.Connect(conn)

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		if err != nil {
			return output.Print(format, map[string]any{
				"connection": name,
				"status":     "failed",
				"error":      err.Error(),
				"duration_ms": time.Since(start).Milliseconds(),
			})
		}

		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		defer sqlDB.Close()

		if err := sqlDB.Ping(); err != nil {
			return output.Print(format, map[string]any{
				"connection": name,
				"status":     "failed",
				"error":      err.Error(),
				"duration_ms": time.Since(start).Milliseconds(),
			})
		}

		return output.Print(format, map[string]any{
			"connection":  name,
			"status":      "ok",
			"duration_ms": time.Since(start).Milliseconds(),
		})
	},
}
