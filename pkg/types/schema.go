package types

type TableInfo struct {
	Schema    string `json:"schema,omitempty"`
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	RowCount  int64  `json:"row_count"`
	Size      string `json:"size,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type ColumnInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Nullable   bool   `json:"nullable"`
	Default    string `json:"default,omitempty"`
	PrimaryKey bool   `json:"primary_key,omitempty"`
	Unique     bool   `json:"unique,omitempty"`
}

type IndexInfo struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type,omitempty"`
}

type ConstraintInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Columns    []string `json:"columns"`
	RefTable   string   `json:"ref_table,omitempty"`
	RefColumns []string `json:"ref_columns,omitempty"`
	Definition string   `json:"definition,omitempty"`
}

type ForeignKey struct {
	Column         string `json:"column"`
	RefTable       string `json:"ref_table"`
	RefColumn      string `json:"ref_column"`
	ConstraintName string `json:"constraint_name,omitempty"`
}
