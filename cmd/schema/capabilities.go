package schema

import (
	"github.com/dumbmachine/db-cli/internal/database"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/spf13/cobra"
)

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities",
	Short: "Show CLI capabilities — runtime self-introspection for agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		caps := map[string]any{
			"tool":    "dq",
			"version": "dev",
			"backends": database.Available(),
			"commands": []map[string]any{
				{"name": "connection add", "description": "Add a new database connection"},
				{"name": "connection list", "description": "List all configured connections"},
				{"name": "connection show", "description": "Show connection details"},
				{"name": "connection test", "description": "Test a database connection"},
				{"name": "connection remove", "description": "Remove a database connection"},
				{"name": "discover", "description": "Full database overview — schemas, tables, columns, FKs"},
				{"name": "postgres", "description": "Execute SQL against PostgreSQL"},
				{"name": "mysql", "description": "Execute SQL against MySQL"},
				{"name": "sqlite", "description": "Execute SQL against SQLite"},
				{"name": "schema tables", "description": "List tables"},
				{"name": "schema columns", "description": "List columns of a table"},
				{"name": "schema indexes", "description": "List indexes of a table"},
				{"name": "schema constraints", "description": "List constraints of a table"},
				{"name": "schema describe", "description": "Full table description with annotations"},
				{"name": "schema capabilities", "description": "Runtime CLI self-introspection"},
				{"name": "annotate set", "description": "Set annotation on table or column"},
				{"name": "annotate get", "description": "Get annotations"},
				{"name": "annotate remove", "description": "Remove annotation"},
			},
			"output_formats": []string{"json", "table", "csv", "ndjson"},
			"global_flags": []map[string]any{
				{"name": "--output", "short": "-o", "description": "Output format"},
				{"name": "--connection", "short": "-c", "description": "Connection name"},
				{"name": "--fields", "description": "Filter output fields"},
				{"name": "--limit", "description": "Limit rows returned"},
				{"name": "--offset", "description": "Skip rows"},
				{"name": "--dry-run", "description": "Preview mutations via transaction rollback"},
				{"name": "--timeout", "description": "Query timeout"},
				{"name": "--yes", "description": "Skip confirmation prompts"},
				{"name": "--quiet", "description": "Suppress non-essential output"},
				{"name": "--verbose", "description": "Verbose output"},
			},
			"exit_codes": map[string]int{
				"success":    0,
				"error":      1,
				"usage":      2,
				"not_found":  3,
				"auth":       4,
				"conflict":   5,
				"timeout":    6,
				"dry_run_ok": 7,
			},
			"features": []string{
				"structured_output",
				"tty_auto_detection",
				"dry_run_via_transaction_rollback",
				"field_masks",
				"pagination",
				"schema_caching",
				"annotations",
				"os_keychain_credential_storage",
			},
		}

		return output.Print(getFormat(cmd), caps)
	},
}
