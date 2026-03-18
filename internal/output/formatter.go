package output

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Formatter interface {
	Format(w io.Writer, data any) error
}

func GetFormatter(format string) Formatter {
	switch strings.ToLower(format) {
	case "json":
		return &JSONFormatter{}
	case "table":
		return &TableFormatter{}
	case "csv":
		return &CSVFormatter{}
	case "ndjson":
		return &NDJSONFormatter{}
	default:
		return &JSONFormatter{}
	}
}

// Print formats data and writes to stdout using the given format.
func Print(format string, data any) error {
	f := GetFormatter(format)
	return f.Format(os.Stdout, data)
}

// FilterFields filters rows to only include specified fields.
func FilterFields(rows []map[string]any, fields string) []map[string]any {
	if fields == "" {
		return rows
	}
	fieldList := strings.Split(fields, ",")
	fieldSet := make(map[string]bool, len(fieldList))
	for _, f := range fieldList {
		fieldSet[strings.TrimSpace(f)] = true
	}

	filtered := make([]map[string]any, len(rows))
	for i, row := range rows {
		newRow := make(map[string]any)
		for k, v := range row {
			if fieldSet[k] {
				newRow[k] = v
			}
		}
		filtered[i] = newRow
	}
	return filtered
}

// PrintError writes a structured error to stderr.
func PrintError(errType, message, suggestion string) {
	fmt.Fprintf(os.Stderr, `{"error":%q,"message":%q,"suggestion":%q}`, errType, message, suggestion)
	fmt.Fprintln(os.Stderr)
}
