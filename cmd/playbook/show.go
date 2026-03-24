package playbook

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/output"
	pb "github.com/dumbmachine/db-cli/internal/playbook"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a playbook's full content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		playbook, err := pb.Load(name)
		if err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		// For table/human output, print the raw markdown
		if format == "table" {
			fmt.Println(playbook.Content)
			return nil
		}

		return output.Print(format, map[string]any{
			"name":        playbook.Name,
			"description": playbook.Description,
			"tags":        playbook.Tags,
			"connections": playbook.Connections,
			"created":     playbook.Created,
			"updated":     playbook.Updated,
			"content":     playbook.Content,
		})
	},
}
