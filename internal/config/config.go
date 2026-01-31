package config

import (
	"os"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Paths     PathsConfig     `yaml:"paths"`
	Services  ServicesConfig  `yaml:"services"`
	PHP       PHPConfig       `yaml:"php"`
	Backup    BackupConfig    `yaml:"backup"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type ServerConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	SecretKey   string `yaml:"secret_key"`
	EnableHTTPS bool   `yaml:"enable_https"`
	CertFile    string `yaml:"cert_file"`
	KeyFile     string `yaml:"key_file"`
}

type DatabaseConfig struct {
	Path         string `yaml:"path"`
	MySQLHost    string `yaml:"mysql_host"`
	MySQLPort    int    `yaml:"mysql_port"`
	MySQLUser    string `yaml:"mysql_user"`
	MySQLPass    string `yaml:"mysql_pass"`
}

type PathsConfig struct {
	WWWRoot   string `yaml:"www_root"`
	BackupDir string `yaml:"backup_dir"`
	LogDir    string `yaml:"log_dir"`
	SSLDir    string `yaml:"ssl_dir"`
	NginxConf string `yaml:"nginx_conf"`
}

type ServicesConfig struct {
	WebServer string `yaml:"webserver"`
}

type PHPConfig struct {
	DefaultVersion string   `yaml:"default_version"`
	Versions       []string `yaml:"versions"`
}

type BackupConfig struct {
	RetentionDays int  `yaml:"retention_days"`
	Compress      bool `yaml:"compress"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	LoginAttempts     int  `yaml:"login_attempts"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Default() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host:        "127.0.0.1",
			Port:        8080,
			SecretKey:   "change-me-in-production",
			EnableHTTPS: false,
			CertFile:    "",
			KeyFile:     "",
		},
		Database: DatabaseConfig{
			Path:      "data/panel.db",
			MySQLHost: "localhost",
			MySQLPort: 3306,
			MySQLUser: "root",
			MySQLPass: "",
		},
		Services: ServicesConfig{
			WebServer: "nginx",
		},
		PHP: PHPConfig{
			DefaultVersion: "8.2",
			Versions:       []string{"7.4", "8.0", "8.1", "8.2", "8.3"},
		},
		Backup: BackupConfig{
			RetentionDays: 7,
			Compress:      true,
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 60,
			LoginAttempts:     5,
		},
	}

	// Platform-specific paths
	if runtime.GOOS == "windows" {
		cfg.Paths = PathsConfig{
			WWWRoot:   "C:\\mcontrolpanel\\www",
			BackupDir: "C:\\mcontrolpanel\\backups",
			LogDir:    "C:\\mcontrolpanel\\logs",
			SSLDir:    "C:\\mcontrolpanel\\ssl",
			NginxConf: "C:\\nginx\\conf\\sites-enabled",
		}
	} else {
		cfg.Paths = PathsConfig{
			WWWRoot:   "/var/www",
			BackupDir: "/var/backups/mcontrolpanel",
			LogDir:    "/var/log/mcontrolpanel",
			SSLDir:    "/etc/ssl/mcontrolpanel",
			NginxConf: "/etc/nginx/sites-enabled",
		}
	}

	return cfg
}
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
