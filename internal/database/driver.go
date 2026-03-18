package database

import (
	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/pkg/types"
	"gorm.io/gorm"
)

type Driver interface {
	Connect(cfg *config.ConnectionConfig) (*gorm.DB, error)
	Type() string

	// Introspection
	ListSchemas(db *gorm.DB) ([]string, error)
	ListTables(db *gorm.DB, schema string) ([]types.TableInfo, error)
	ListColumns(db *gorm.DB, schema, table string) ([]types.ColumnInfo, error)
	ListIndexes(db *gorm.DB, schema, table string) ([]types.IndexInfo, error)
	ListConstraints(db *gorm.DB, schema, table string) ([]types.ConstraintInfo, error)
	ListForeignKeys(db *gorm.DB, schema, table string) ([]types.ForeignKey, error)
}
