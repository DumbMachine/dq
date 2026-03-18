package output

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type TableFormatter struct{}

func (f *TableFormatter) Format(w io.Writer, data any) error {
	switch v := data.(type) {
	case []map[string]any:
		return f.formatRows(w, v)
	case map[string]any:
		return f.formatSingle(w, v)
	default:
		jf := &JSONFormatter{}
		return jf.Format(w, data)
	}
}

func (f *TableFormatter) formatRows(w io.Writer, rows []map[string]any) error {
	if len(rows) == 0 {
		fmt.Fprintln(w, "(0 rows)")
		return nil
	}

	headers := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	table := tablewriter.NewWriter(w)
	table.Header(headers)

	for _, row := range rows {
		vals := make([]string, len(headers))
		for i, h := range headers {
			vals[i] = fmt.Sprintf("%v", row[h])
		}
		table.Append(vals)
	}

	return table.Render()
}

func (f *TableFormatter) formatSingle(w io.Writer, row map[string]any) error {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxKeyLen := 0
	for _, k := range keys {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	for _, k := range keys {
		fmt.Fprintf(w, "%-*s  %v\n", maxKeyLen, k, formatValue(row[k]))
	}
	return nil
}

func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}
