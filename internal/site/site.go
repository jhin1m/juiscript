package site

import (
	"time"
)

// ProjectType identifies the application framework for a site.
type ProjectType string

const (
	ProjectLaravel   ProjectType = "laravel"
	ProjectWordPress ProjectType = "wordpress"
)

// Site holds metadata for a managed website.
// Each site corresponds to one Linux user, one PHP-FPM pool, one Nginx vhost.
type Site struct {
	Domain      string      `toml:"domain"`
	User        string      `toml:"user"`
	ProjectType ProjectType `toml:"project_type"`
	PHPVersion  string      `toml:"php_version"`
	WebRoot     string      `toml:"web_root"`
	DBName      string      `toml:"db_name"`
	DBUser      string      `toml:"db_user"`
	SSLEnabled  bool        `toml:"ssl_enabled"`
	Enabled     bool        `toml:"enabled"`
	CreatedAt   time.Time   `toml:"created_at"`
}

// CreateOptions holds parameters for creating a new site.
type CreateOptions struct {
	Domain      string
	ProjectType ProjectType
	PHPVersion  string
	CreateDB    bool
}

// HomeDir returns the user's home directory path.
func (s *Site) HomeDir(sitesRoot string) string {
	return sitesRoot + "/" + s.User
}

// SiteDir returns the site application directory.
func (s *Site) SiteDir(sitesRoot string) string {
	home := s.HomeDir(sitesRoot)
	switch s.ProjectType {
	case ProjectLaravel:
		return home + "/" + s.Domain
	case ProjectWordPress:
		return home + "/public_html/" + s.Domain
	default:
		return home + "/" + s.Domain
	}
}

// PHPSocket returns the PHP-FPM socket path for this site.
func (s *Site) PHPSocket() string {
	return "/run/php/php" + s.PHPVersion + "-fpm-" + s.User + ".sock"
}

// FPMPoolConfigPath returns the path to the PHP-FPM pool config.
func (s *Site) FPMPoolConfigPath() string {
	return "/etc/php/" + s.PHPVersion + "/fpm/pool.d/" + s.Domain + ".conf"
}

// NginxConfigPath returns the path in sites-available.
func (s *Site) NginxConfigPath(sitesAvailable string) string {
	return sitesAvailable + "/" + s.Domain + ".conf"
}

// NginxEnabledPath returns the symlink path in sites-enabled.
func (s *Site) NginxEnabledPath(sitesEnabled string) string {
	return sitesEnabled + "/" + s.Domain + ".conf"
}

// MetadataPath returns the path to the site's TOML metadata file.
func (s *Site) MetadataPath(sitesDir string) string {
	return sitesDir + "/" + s.Domain + ".toml"
}
