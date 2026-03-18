package config

type ConnectionConfig struct {
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Type     string `yaml:"type" json:"type"`
	Host     string `yaml:"host,omitempty" json:"host,omitempty"`
	Port     int    `yaml:"port,omitempty" json:"port,omitempty"`
	Database string `yaml:"database,omitempty" json:"database,omitempty"`
	User     string `yaml:"user,omitempty" json:"user,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	Path     string `yaml:"path,omitempty" json:"path,omitempty"` // For SQLite
	SSLMode  string `yaml:"ssl_mode,omitempty" json:"ssl_mode,omitempty"`
}

func (c *ConnectionConfig) DefaultPort() int {
	switch c.Type {
	case "postgres":
		return 5432
	case "mysql":
		return 3306
	default:
		return 0
	}
}
