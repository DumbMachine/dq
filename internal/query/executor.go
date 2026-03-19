package query

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dumbmachine/db-cli/pkg/types"
	"gorm.io/gorm"
)

type ExecOptions struct {
	Timeout time.Duration
	Limit   int
	Offset  int
}

type ExecResult struct {
	Columns  []types.ColumnMeta
	Rows     []map[string]any
	Duration time.Duration
}

func Execute(db *gorm.DB, sql string, opts ExecOptions) (*ExecResult, error) {
	start := time.Now()

	ctx := context.Background()
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Apply pagination via wrapping (simple approach)
	execSQL := sql
	if opts.Limit > 0 && opts.Offset > 0 {
		execSQL = fmt.Sprintf("SELECT * FROM (%s) AS _dq_sub LIMIT %d OFFSET %d", sql, opts.Limit, opts.Offset)
	} else if opts.Limit > 0 {
		execSQL = fmt.Sprintf("SELECT * FROM (%s) AS _dq_sub LIMIT %d", sql, opts.Limit)
	} else if opts.Offset > 0 {
		execSQL = fmt.Sprintf("SELECT * FROM (%s) AS _dq_sub OFFSET %d", sql, opts.Offset)
	}

	return executeNormal(ctx, db, execSQL, start)
}

func executeNormal(ctx context.Context, db *gorm.DB, sql string, start time.Time) (*ExecResult, error) {
	rows, err := db.WithContext(ctx).Raw(sql).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanResult(rows, start)
}

func scanResult(rows *sql.Rows, start time.Time) (*ExecResult, error) {
	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	colTypes, _ := rows.ColumnTypes()
	columns := make([]types.ColumnMeta, len(colNames))
	for i, name := range colNames {
		typ := "unknown"
		if i < len(colTypes) {
			typ = colTypes[i].DatabaseTypeName()
		}
		columns[i] = types.ColumnMeta{Name: name, Type: typ}
	}

	var resultRows []map[string]any
	for rows.Next() {
		values := make([]any, len(colNames))
		valuePtrs := make([]any, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(colNames))
		for i, name := range colNames {
			val := values[i]
			// Convert byte slices to strings for JSON friendliness
			if b, ok := val.([]byte); ok {
				row[name] = string(b)
			} else {
				row[name] = val
			}
		}
		resultRows = append(resultRows, row)
	}

	return &ExecResult{
		Columns:  columns,
		Rows:     resultRows,
		Duration: time.Since(start),
	}, nil
}
