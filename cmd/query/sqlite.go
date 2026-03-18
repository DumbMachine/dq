package query

import (
	"github.com/spf13/cobra"
)

var sqliteCmd = &cobra.Command{
	Use:   "sqlite <sql>",
	Short: "Execute a SQL query against a SQLite database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuery(cmd, args, "sqlite")
	},
}

func init() {
	addQueryFlags(sqliteCmd)
}
