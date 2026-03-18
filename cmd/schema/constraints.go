package schema

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var constraintsTable string

var constraintsCmd = &cobra.Command{
	Use:   "constraints",
	Short: "List constraints of a table",
	RunE: func(cmd *cobra.Command, args []string) error {
		if constraintsTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		drv, db, _, err := getDriverAndDB(cmd)
		if err != nil {
			return err
		}
		sqlDB, _ := db.DB()
		defer sqlDB.Close()

		schemaName, _ := cmd.Flags().GetString("schema")
		constraints, err := drv.ListConstraints(db, schemaName, constraintsTable)
		if err != nil {
			return err
		}

		return output.Print(getFormat(cmd), constraints)
	},
}

func init() {
	constraintsCmd.Flags().StringVar(&constraintsTable, "table", "", "Table name")
	constraintsCmd.Flags().String("schema", "", "Schema name")
	_ = constraintsCmd.MarkFlagRequired("table")
}
