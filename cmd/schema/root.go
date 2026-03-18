package schema

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "schema",
	Short: "Inspect database schema — tables, columns, indexes, constraints",
}

func init() {
	Cmd.AddCommand(tablesCmd)
	Cmd.AddCommand(columnsCmd)
	Cmd.AddCommand(indexesCmd)
	Cmd.AddCommand(constraintsCmd)
	Cmd.AddCommand(describeCmd)
	Cmd.AddCommand(capabilitiesCmd)
}
