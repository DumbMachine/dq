package annotate

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/annotations"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	setTable  string
	setColumn string
	setNote   string
)

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set an annotation on a table or column",
	RunE: func(cmd *cobra.Command, args []string) error {
		connName, _ := cmd.Flags().GetString("connection")
		if connName == "" {
			return fmt.Errorf("--connection (-c) flag is required")
		}
		if setTable == "" {
			return fmt.Errorf("--table flag is required")
		}
		if setNote == "" {
			return fmt.Errorf("--note flag is required")
		}

		var err error
		if setColumn != "" {
			err = annotations.SetColumnNote(connName, setTable, setColumn, setNote)
		} else {
			err = annotations.SetTableNote(connName, setTable, setNote)
		}
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		result := map[string]any{
			"status":     "set",
			"connection": connName,
			"table":      setTable,
			"note":       setNote,
		}
		if setColumn != "" {
			result["column"] = setColumn
		}
		return output.Print(format, result)
	},
}

func init() {
	setCmd.Flags().StringVar(&setTable, "table", "", "Table name")
	setCmd.Flags().StringVar(&setColumn, "column", "", "Column name (optional)")
	setCmd.Flags().StringVar(&setNote, "note", "", "Annotation text")
	_ = setCmd.MarkFlagRequired("table")
	_ = setCmd.MarkFlagRequired("note")
}
