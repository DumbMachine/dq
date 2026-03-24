package playbook

import (
	"github.com/dumbmachine/db-cli/internal/output"
	pb "github.com/dumbmachine/db-cli/internal/playbook"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a playbook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := pb.Remove(name); err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		return output.Print(format, map[string]any{
			"status": "removed",
			"name":   name,
		})
	},
}
