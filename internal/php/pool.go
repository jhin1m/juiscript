package php

import (
	"context"
	"fmt"
	"strings"
)

// PoolConfig holds all settings for a PHP-FPM pool.
// Matches the fields expected by the php-fpm-pool.conf.tmpl template.
type PoolConfig struct {
	SiteDomain    string
	Username      string
	PHPVersion    string
	ListenSocket  string // /run/php/php{ver}-fpm-{user}.sock
	MaxChildren   int    // default: 5
	StartServers  int    // default: 2
	MinSpare      int    // default: 1
	MaxSpare      int    // default: 3
	MaxRequests   int    // default: 500
	MemoryLimit   string // default: "256M"
	UploadMaxSize string // default: "64M"
	Timezone      string // default: "UTC"
}

// poolTemplateData maps PoolConfig to the template's expected field names.
type poolTemplateData struct {
	PoolName      string
	User          string
	SocketPath    string
	MaxChildren   int
	StartServers  int
	MinSpare      int
	MaxSpare      int
	MaxRequests   int
	MemoryLimit   string
	UploadMaxSize string
	Timezone      string
}

// DefaultPool returns a PoolConfig with sensible defaults filled in.
func DefaultPool(domain, username, phpVersion string) PoolConfig {
	return PoolConfig{
		SiteDomain:    domain,
		Username:      username,
		PHPVersion:    phpVersion,
		ListenSocket:  fmt.Sprintf("/run/php/php%s-fpm-%s.sock", phpVersion, username),
		MaxChildren:   5,
		StartServers:  2,
		MinSpare:      1,
		MaxSpare:      3,
		MaxRequests:   500,
		MemoryLimit:   "256M",
		UploadMaxSize: "64M",
		Timezone:      "UTC",
	}
}

// validateDomain guards against path traversal in domain strings.
func validateDomain(domain string) error {
	if domain == "" || strings.Contains(domain, "/") || strings.Contains(domain, "..") {
		return fmt.Errorf("invalid domain: %q", domain)
	}
	return nil
}

// poolConfigPath returns the FPM pool config file path.
func poolConfigPath(version, domain string) string {
	return fmt.Sprintf("/etc/php/%s/fpm/pool.d/%s.conf", version, domain)
}

// CreatePool renders and writes a PHP-FPM pool config, then reloads FPM.
func (m *Manager) CreatePool(ctx context.Context, cfg PoolConfig) error {
	if err := validateVersion(cfg.PHPVersion); err != nil {
		return err
	}
	if err := validateDomain(cfg.SiteDomain); err != nil {
		return err
	}

	// Apply defaults for zero values
	if cfg.MaxChildren == 0 {
		cfg.MaxChildren = 5
	}
	if cfg.StartServers == 0 {
		cfg.StartServers = 2
	}
	if cfg.MinSpare == 0 {
		cfg.MinSpare = 1
	}
	if cfg.MaxSpare == 0 {
		cfg.MaxSpare = 3
	}
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 500
	}
	if cfg.MemoryLimit == "" {
		cfg.MemoryLimit = "256M"
	}
	if cfg.UploadMaxSize == "" {
		cfg.UploadMaxSize = "64M"
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}
	if cfg.ListenSocket == "" {
		cfg.ListenSocket = fmt.Sprintf("/run/php/php%s-fpm-%s.sock", cfg.PHPVersion, cfg.Username)
	}

	// Map to template data
	data := poolTemplateData{
		PoolName:      cfg.SiteDomain,
		User:          cfg.Username,
		SocketPath:    cfg.ListenSocket,
		MaxChildren:   cfg.MaxChildren,
		StartServers:  cfg.StartServers,
		MinSpare:      cfg.MinSpare,
		MaxSpare:      cfg.MaxSpare,
		MaxRequests:   cfg.MaxRequests,
		MemoryLimit:   cfg.MemoryLimit,
		UploadMaxSize: cfg.UploadMaxSize,
		Timezone:      cfg.Timezone,
	}

	rendered, err := m.tpl.Render("php-fpm-pool.conf.tmpl", data)
	if err != nil {
		return fmt.Errorf("render pool config: %w", err)
	}

	// Write pool config atomically
	path := poolConfigPath(cfg.PHPVersion, cfg.SiteDomain)
	if err := m.files.WriteAtomic(path, []byte(rendered), 0640); err != nil {
		return fmt.Errorf("write pool config: %w", err)
	}

	// Reload FPM to pick up the new pool
	return m.ReloadFPM(ctx, cfg.PHPVersion)
}

// DeletePool removes a pool config file and reloads FPM.
func (m *Manager) DeletePool(ctx context.Context, domain, version string) error {
	if err := validateVersion(version); err != nil {
		return err
	}
	if err := validateDomain(domain); err != nil {
		return err
	}

	path := poolConfigPath(version, domain)
	if err := m.files.Remove(path); err != nil {
		return fmt.Errorf("remove pool config: %w", err)
	}

	return m.ReloadFPM(ctx, version)
}

// SwitchVersion moves a site from one PHP version to another.
// Strategy: create new pool first, update Nginx, then delete old pool.
// This ensures zero downtime during the switch.
func (m *Manager) SwitchVersion(ctx context.Context, cfg PoolConfig, fromVersion string, nginxReloadFn func() error) error {
	if err := validateVersion(fromVersion); err != nil {
		return err
	}
	if err := validateVersion(cfg.PHPVersion); err != nil {
		return err
	}

	if fromVersion == cfg.PHPVersion {
		return fmt.Errorf("site already uses php%s", cfg.PHPVersion)
	}

	// 1. Create new pool with the target version
	if err := m.CreatePool(ctx, cfg); err != nil {
		return fmt.Errorf("create pool for php%s: %w", cfg.PHPVersion, err)
	}

	// 2. Reload Nginx to use new socket path (caller updates vhost config)
	if nginxReloadFn != nil {
		if err := nginxReloadFn(); err != nil {
			// Rollback: remove the new pool
			_ = m.DeletePool(ctx, cfg.SiteDomain, cfg.PHPVersion)
			return fmt.Errorf("reload nginx after switch: %w", err)
		}
	}

	// 3. Delete old pool (new one is already serving)
	if err := m.DeletePool(ctx, cfg.SiteDomain, fromVersion); err != nil {
		// Non-fatal: log but continue (new pool is working)
		return fmt.Errorf("warning: failed to delete old pool for php%s: %w", fromVersion, err)
	}

	return nil
}
