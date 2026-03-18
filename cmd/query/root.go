package query

import (
	"github.com/spf13/cobra"
)

func RegisterCommands(root *cobra.Command) {
	root.AddCommand(postgresCmd)
	root.AddCommand(mysqlCmd)
	root.AddCommand(sqliteCmd)
}
