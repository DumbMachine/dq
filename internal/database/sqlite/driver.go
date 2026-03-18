package sqlite

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SQLiteDriver struct{}

func init() {
	database.Register("sqlite", &SQLiteDriver{})
}

func (d *SQLiteDriver) Type() string {
	return "sqlite"
}

func (d *SQLiteDriver) Connect(cfg *config.ConnectionConfig) (*gorm.DB, error) {
	path := cfg.Path
	if path == "" {
		path = cfg.Database
	}
	if path == "" {
		return nil, fmt.Errorf("sqlite requires a path or database field")
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to sqlite: %w", err)
	}

	return db, nil
}
