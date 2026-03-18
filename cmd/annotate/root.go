package annotate

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "annotate",
	Short: "Manage annotations — agent knowledge base persisted between conversations",
}

func init() {
	Cmd.AddCommand(setCmd)
	Cmd.AddCommand(getCmd)
	Cmd.AddCommand(removeCmd)
}
