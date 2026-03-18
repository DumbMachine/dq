package annotate

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/annotations"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	removeTable  string
	removeColumn string
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an annotation from a table or column",
	RunE: func(cmd *cobra.Command, args []string) error {
		connName, _ := cmd.Flags().GetString("connection")
		if connName == "" {
			return fmt.Errorf("--connection (-c) flag is required")
		}
		if removeTable == "" {
			return fmt.Errorf("--table flag is required")
		}

		var err error
		if removeColumn != "" {
			err = annotations.RemoveColumnNote(connName, removeTable, removeColumn)
		} else {
			err = annotations.RemoveTableNote(connName, removeTable)
		}
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		result := map[string]any{
			"status":     "removed",
			"connection": connName,
			"table":      removeTable,
		}
		if removeColumn != "" {
			result["column"] = removeColumn
		}
		return output.Print(format, result)
	},
}

func init() {
	removeCmd.Flags().StringVar(&removeTable, "table", "", "Table name")
	removeCmd.Flags().StringVar(&removeColumn, "column", "", "Column name (optional)")
	_ = removeCmd.MarkFlagRequired("table")
}
