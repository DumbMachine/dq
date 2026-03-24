package playbook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dumbmachine/db-cli/internal/output"
	pb "github.com/dumbmachine/db-cli/internal/playbook"
	"github.com/spf13/cobra"
)

var initDir string

var initCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Generate a playbook template file",
	Long: `Creates a starter playbook markdown file that you can edit and then add with 'dq playbook add'.

  dq playbook init my-analysis
  # edit my-analysis.md
  dq playbook add my-analysis --file my-analysis.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content := pb.Template(name)

		dir := initDir
		if dir == "" {
			dir = "."
		}

		filename := name + ".md"
		path := filepath.Join(dir, filename)

		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file %s already exists", path)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing template: %w", err)
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		return output.Print(format, map[string]any{
			"status": "created",
			"file":   path,
			"name":   name,
		})
	},
}

func init() {
	initCmd.Flags().StringVar(&initDir, "dir", ".", "Directory to create the template file in")
}
