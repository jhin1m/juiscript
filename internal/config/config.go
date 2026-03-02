package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	DefaultConfigDir  = "/etc/juiscript"
	DefaultConfigFile = "config.toml"
	DefaultSitesDir   = "sites"
)

// Config holds all juiscript configuration.
type Config struct {
	General  GeneralConfig  `toml:"general"`
	Nginx    NginxConfig    `toml:"nginx"`
	PHP      PHPConfig      `toml:"php"`
	Database DatabaseConfig `toml:"database"`
	Backup   BackupConfig   `toml:"backup"`
	Redis    RedisConfig    `toml:"redis"`
}

type GeneralConfig struct {
	SitesRoot string `toml:"sites_root"` // base dir for site home dirs
	LogFile   string `toml:"log_file"`
	BackupDir string `toml:"backup_dir"`
}

type NginxConfig struct {
	SitesAvailable string `toml:"sites_available"`
	SitesEnabled   string `toml:"sites_enabled"`
	ConfDir        string `toml:"conf_dir"`
}

type PHPConfig struct {
	DefaultVersion string   `toml:"default_version"`
	Versions       []string `toml:"versions"` // installed versions
}

type DatabaseConfig struct {
	RootUser   string `toml:"root_user"`
	SocketPath string `toml:"socket_path"`
}

type BackupConfig struct {
	Dir            string `toml:"dir"`
	RetentionDays  int    `toml:"retention_days"`
	CompressLevel  int    `toml:"compress_level"`
}

type RedisConfig struct {
	MaxDatabases int `toml:"max_databases"`
}

// Default returns config with sensible defaults for Ubuntu.
func Default() *Config {
	return &Config{
		General: GeneralConfig{
			SitesRoot: "/home",
			LogFile:   "/var/log/juiscript.log",
			BackupDir: "/var/backups/juiscript",
		},
		Nginx: NginxConfig{
			SitesAvailable: "/etc/nginx/sites-available",
			SitesEnabled:   "/etc/nginx/sites-enabled",
			ConfDir:        "/etc/nginx/conf.d",
		},
		PHP: PHPConfig{
			DefaultVersion: "8.3",
			Versions:       []string{"8.3"},
		},
		Database: DatabaseConfig{
			RootUser:   "root",
			SocketPath: "/var/run/mysqld/mysqld.sock",
		},
		Backup: BackupConfig{
			Dir:           "/var/backups/juiscript",
			RetentionDays: 30,
			CompressLevel: 6,
		},
		Redis: RedisConfig{
			MaxDatabases: 16,
		},
	}
}

// ConfigPath returns the full path to config file.
func ConfigPath() string {
	return filepath.Join(DefaultConfigDir, DefaultConfigFile)
}

// SitesPath returns the path to site metadata directory.
func SitesPath() string {
	return filepath.Join(DefaultConfigDir, DefaultSitesDir)
}

// Load reads config from the given path. Falls back to defaults on missing file.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // use defaults if no config file
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

// Save writes config to the given path with restrictive permissions.
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return nil
}

// EnsureDirs creates required directories for juiscript operation.
func EnsureDirs(cfg *Config) error {
	dirs := []string{
		DefaultConfigDir,
		SitesPath(),
		cfg.General.BackupDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	return nil
}
