package schema

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	_ "github.com/dumbmachine/db-cli/internal/database/mysql"
	_ "github.com/dumbmachine/db-cli/internal/database/postgres"
	_ "github.com/dumbmachine/db-cli/internal/database/sqlite"
)

func getDriverAndDB(cmd *cobra.Command) (database.Driver, *gorm.DB, string, error) {
	connName, _ := cmd.Flags().GetString("connection")
	if connName == "" {
		return nil, nil, "", fmt.Errorf("--connection (-c) flag is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, nil, "", err
	}

	conn, err := cfg.GetConnection(connName)
	if err != nil {
		return nil, nil, "", err
	}

	drv, err := database.Get(conn.Type)
	if err != nil {
		return nil, nil, "", err
	}

	db, err := drv.Connect(conn)
	if err != nil {
		return nil, nil, "", err
	}

	return drv, db, connName, nil
}

func getFormat(cmd *cobra.Command) string {
	format, _ := cmd.Flags().GetString("output")
	if format == "" {
		format = output.DefaultFormat()
	}
	return format
}
