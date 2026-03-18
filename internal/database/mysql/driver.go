package mysql

import (
	"fmt"

	"github.com/dumbmachine/db-cli/internal/config"
	"github.com/dumbmachine/db-cli/internal/database"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MySQLDriver struct{}

func init() {
	database.Register("mysql", &MySQLDriver{})
}

func (d *MySQLDriver) Type() string {
	return "mysql"
}

func (d *MySQLDriver) Connect(cfg *config.ConnectionConfig) (*gorm.DB, error) {
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

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, password, cfg.Host, port, cfg.Database)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to mysql: %w", err)
	}

	return db, nil
}
