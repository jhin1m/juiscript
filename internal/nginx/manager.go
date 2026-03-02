package nginx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhin1m/juiscript/internal/system"
	"github.com/jhin1m/juiscript/internal/template"
)

// ProjectType identifies the framework (avoids importing site package).
type ProjectType string

const (
	ProjectLaravel   ProjectType = "laravel"
	ProjectWordPress ProjectType = "wordpress"
)

// VhostConfig holds all data needed to render an Nginx vhost template.
type VhostConfig struct {
	Domain      string
	WebRoot     string
	PHPSocket   string // e.g., /run/php/php8.3-fpm-{user}.sock
	AccessLog   string
	ErrorLog    string
	SSLEnabled  bool
	SSLCertPath string
	SSLKeyPath  string
	ProjectType ProjectType
	MaxBodySize string // default: "64m"
	ExtraConfig string // raw Nginx directives injected into server block
}

// VhostInfo summarizes a vhost for listing purposes.
type VhostInfo struct {
	Domain  string
	Enabled bool
	Path    string // full path in sites-available
}

// Manager handles Nginx vhost CRUD, config testing, and reload.
type Manager struct {
	executor       system.Executor
	files          system.FileManager
	tpl            *template.Engine
	sitesAvailable string // e.g., /etc/nginx/sites-available
	sitesEnabled   string // e.g., /etc/nginx/sites-enabled
}

// NewManager creates an Nginx manager.
func NewManager(
	exec system.Executor,
	files system.FileManager,
	tpl *template.Engine,
	sitesAvailable, sitesEnabled string,
) *Manager {
	return &Manager{
		executor:       exec,
		files:          files,
		tpl:            tpl,
		sitesAvailable: sitesAvailable,
		sitesEnabled:   sitesEnabled,
	}
}

// Create renders the appropriate template and writes to sites-available.
// It also enables the vhost, tests the config, and reloads Nginx.
// If the config test fails, it automatically rolls back.
func (m *Manager) Create(cfg VhostConfig) error {
	if err := validateDomain(cfg.Domain); err != nil {
		return err
	}
	if cfg.MaxBodySize == "" {
		cfg.MaxBodySize = "64m"
	}

	// Pick template based on project type
	tmplName, err := templateName(cfg.ProjectType)
	if err != nil {
		return err
	}

	rendered, err := m.tpl.Render(tmplName, cfg)
	if err != nil {
		return fmt.Errorf("render vhost template: %w", err)
	}

	// Write config to sites-available atomically
	availablePath := m.availablePath(cfg.Domain)
	if err := m.files.WriteAtomic(availablePath, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("write vhost config: %w", err)
	}

	// Enable the vhost (create symlink)
	if err := m.enable(cfg.Domain); err != nil {
		m.files.Remove(availablePath) // rollback: remove config
		return fmt.Errorf("enable vhost: %w", err)
	}

	// Test Nginx config before reload
	if err := m.Test(); err != nil {
		// Rollback: disable + remove config
		m.disable(cfg.Domain)
		m.files.Remove(availablePath)
		return fmt.Errorf("nginx config test failed (rolled back): %w", err)
	}

	// Safe to reload
	if err := m.Reload(); err != nil {
		return fmt.Errorf("reload nginx: %w", err)
	}

	return nil
}

// Delete removes a vhost config and reloads Nginx.
func (m *Manager) Delete(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	// Disable first (remove symlink)
	_ = m.disable(domain)

	// Remove the config file
	availablePath := m.availablePath(domain)
	if err := m.files.Remove(availablePath); err != nil {
		return fmt.Errorf("remove vhost config: %w", err)
	}

	// Reload to apply removal
	return m.Reload()
}

// Enable creates a symlink from sites-available to sites-enabled,
// tests the config, and reloads Nginx. Rolls back if test fails.
func (m *Manager) Enable(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	if err := m.enable(domain); err != nil {
		return err
	}

	if err := m.Test(); err != nil {
		m.disable(domain) // rollback
		return fmt.Errorf("nginx test failed after enable (rolled back): %w", err)
	}

	return m.Reload()
}

// Disable removes the symlink from sites-enabled and reloads Nginx.
func (m *Manager) Disable(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	if err := m.disable(domain); err != nil {
		return err
	}
	return m.Reload()
}

// Test runs `nginx -t` and returns a parsed error if the config is invalid.
func (m *Manager) Test() error {
	output, err := m.executor.Run(context.Background(), "nginx", "-t")
	if err != nil {
		return parseNginxTestError(output, err)
	}
	return nil
}

// Reload tells Nginx to reload its configuration.
func (m *Manager) Reload() error {
	_, err := m.executor.Run(context.Background(), "systemctl", "reload", "nginx")
	if err != nil {
		return fmt.Errorf("reload nginx: %w", err)
	}
	return nil
}

// List returns all vhosts in sites-available with their enabled status.
func (m *Manager) List() ([]VhostInfo, error) {
	entries, err := os.ReadDir(m.sitesAvailable)
	if err != nil {
		return nil, fmt.Errorf("read sites-available: %w", err)
	}

	var vhosts []VhostInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		domain := strings.TrimSuffix(name, ".conf")

		// Check if enabled (symlink exists in sites-enabled)
		enabledPath := m.enabledPath(domain)
		enabled := m.files.Exists(enabledPath)

		vhosts = append(vhosts, VhostInfo{
			Domain:  domain,
			Enabled: enabled,
			Path:    filepath.Join(m.sitesAvailable, name),
		})
	}

	return vhosts, nil
}

// enable creates a symlink (internal, no test/reload).
func (m *Manager) enable(domain string) error {
	available := m.availablePath(domain)
	enabled := m.enabledPath(domain)
	return m.files.Symlink(available, enabled)
}

// disable removes symlink (internal, no reload).
func (m *Manager) disable(domain string) error {
	enabled := m.enabledPath(domain)
	return m.files.RemoveSymlink(enabled)
}

// validateDomain guards against path traversal in domain strings.
func validateDomain(domain string) error {
	if strings.Contains(domain, "/") || strings.Contains(domain, "..") || domain == "" {
		return fmt.Errorf("invalid domain: %q", domain)
	}
	return nil
}

func (m *Manager) availablePath(domain string) string {
	return filepath.Join(m.sitesAvailable, domain+".conf")
}

func (m *Manager) enabledPath(domain string) string {
	return filepath.Join(m.sitesEnabled, domain+".conf")
}

// templateName maps project type to the correct template file.
func templateName(pt ProjectType) (string, error) {
	switch pt {
	case ProjectLaravel:
		return "nginx-laravel.conf.tmpl", nil
	case ProjectWordPress:
		return "nginx-wordpress.conf.tmpl", nil
	default:
		return "", fmt.Errorf("unsupported project type for nginx: %s", pt)
	}
}

// parseNginxTestError extracts useful info from `nginx -t` failure output.
func parseNginxTestError(output string, original error) error {
	// nginx -t writes errors to stderr, which is captured in the error output
	// Example: "nginx: [emerg] unknown directive "servr" in /etc/nginx/sites-enabled/example.conf:2"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "[emerg]") || strings.Contains(line, "[error]") {
			return fmt.Errorf("nginx config error: %s", line)
		}
	}
	return fmt.Errorf("nginx -t failed: %w", original)
}
