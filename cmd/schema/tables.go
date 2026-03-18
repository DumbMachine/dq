package schema

import (
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var tablesCmd = &cobra.Command{
	Use:   "tables",
	Short: "List tables in the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		drv, db, _, err := getDriverAndDB(cmd)
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		schemaName, _ := cmd.Flags().GetString("schema")
		tables, err := drv.ListTables(db, schemaName)
		if err != nil {
			return err
		}

		return output.Print(getFormat(cmd), tables)
	},
}

func init() {
	tablesCmd.Flags().String("schema", "", "Schema name (default: public/main)")
}
