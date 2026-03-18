package types

import "time"

type DiscoverResult struct {
	Connection string           `json:"connection"`
	Database   string           `json:"database"`
	CachedAt   time.Time        `json:"cached_at"`
	Schemas    []SchemaOverview `json:"schemas"`
}

type SchemaOverview struct {
	Name   string          `json:"name"`
	Tables []TableOverview `json:"tables"`
}

type TableOverview struct {
	Name        string           `json:"name"`
	Type        string           `json:"type,omitempty"`
	RowCount    int64            `json:"row_count"`
	Size        string           `json:"size,omitempty"`
	SizeBytes   int64            `json:"size_bytes,omitempty"`
	Columns     []ColumnInfo     `json:"columns"`
	ForeignKeys []ForeignKey     `json:"foreign_keys,omitempty"`
	Indexes     []IndexInfo      `json:"indexes,omitempty"`
	Annotations *TableAnnotation `json:"annotations,omitempty"`
}

type TableAnnotation struct {
	Table   string            `json:"table,omitempty"`
	Columns map[string]string `json:"columns,omitempty"`
}
