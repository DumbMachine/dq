package playbook

import (
	"fmt"
	"io"
	"os"

	"github.com/dumbmachine/db-cli/internal/output"
	pb "github.com/dumbmachine/db-cli/internal/playbook"
	"github.com/spf13/cobra"
)

var addFile string

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a playbook from a markdown file or stdin",
	Long: `Add a playbook to dq's local store.

From a file:
  dq playbook add my-analysis --file playbook.md

From stdin:
  cat playbook.md | dq playbook add my-analysis

The file must have YAML frontmatter with at least a 'name' field.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var data []byte
		var err error

		if addFile != "" {
			data, err = os.ReadFile(addFile)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}
		} else {
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) != 0 {
				return fmt.Errorf("provide --file <path> or pipe markdown to stdin")
			}
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
		}

		playbook, err := pb.Parse(string(data))
		if err != nil {
			return err
		}

		// Override name with CLI argument
		playbook.Name = name

		if err := pb.Save(playbook); err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		return output.Print(format, map[string]any{
			"status":      "added",
			"name":        name,
			"description": playbook.Description,
			"tags":        playbook.Tags,
		})
	},
}

func init() {
	addCmd.Flags().StringVar(&addFile, "file", "", "Path to playbook markdown file")
}
