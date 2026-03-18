package sqlite

import (
	"fmt"
	"strings"

	"github.com/dumbmachine/db-cli/pkg/types"
	"gorm.io/gorm"
)

func (d *SQLiteDriver) ListSchemas(db *gorm.DB) ([]string, error) {
	return []string{"main"}, nil
}

func (d *SQLiteDriver) ListTables(db *gorm.DB, schema string) ([]types.TableInfo, error) {
	var names []string
	err := db.Raw(`SELECT name FROM sqlite_master WHERE type IN ('table', 'view') AND name NOT LIKE 'sqlite_%' ORDER BY name`).Scan(&names).Error
	if err != nil {
		return nil, err
	}

	tables := make([]types.TableInfo, 0, len(names))
	for _, name := range names {
		var count int64
		db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %q", name)).Scan(&count)
		tables = append(tables, types.TableInfo{
			Schema:   "main",
			Name:     name,
			Type:     "table",
			RowCount: count,
		})
	}
	return tables, nil
}

func (d *SQLiteDriver) ListColumns(db *gorm.DB, schema, table string) ([]types.ColumnInfo, error) {
	type pragmaCol struct {
		CID       int    `gorm:"column:cid"`
		Name      string `gorm:"column:name"`
		Type      string `gorm:"column:type"`
		NotNull   int    `gorm:"column:notnull"`
		DfltValue *string `gorm:"column:dflt_value"`
		PK        int    `gorm:"column:pk"`
	}

	var cols []pragmaCol
	err := db.Raw(fmt.Sprintf("PRAGMA table_info(%q)", table)).Scan(&cols).Error
	if err != nil {
		return nil, err
	}

	columns := make([]types.ColumnInfo, len(cols))
	for i, c := range cols {
		col := types.ColumnInfo{
			Name:       c.Name,
			Type:       c.Type,
			Nullable:   c.NotNull == 0,
			PrimaryKey: c.PK > 0,
		}
		if c.DfltValue != nil {
			col.Default = *c.DfltValue
		}
		columns[i] = col
	}
	return columns, nil
}

func (d *SQLiteDriver) ListIndexes(db *gorm.DB, schema, table string) ([]types.IndexInfo, error) {
	type indexListRow struct {
		Name   string `gorm:"column:name"`
		Unique int    `gorm:"column:unique"`
	}

	var indexes []indexListRow
	err := db.Raw(fmt.Sprintf("PRAGMA index_list(%q)", table)).Scan(&indexes).Error
	if err != nil {
		return nil, err
	}

	result := make([]types.IndexInfo, 0, len(indexes))
	for _, idx := range indexes {
		if idx.Name == "" {
			continue
		}
		type indexInfoRow struct {
			Name string `gorm:"column:name"`
		}
		var cols []indexInfoRow
		db.Raw(fmt.Sprintf("PRAGMA index_info(%q)", idx.Name)).Scan(&cols)

		colNames := make([]string, len(cols))
		for i, c := range cols {
			colNames[i] = c.Name
		}

		result = append(result, types.IndexInfo{
			Name:    idx.Name,
			Columns: colNames,
			Unique:  idx.Unique == 1,
		})
	}
	return result, nil
}

func (d *SQLiteDriver) ListConstraints(db *gorm.DB, schema, table string) ([]types.ConstraintInfo, error) {
	// SQLite doesn't have a clean constraints view; extract PK and FK info
	var constraints []types.ConstraintInfo

	// Primary key
	columns, err := d.ListColumns(db, schema, table)
	if err != nil {
		return nil, err
	}
	var pkCols []string
	for _, c := range columns {
		if c.PrimaryKey {
			pkCols = append(pkCols, c.Name)
		}
	}
	if len(pkCols) > 0 {
		constraints = append(constraints, types.ConstraintInfo{
			Name:    "pk_" + table,
			Type:    "PRIMARY KEY",
			Columns: pkCols,
		})
	}

	// Foreign keys
	fks, err := d.ListForeignKeys(db, schema, table)
	if err == nil {
		for _, fk := range fks {
			constraints = append(constraints, types.ConstraintInfo{
				Name:       fk.ConstraintName,
				Type:       "FOREIGN KEY",
				Columns:    []string{fk.Column},
				RefTable:   fk.RefTable,
				RefColumns: []string{fk.RefColumn},
			})
		}
	}

	return constraints, nil
}

func (d *SQLiteDriver) ListForeignKeys(db *gorm.DB, schema, table string) ([]types.ForeignKey, error) {
	type fkRow struct {
		ID    int    `gorm:"column:id"`
		Table string `gorm:"column:table"`
		From  string `gorm:"column:from"`
		To    string `gorm:"column:to"`
	}

	var rows []fkRow
	err := db.Raw(fmt.Sprintf("PRAGMA foreign_key_list(%q)", table)).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	fks := make([]types.ForeignKey, len(rows))
	for i, r := range rows {
		fks[i] = types.ForeignKey{
			Column:         r.From,
			RefTable:       r.Table,
			RefColumn:      r.To,
			ConstraintName: fmt.Sprintf("fk_%s_%s", table, strings.ToLower(r.From)),
		}
	}
	return fks, nil
}
