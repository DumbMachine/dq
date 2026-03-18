package query

import (
	"github.com/spf13/cobra"
)

var postgresCmd = &cobra.Command{
	Use:   "postgres <sql>",
	Short: "Execute a SQL query against a PostgreSQL database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuery(cmd, args, "postgres")
	},
}

func init() {
	addQueryFlags(postgresCmd)
}
