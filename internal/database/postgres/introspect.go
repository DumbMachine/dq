package postgres

import (
	"github.com/dumbmachine/db-cli/pkg/types"
	"gorm.io/gorm"
)

func (d *PostgresDriver) ListSchemas(db *gorm.DB) ([]string, error) {
	var schemas []string
	err := db.Raw(`
		SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name
	`).Scan(&schemas).Error
	return schemas, err
}

func (d *PostgresDriver) ListTables(db *gorm.DB, schema string) ([]types.TableInfo, error) {
	if schema == "" {
		schema = "public"
	}
	var tables []types.TableInfo
	err := db.Raw(`
		SELECT
			t.table_schema AS "schema",
			t.table_name AS name,
			t.table_type AS type,
			COALESCE(s.n_live_tup, 0) AS row_count,
			COALESCE(pg_size_pretty(pg_total_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name))), '') AS size,
			COALESCE(pg_total_relation_size(quote_ident(t.table_schema) || '.' || quote_ident(t.table_name)), 0) AS size_bytes
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s
			ON s.schemaname = t.table_schema AND s.relname = t.table_name
		WHERE t.table_schema = ?
		ORDER BY t.table_name
	`, schema).Scan(&tables).Error
	return tables, err
}

func (d *PostgresDriver) ListColumns(db *gorm.DB, schema, table string) ([]types.ColumnInfo, error) {
	if schema == "" {
		schema = "public"
	}
	var columns []types.ColumnInfo
	err := db.Raw(`
		SELECT
			c.column_name AS name,
			c.data_type || CASE
				WHEN c.character_maximum_length IS NOT NULL THEN '(' || c.character_maximum_length || ')'
				ELSE ''
			END AS type,
			(c.is_nullable = 'YES') AS nullable,
			COALESCE(c.column_default, '') AS "default",
			EXISTS (
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
					ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'PRIMARY KEY'
					AND tc.table_schema = c.table_schema
					AND tc.table_name = c.table_name
					AND kcu.column_name = c.column_name
			) AS primary_key,
			EXISTS (
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
					ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
				WHERE tc.constraint_type = 'UNIQUE'
					AND tc.table_schema = c.table_schema
					AND tc.table_name = c.table_name
					AND kcu.column_name = c.column_name
			) AS "unique"
		FROM information_schema.columns c
		WHERE c.table_schema = ? AND c.table_name = ?
		ORDER BY c.ordinal_position
	`, schema, table).Scan(&columns).Error
	return columns, err
}

func (d *PostgresDriver) ListIndexes(db *gorm.DB, schema, table string) ([]types.IndexInfo, error) {
	if schema == "" {
		schema = "public"
	}

	type indexRow struct {
		Name    string `gorm:"column:name"`
		Column  string `gorm:"column:column_name"`
		Unique  bool   `gorm:"column:is_unique"`
		IdxType string `gorm:"column:idx_type"`
	}

	var rows []indexRow
	err := db.Raw(`
		SELECT
			i.relname AS name,
			a.attname AS column_name,
			ix.indisunique AS is_unique,
			am.amname AS idx_type
		FROM pg_index ix
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		JOIN pg_am am ON am.oid = i.relam
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE n.nspname = ? AND t.relname = ?
		ORDER BY i.relname, a.attnum
	`, schema, table).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	indexMap := make(map[string]*types.IndexInfo)
	var order []string
	for _, r := range rows {
		idx, ok := indexMap[r.Name]
		if !ok {
			idx = &types.IndexInfo{
				Name:   r.Name,
				Unique: r.Unique,
				Type:   r.IdxType,
			}
			indexMap[r.Name] = idx
			order = append(order, r.Name)
		}
		idx.Columns = append(idx.Columns, r.Column)
	}

	result := make([]types.IndexInfo, 0, len(order))
	for _, name := range order {
		result = append(result, *indexMap[name])
	}
	return result, nil
}

func (d *PostgresDriver) ListConstraints(db *gorm.DB, schema, table string) ([]types.ConstraintInfo, error) {
	if schema == "" {
		schema = "public"
	}

	type constraintRow struct {
		Name       string `gorm:"column:name"`
		Type       string `gorm:"column:type"`
		Column     string `gorm:"column:column_name"`
		RefTable   string `gorm:"column:ref_table"`
		RefColumn  string `gorm:"column:ref_column"`
		Definition string `gorm:"column:definition"`
	}

	var rows []constraintRow
	err := db.Raw(`
		SELECT
			tc.constraint_name AS name,
			tc.constraint_type AS type,
			kcu.column_name AS column_name,
			COALESCE(ccu.table_name, '') AS ref_table,
			COALESCE(ccu.column_name, '') AS ref_column,
			COALESCE(cc.check_clause, '') AS definition
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		LEFT JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
			AND tc.constraint_type = 'FOREIGN KEY'
		LEFT JOIN information_schema.check_constraints cc
			ON tc.constraint_name = cc.constraint_name AND tc.constraint_schema = cc.constraint_schema
		WHERE tc.table_schema = ? AND tc.table_name = ?
		ORDER BY tc.constraint_name, kcu.ordinal_position
	`, schema, table).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	cMap := make(map[string]*types.ConstraintInfo)
	var order []string
	for _, r := range rows {
		c, ok := cMap[r.Name]
		if !ok {
			c = &types.ConstraintInfo{
				Name:       r.Name,
				Type:       r.Type,
				RefTable:   r.RefTable,
				Definition: r.Definition,
			}
			cMap[r.Name] = c
			order = append(order, r.Name)
		}
		c.Columns = appendUnique(c.Columns, r.Column)
		if r.RefColumn != "" {
			c.RefColumns = appendUnique(c.RefColumns, r.RefColumn)
		}
	}

	result := make([]types.ConstraintInfo, 0, len(order))
	for _, name := range order {
		result = append(result, *cMap[name])
	}
	return result, nil
}

func (d *PostgresDriver) ListForeignKeys(db *gorm.DB, schema, table string) ([]types.ForeignKey, error) {
	if schema == "" {
		schema = "public"
	}
	var fks []types.ForeignKey
	err := db.Raw(`
		SELECT
			kcu.column_name AS "column",
			ccu.table_name AS ref_table,
			ccu.column_name AS ref_column,
			tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = ? AND tc.table_name = ?
		ORDER BY kcu.column_name
	`, schema, table).Scan(&fks).Error
	return fks, err
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
