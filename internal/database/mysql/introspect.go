package mysql

import (
	"github.com/dumbmachine/db-cli/pkg/types"
	"gorm.io/gorm"
)

func (d *MySQLDriver) ListSchemas(db *gorm.DB) ([]string, error) {
	var schemas []string
	err := db.Raw("SHOW DATABASES").Scan(&schemas).Error
	return schemas, err
}

func (d *MySQLDriver) ListTables(db *gorm.DB, schema string) ([]types.TableInfo, error) {
	if schema == "" {
		schema = db.Migrator().CurrentDatabase()
	}
	var tables []types.TableInfo
	err := db.Raw(`
		SELECT
			TABLE_SCHEMA AS 'schema',
			TABLE_NAME AS name,
			TABLE_TYPE AS type,
			COALESCE(TABLE_ROWS, 0) AS row_count,
			COALESCE(CONCAT(ROUND((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024, 2), ' MB'), '') AS size,
			COALESCE(DATA_LENGTH + INDEX_LENGTH, 0) AS size_bytes
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		ORDER BY TABLE_NAME
	`, schema).Scan(&tables).Error
	return tables, err
}

func (d *MySQLDriver) ListColumns(db *gorm.DB, schema, table string) ([]types.ColumnInfo, error) {
	if schema == "" {
		schema = db.Migrator().CurrentDatabase()
	}
	var columns []types.ColumnInfo
	err := db.Raw(`
		SELECT
			COLUMN_NAME AS name,
			COLUMN_TYPE AS type,
			(IS_NULLABLE = 'YES') AS nullable,
			COALESCE(COLUMN_DEFAULT, '') AS 'default',
			(COLUMN_KEY = 'PRI') AS primary_key,
			(COLUMN_KEY = 'UNI') AS 'unique'
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION
	`, schema, table).Scan(&columns).Error
	return columns, err
}

func (d *MySQLDriver) ListIndexes(db *gorm.DB, schema, table string) ([]types.IndexInfo, error) {
	if schema == "" {
		schema = db.Migrator().CurrentDatabase()
	}

	type indexRow struct {
		Name   string `gorm:"column:name"`
		Column string `gorm:"column:column_name"`
		Unique bool   `gorm:"column:is_unique"`
	}

	var rows []indexRow
	err := db.Raw(`
		SELECT
			INDEX_NAME AS name,
			COLUMN_NAME AS column_name,
			(NON_UNIQUE = 0) AS is_unique
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`, schema, table).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	indexMap := make(map[string]*types.IndexInfo)
	var order []string
	for _, r := range rows {
		idx, ok := indexMap[r.Name]
		if !ok {
			idx = &types.IndexInfo{Name: r.Name, Unique: r.Unique}
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

func (d *MySQLDriver) ListConstraints(db *gorm.DB, schema, table string) ([]types.ConstraintInfo, error) {
	if schema == "" {
		schema = db.Migrator().CurrentDatabase()
	}

	type constraintRow struct {
		Name      string `gorm:"column:name"`
		Type      string `gorm:"column:type"`
		Column    string `gorm:"column:column_name"`
		RefTable  string `gorm:"column:ref_table"`
		RefColumn string `gorm:"column:ref_column"`
	}

	var rows []constraintRow
	err := db.Raw(`
		SELECT
			tc.CONSTRAINT_NAME AS name,
			tc.CONSTRAINT_TYPE AS type,
			kcu.COLUMN_NAME AS column_name,
			COALESCE(kcu.REFERENCED_TABLE_NAME, '') AS ref_table,
			COALESCE(kcu.REFERENCED_COLUMN_NAME, '') AS ref_column
		FROM information_schema.TABLE_CONSTRAINTS tc
		JOIN information_schema.KEY_COLUMN_USAGE kcu
			ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA AND tc.TABLE_NAME = kcu.TABLE_NAME
		WHERE tc.TABLE_SCHEMA = ? AND tc.TABLE_NAME = ?
		ORDER BY tc.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
	`, schema, table).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	cMap := make(map[string]*types.ConstraintInfo)
	var order []string
	for _, r := range rows {
		c, ok := cMap[r.Name]
		if !ok {
			c = &types.ConstraintInfo{Name: r.Name, Type: r.Type, RefTable: r.RefTable}
			cMap[r.Name] = c
			order = append(order, r.Name)
		}
		c.Columns = append(c.Columns, r.Column)
		if r.RefColumn != "" {
			c.RefColumns = append(c.RefColumns, r.RefColumn)
		}
	}

	result := make([]types.ConstraintInfo, 0, len(order))
	for _, name := range order {
		result = append(result, *cMap[name])
	}
	return result, nil
}

func (d *MySQLDriver) ListForeignKeys(db *gorm.DB, schema, table string) ([]types.ForeignKey, error) {
	if schema == "" {
		schema = db.Migrator().CurrentDatabase()
	}
	var fks []types.ForeignKey
	err := db.Raw(`
		SELECT
			kcu.COLUMN_NAME AS 'column',
			kcu.REFERENCED_TABLE_NAME AS ref_table,
			kcu.REFERENCED_COLUMN_NAME AS ref_column,
			kcu.CONSTRAINT_NAME AS constraint_name
		FROM information_schema.KEY_COLUMN_USAGE kcu
		JOIN information_schema.TABLE_CONSTRAINTS tc
			ON kcu.CONSTRAINT_NAME = tc.CONSTRAINT_NAME AND kcu.TABLE_SCHEMA = tc.TABLE_SCHEMA AND kcu.TABLE_NAME = tc.TABLE_NAME
		WHERE tc.CONSTRAINT_TYPE = 'FOREIGN KEY'
			AND kcu.TABLE_SCHEMA = ? AND kcu.TABLE_NAME = ?
		ORDER BY kcu.COLUMN_NAME
	`, schema, table).Scan(&fks).Error
	return fks, err
}
