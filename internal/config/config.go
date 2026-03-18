package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Connections map[string]*ConnectionConfig `yaml:"connections"`
}

func ConfigDir() string {
	if dir := os.Getenv("DQ_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dq")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Connections: make(map[string]*ConnectionConfig)}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Connections == nil {
		cfg.Connections = make(map[string]*ConnectionConfig)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(ConfigPath(), data, 0600)
}

func (c *Config) GetConnection(name string) (*ConnectionConfig, error) {
	conn, ok := c.Connections[name]
	if !ok {
		return nil, fmt.Errorf("connection %q not found", name)
	}
	conn.Name = name
	return conn, nil
}

func (c *Config) AddConnection(name string, conn *ConnectionConfig) error {
	if _, exists := c.Connections[name]; exists {
		return fmt.Errorf("connection %q already exists", name)
	}
	c.Connections[name] = conn
	return c.Save()
}

func (c *Config) RemoveConnection(name string) error {
	if _, exists := c.Connections[name]; !exists {
		return fmt.Errorf("connection %q not found", name)
	}
	delete(c.Connections, name)
	return c.Save()
}
