package schema

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/annotations"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var describeTable string

var describeCmd = &cobra.Command{
	Use:   "describe",
	Short: "Describe a table — columns, indexes, constraints, and annotations",
	RunE: func(cmd *cobra.Command, args []string) error {
		if describeTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		drv, db, connName, err := getDriverAndDB(cmd)
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		schemaName, _ := cmd.Flags().GetString("schema")

		columns, _ := drv.ListColumns(db, schemaName, describeTable)
		indexes, _ := drv.ListIndexes(db, schemaName, describeTable)
		constraints, _ := drv.ListConstraints(db, schemaName, describeTable)
		ann := annotations.GetTableAnnotation(connName, describeTable)

		result := map[string]any{
			"connection":  connName,
			"table":       describeTable,
			"columns":     columns,
			"indexes":     indexes,
			"constraints": constraints,
		}
		if ann != nil {
			result["annotations"] = ann
		}

		return output.Print(getFormat(cmd), result)
	},
}

func init() {
	describeCmd.Flags().StringVar(&describeTable, "table", "", "Table name")
	describeCmd.Flags().String("schema", "", "Schema name")
	_ = describeCmd.MarkFlagRequired("table")
}
