package php

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jhin1m/juiscript/internal/system"
	"github.com/jhin1m/juiscript/internal/template"
)

// CommonExtensions are installed with every PHP version.
var CommonExtensions = []string{
	"cli", "fpm", "common", "mysql", "xml", "mbstring",
	"curl", "zip", "gd", "bcmath", "intl", "readline",
	"opcache",
}

// OptionalExtensions may fail to install on some systems; skipped on error.
var OptionalExtensions = []string{
	"redis", "imagick",
}

// VersionInfo describes an installed PHP version and its FPM service status.
type VersionInfo struct {
	Version string // e.g. "8.3"
	Active  bool   // FPM service running
	Enabled bool   // FPM service enabled at boot
}

// Manager handles PHP version install/remove and FPM pool lifecycle.
type Manager struct {
	executor system.Executor
	files    system.FileManager
	tpl      *template.Engine
}

// NewManager creates a PHP manager.
func NewManager(exec system.Executor, files system.FileManager, tpl *template.Engine) *Manager {
	return &Manager{
		executor: exec,
		files:    files,
		tpl:      tpl,
	}
}

// EnsurePPA adds ondrej/php PPA if not already present.
func (m *Manager) EnsurePPA(ctx context.Context) error {
	// Check if PPA list file already exists
	matches, _ := filepath.Glob("/etc/apt/sources.list.d/*ondrej*php*")
	if len(matches) > 0 {
		return nil // PPA already added
	}

	_, err := m.executor.Run(ctx, "add-apt-repository", "ppa:ondrej/php", "-y")
	if err != nil {
		return fmt.Errorf("add ondrej/php PPA: %w", err)
	}

	// Update package lists after adding PPA
	_, err = m.executor.Run(ctx, "apt-get", "update", "-y")
	if err != nil {
		return fmt.Errorf("apt-get update after PPA: %w", err)
	}

	return nil
}

// InstallVersion installs a PHP version with common extensions.
// Ensures the PPA is present before installing.
func (m *Manager) InstallVersion(ctx context.Context, version string) error {
	if err := validateVersion(version); err != nil {
		return err
	}

	if err := m.EnsurePPA(ctx); err != nil {
		return err
	}

	// Build package list: php{ver}-{ext} for each common extension
	var packages []string
	for _, ext := range CommonExtensions {
		packages = append(packages, fmt.Sprintf("php%s-%s", version, ext))
	}

	// Install common extensions (must all succeed)
	args := append([]string{"install", "-y"}, packages...)
	installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, err := m.executor.Run(installCtx, "apt-get", args...)
	if err != nil {
		return fmt.Errorf("install php%s: %w", version, err)
	}

	// Try optional extensions (failures are not fatal)
	for _, ext := range OptionalExtensions {
		pkg := fmt.Sprintf("php%s-%s", version, ext)
		_, _ = m.executor.Run(ctx, "apt-get", "install", "-y", pkg)
	}

	// Enable and start the FPM service
	svc := fpmServiceName(version)
	_, _ = m.executor.Run(ctx, "systemctl", "enable", svc)
	_, err = m.executor.Run(ctx, "systemctl", "start", svc)
	if err != nil {
		return fmt.Errorf("start %s: %w", svc, err)
	}

	return nil
}

// RemoveVersion removes a PHP version. Returns error if any site still uses it.
func (m *Manager) RemoveVersion(ctx context.Context, version string, activeSites []string) error {
	if err := validateVersion(version); err != nil {
		return err
	}

	// Safety: refuse if sites still use this version
	if len(activeSites) > 0 {
		return fmt.Errorf("cannot remove php%s: still used by sites: %s",
			version, strings.Join(activeSites, ", "))
	}

	// Stop and disable the FPM service first
	svc := fpmServiceName(version)
	_, _ = m.executor.Run(ctx, "systemctl", "stop", svc)
	_, _ = m.executor.Run(ctx, "systemctl", "disable", svc)

	// Remove all packages for this version
	removeCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	_, err := m.executor.Run(removeCtx, "apt-get", "remove", "--purge", "-y",
		fmt.Sprintf("php%s-*", version))
	if err != nil {
		return fmt.Errorf("remove php%s: %w", version, err)
	}

	return nil
}

// ListVersions scans /etc/php/ to find installed PHP versions and their FPM status.
func (m *Manager) ListVersions(ctx context.Context) ([]VersionInfo, error) {
	entries, err := os.ReadDir("/etc/php")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no PHP installed
		}
		return nil, fmt.Errorf("read /etc/php: %w", err)
	}

	var versions []VersionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ver := entry.Name()
		// Only include directories that look like version numbers
		if !isVersionDir(ver) {
			continue
		}

		// Check if FPM is installed (pool.d dir exists)
		poolDir := fmt.Sprintf("/etc/php/%s/fpm/pool.d", ver)
		if !m.files.Exists(poolDir) {
			continue
		}

		info := VersionInfo{Version: ver}

		// Check service status
		svc := fpmServiceName(ver)
		output, err := m.executor.Run(ctx, "systemctl", "is-active", svc)
		info.Active = err == nil && strings.TrimSpace(output) == "active"

		output, err = m.executor.Run(ctx, "systemctl", "is-enabled", svc)
		info.Enabled = err == nil && strings.TrimSpace(output) == "enabled"

		versions = append(versions, info)
	}

	return versions, nil
}

// ReloadFPM reloads a specific PHP-FPM version's service.
func (m *Manager) ReloadFPM(ctx context.Context, version string) error {
	if err := validateVersion(version); err != nil {
		return err
	}

	svc := fpmServiceName(version)
	_, err := m.executor.Run(ctx, "systemctl", "reload", svc)
	if err != nil {
		return fmt.Errorf("reload %s: %w", svc, err)
	}
	return nil
}

// fpmServiceName returns the systemd service name for a PHP-FPM version.
func fpmServiceName(version string) string {
	return fmt.Sprintf("php%s-fpm", version)
}

// validateVersion checks that a PHP version string is safe (e.g. "8.3", "7.4").
func validateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("php version cannot be empty")
	}
	// Must match major.minor format (e.g. "8.3", "7.4")
	if !isVersionDir(version) {
		return fmt.Errorf("invalid php version: %q (expected format: X.Y)", version)
	}
	return nil
}

// isVersionDir checks if a directory name looks like a PHP version.
func isVersionDir(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
