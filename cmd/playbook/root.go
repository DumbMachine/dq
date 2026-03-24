package playbook

import (
	"github.com/spf13/cobra"
)

// Cmd is the playbook command group.
var Cmd = &cobra.Command{
	Use:   "playbook",
	Short: "Manage playbooks — reusable analytics workflows and org knowledge",
}

func init() {
	Cmd.AddCommand(addCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(showCmd)
	Cmd.AddCommand(removeCmd)
	Cmd.AddCommand(initCmd)
}
