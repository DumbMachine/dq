package output

import (
	"encoding/json"
	"io"
)

type NDJSONFormatter struct{}

func (f *NDJSONFormatter) Format(w io.Writer, data any) error {
	switch v := data.(type) {
	case []map[string]any:
		enc := json.NewEncoder(w)
		for _, row := range v {
			if err := enc.Encode(row); err != nil {
				return err
			}
		}
		return nil
	default:
		enc := json.NewEncoder(w)
		return enc.Encode(data)
	}
}
