package connection

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "connection",
	Short: "Manage database connections",
}

func init() {
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(testCmd)
	Cmd.AddCommand(removeCmd)
}
