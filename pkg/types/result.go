package types

type QueryResult struct {
	Meta    ResultMeta       `json:"meta"`
	Columns []ColumnMeta     `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

type ResultMeta struct {
	Connection   string `json:"connection"`
	Database     string `json:"database,omitempty"`
	RowCount     int    `json:"row_count"`
	AffectedRows int64  `json:"affected_rows,omitempty"`
	DurationMs   int64  `json:"duration_ms"`
	DryRun       bool   `json:"dry_run,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
	TotalRows    int64  `json:"total_rows,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
}

type ColumnMeta struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
