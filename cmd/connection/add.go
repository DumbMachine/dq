package connection

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/dumbmachine/db-cli/internal/validation"
	"github.com/spf13/cobra"
)

var (
	addType     string
	addHost     string
	addPort     int
	addDatabase string
	addUser     string
	addPassword string
	addPath     string
	addSSLMode  string
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new database connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validation.ValidateName(name, "connection name"); err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		conn := &config.ConnectionConfig{
			Type:     addType,
			Host:     addHost,
			Port:     addPort,
			Database: addDatabase,
			User:     addUser,
			Password: addPassword,
			Path:     addPath,
			SSLMode:  addSSLMode,
		}

		if err := cfg.AddConnection(name, conn); err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("output")
		if format == "" {
			format = output.DefaultFormat()
		}

		// Check if password is plaintext and warn
		if addPassword != "" {
			_, isPlain, _ := config.ResolvePassword(addPassword)
			if isPlain {
				fmt.Fprintln(cmd.ErrOrStderr(), `{"warning":"Password stored as plain text. Consider using env:VAR_NAME format."}`)
			}
		}

		return output.Print(format, map[string]any{
			"status":     "created",
			"connection": name,
			"type":       addType,
		})
	},
}

func init() {
	addCmd.Flags().StringVar(&addType, "type", "", "Database type: postgres, mysql, sqlite")
	addCmd.Flags().StringVar(&addHost, "host", "", "Database host")
	addCmd.Flags().IntVar(&addPort, "port", 0, "Database port")
	addCmd.Flags().StringVar(&addDatabase, "database", "", "Database name")
	addCmd.Flags().StringVar(&addUser, "user", "", "Database user")
	addCmd.Flags().StringVar(&addPassword, "password", "", "Database password (use env:VAR for env vars)")
	addCmd.Flags().StringVar(&addPath, "path", "", "Database file path (SQLite)")
	addCmd.Flags().StringVar(&addSSLMode, "ssl-mode", "", "SSL mode")
	_ = addCmd.MarkFlagRequired("type")
}
