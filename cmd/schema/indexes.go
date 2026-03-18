package schema

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var indexesTable string

var indexesCmd = &cobra.Command{
	Use:   "indexes",
	Short: "List indexes of a table",
	RunE: func(cmd *cobra.Command, args []string) error {
		if indexesTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		drv, db, _, err := getDriverAndDB(cmd)
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		schemaName, _ := cmd.Flags().GetString("schema")
		indexes, err := drv.ListIndexes(db, schemaName, indexesTable)
		if err != nil {
			return err
		}

		return output.Print(getFormat(cmd), indexes)
	},
}

func init() {
	indexesCmd.Flags().StringVar(&indexesTable, "table", "", "Table name")
	indexesCmd.Flags().String("schema", "", "Schema name")
	_ = indexesCmd.MarkFlagRequired("table")
}
