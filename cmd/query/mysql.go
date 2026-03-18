package query

import (
	"github.com/spf13/cobra"
)

var mysqlCmd = &cobra.Command{
	Use:   "mysql <sql>",
	Short: "Execute a SQL query against a MySQL database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuery(cmd, args, "mysql")
	},
}

func init() {
	addQueryFlags(mysqlCmd)
}
