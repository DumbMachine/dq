package connection

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/dumbmachine/db-cli/internal/validation"
	"github.com/spf13/cobra"
)

var (
	addType           string
	addHost           string
	addPort           int
	addDatabase       string
	addUser           string
	addPassword       string
	addPath           string
	addSSLMode        string
	addStoreInKeyring bool
	addPasswordStdin  bool
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

		// --password and --password-stdin are mutually exclusive
		if addPassword != "" && addPasswordStdin {
			return fmt.Errorf("--password and --password-stdin are mutually exclusive")
		}

		// Read password from stdin if requested
		if addPasswordStdin {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				addPassword = strings.TrimRight(scanner.Text(), "\r\n")
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("reading password from stdin: %w", err)
			}
			if addPassword == "" {
				return fmt.Errorf("password from stdin is empty")
			}
		}

		// Store in keyring if requested
		passwordForConfig := addPassword
		if addStoreInKeyring {
			if addPassword == "" {
				return fmt.Errorf("--store-in-keyring requires a password (via --password or --password-stdin)")
			}
			if err := config.StoreInKeyring(name, addPassword); err != nil {
				return fmt.Errorf("storing password in keyring: %w", err)
			}
			passwordForConfig = "keyring:" + name
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
			Password: passwordForConfig,
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
		if addPassword != "" && !addStoreInKeyring {
			_, isPlain, _ := config.ResolvePassword(addPassword)
			if isPlain {
				fmt.Fprintln(cmd.ErrOrStderr(), `{"warning":"Password stored as plain text in config file. Consider using --store-in-keyring or env:VAR_NAME format."}`)
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
	addCmd.Flags().StringVar(&addPassword, "password", "", "Database password (use env:VAR for env vars, or --store-in-keyring)")
	addCmd.Flags().StringVar(&addPath, "path", "", "Database file path (SQLite)")
	addCmd.Flags().StringVar(&addSSLMode, "ssl-mode", "", "SSL mode")
	addCmd.Flags().BoolVar(&addStoreInKeyring, "store-in-keyring", false, "Store password in OS keychain instead of config file")
	addCmd.Flags().BoolVar(&addPasswordStdin, "password-stdin", false, "Read password from stdin (avoids shell history and /proc exposure)")
	_ = addCmd.MarkFlagRequired("type")
}
