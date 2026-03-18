package annotate

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/annotations"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	getTable  string
	getColumn string
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get annotations for a connection, table, or column",
	RunE: func(cmd *cobra.Command, args []string) error {
		connName, _ := cmd.Flags().GetString("connection")
		if connName == "" {
			return fmt.Errorf("--connection (-c) flag is required")
		}

		af, err := annotations.Load(connName)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		if getTable == "" {
			// Return all annotations for this connection
			return output.Print(format, af)
		}

		t, ok := af.Tables[getTable]
		if !ok {
			return output.Print(format, map[string]any{
				"connection": connName,
				"table":      getTable,
				"annotations": nil,
			})
		}

		if getColumn != "" {
			col, ok := t.Columns[getColumn]
			if !ok {
				return output.Print(format, map[string]any{
					"connection": connName,
					"table":      getTable,
					"column":     getColumn,
					"note":       nil,
				})
			}
			return output.Print(format, map[string]any{
				"connection": connName,
				"table":      getTable,
				"column":     getColumn,
				"note":       col.Note,
			})
		}

		return output.Print(format, map[string]any{
			"connection": connName,
			"table":      getTable,
			"note":       t.Note,
			"columns":    t.Columns,
		})
	},
}

func init() {
	getCmd.Flags().StringVar(&getTable, "table", "", "Table name")
	getCmd.Flags().StringVar(&getColumn, "column", "", "Column name")
}
