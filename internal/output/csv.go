package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
)

type CSVFormatter struct{}

func (f *CSVFormatter) Format(w io.Writer, data any) error {
	rows, ok := data.([]map[string]any)
	if !ok {
		jf := &JSONFormatter{}
		return jf.Format(w, data)
	}

	if len(rows) == 0 {
		return nil
	}

	headers := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	cw := csv.NewWriter(w)
	if err := cw.Write(headers); err != nil {
		return err
	}

	for _, row := range rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			record[i] = fmt.Sprintf("%v", row[h])
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}
