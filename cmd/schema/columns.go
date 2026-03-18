package schema

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var columnsTable string

var columnsCmd = &cobra.Command{
	Use:   "columns",
	Short: "List columns of a table",
	RunE: func(cmd *cobra.Command, args []string) error {
		if columnsTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		drv, db, _, err := getDriverAndDB(cmd)
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		schemaName, _ := cmd.Flags().GetString("schema")
		columns, err := drv.ListColumns(db, schemaName, columnsTable)
		if err != nil {
			return err
		}

		return output.Print(getFormat(cmd), columns)
	},
}

func init() {
	columnsCmd.Flags().StringVar(&columnsTable, "table", "", "Table name")
	columnsCmd.Flags().String("schema", "", "Schema name")
	_ = columnsCmd.MarkFlagRequired("table")
}
