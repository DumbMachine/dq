package playbook

import (
	"github.com/dumbmachine/db-cli/internal/output"
	pb "github.com/dumbmachine/db-cli/internal/playbook"
	"github.com/spf13/cobra"
)

var listTag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all playbooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		playbooks, err := pb.List()
		if err != nil {
			return err
		}

		// Filter by tag if specified
		if listTag != "" {
			var filtered []pb.Meta
			for _, p := range playbooks {
				for _, t := range p.Tags {
					if t == listTag {
						filtered = append(filtered, p)
						break
					}
				}
			}
			playbooks = filtered
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		if playbooks == nil {
			playbooks = []pb.Meta{}
		}

		return output.Print(format, map[string]any{
			"playbooks": playbooks,
			"count":     len(playbooks),
		})
	},
}

func init() {
	listCmd.Flags().StringVar(&listTag, "tag", "", "Filter playbooks by tag")
}
