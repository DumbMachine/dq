package postgres

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresDriver struct{}

func init() {
	database.Register("postgres", &PostgresDriver{})
}

func (d *PostgresDriver) Type() string {
	return "postgres"
}

func (d *PostgresDriver) Connect(cfg *config.ConnectionConfig) (*gorm.DB, error) {
	password := cfg.Password
	if password != "" {
		resolved, _, err := config.ResolvePassword(password)
		if err != nil {
			return nil, fmt.Errorf("resolving password: %w", err)
		}
		password = resolved
	}

	port := cfg.Port
	if port == 0 {
		port = cfg.DefaultPort()
	}

	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, port, cfg.User, password, cfg.Database, sslMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	return db, nil
}
