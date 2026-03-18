package query

import (
	"fmt"
	"time"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"github.com/dumbmachine/db-cli/internal/output"
	"github.com/dumbmachine/db-cli/internal/query"
	"github.com/dumbmachine/db-cli/internal/validation"
	"github.com/dumbmachine/db-cli/pkg/types"
	"github.com/spf13/cobra"

	_ "github.com/dumbmachine/db-cli/internal/database/mysql"
	_ "github.com/dumbmachine/db-cli/internal/database/postgres"
	_ "github.com/dumbmachine/db-cli/internal/database/sqlite"
)

func runQuery(cmd *cobra.Command, args []string, expectedType string) error {
	if len(args) == 0 {
		return fmt.Errorf("SQL query is required")
	}
	sql := args[0]

	if err := validation.ValidateSQL(sql); err != nil {
		return err
	}

	connName, _ := cmd.Flags().GetString("connection")
	if connName == "" {
		return fmt.Errorf("--connection (-c) flag is required")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn, err := cfg.GetConnection(connName)
	if err != nil {
		return err
	}

	if expectedType != "" && conn.Type != expectedType {
		return fmt.Errorf("connection %q is type %s, expected %s", connName, conn.Type, expectedType)
	}

	drv, err := database.Get(conn.Type)
	if err != nil {
		return err
	}

	db, err := drv.Connect(conn)
	if err != nil {
		return err
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	timeoutStr, _ := cmd.Flags().GetString("timeout")
	limitVal, _ := cmd.Flags().GetInt("limit")
	offsetVal, _ := cmd.Flags().GetInt("offset")
	fieldsVal, _ := cmd.Flags().GetString("fields")
	explain, _ := cmd.Flags().GetBool("explain")

	timeout, _ := time.ParseDuration(timeoutStr)

	if explain {
		sql = "EXPLAIN " + sql
	}

	result, err := query.Execute(db, sql, query.ExecOptions{
		DryRun:  dryRun,
		Timeout: timeout,
		Limit:   limitVal,
		Offset:  offsetVal,
	})
	if err != nil {
		return err
	}

	rows := result.Rows
	if fieldsVal != "" {
		rows = output.FilterFields(rows, fieldsVal)
	}

	format, _ := cmd.Flags().GetString("output")
	if format == "" {
		format = output.DefaultFormat()
	}

	qr := types.QueryResult{
		Meta: types.ResultMeta{
			Connection: connName,
			Database:   conn.Database,
			RowCount:   len(rows),
			DurationMs: result.Duration.Milliseconds(),
			DryRun:     dryRun,
			Limit:      limitVal,
			Offset:     offsetVal,
		},
		Columns: result.Columns,
		Rows:    rows,
	}

	if dryRun {
		qr.Meta.AffectedRows = result.AffectedRows
	}

	return output.Print(format, qr)
}

func addQueryFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("explain", false, "Prepend EXPLAIN to the query")
}
