package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Log      LogConfig      `yaml:"log"`
	Database DatabaseConfig `yaml:"database"`
	JWT      JWTConfig      `yaml:"jwt"`
	Admin    AdminConfig    `yaml:"admin"`
	Xray     XrayConfig     `yaml:"xray"`
	Nginx    NginxConfig    `yaml:"nginx"`
}

// ServerConfig holds web server settings
type ServerConfig struct {
	Listen string `yaml:"listen"`
	Debug  bool   `yaml:"debug"`
}

// LogConfig holds logging settings
type LogConfig struct {
	Level      string `yaml:"level"`       // debug, info, warning, error
	File       string `yaml:"file"`        // log file path, empty for stdout only
	MaxSize    int    `yaml:"max_size"`    // max size in MB before rotation
	MaxBackups int    `yaml:"max_backups"` // max number of old log files
	MaxAge     int    `yaml:"max_age"`     // max days to retain old log files
	Compress   bool   `yaml:"compress"`    // compress rotated files
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// JWTConfig holds JWT authentication settings
type JWTConfig struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

// AdminConfig holds initial admin account settings
type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Email    string `yaml:"email"`
}

// XrayConfig holds Xray core settings
type XrayConfig struct {
	BinaryPath string `yaml:"binary_path"`
	ConfigPath string `yaml:"config_path"`
	AssetsPath string `yaml:"assets_path"`
	APIPort    int    `yaml:"api_port"`
	SocketDir  string `yaml:"socket_dir"`
}

// NginxConfig holds Nginx settings
type NginxConfig struct {
	ConfigDir string `yaml:"config_dir"`
	StreamDir string `yaml:"stream_dir"`
	ReloadCmd string `yaml:"reload_cmd"`
	CertDir   string `yaml:"cert_dir"`
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks configuration values for obvious problems
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt.secret 不能为空")
	}
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("jwt.secret 长度至少 16 位，当前 %d 位", len(c.JWT.Secret))
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path 不能为空")
	}
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen 不能为空")
	}
	return nil
}

// Default returns default configuration
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Listen: "127.0.0.1:8082",
			Debug:  false,
		},
		Log: LogConfig{
			Level:      "info",
			File:       "/opt/xray-panel/logs/panel.log",
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
			Compress:   true,
		},
		Database: DatabaseConfig{
			Path: "/opt/xray-panel/data/panel.db",
		},
		JWT: JWTConfig{
			Secret:     "change-me-in-production",
			ExpireHour: 168,
		},
		Admin: AdminConfig{
			Username: "",
			Password: "",
			Email:    "",
		},
		Xray: XrayConfig{
			BinaryPath: "/usr/local/bin/xray",
			ConfigPath: "/usr/local/etc/xray/config.json",
			AssetsPath: "/usr/local/share/xray",
			APIPort:    10085,
			SocketDir:  "/dev/shm",
		},
		Nginx: NginxConfig{
			ConfigDir: "/etc/nginx/conf.d",
			StreamDir: "/etc/nginx/stream.d",
			ReloadCmd: "systemctl reload nginx",
			CertDir:   "/root/.acme.sh",
		},
	}
}
