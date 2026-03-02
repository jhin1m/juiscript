package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"github.com/jhin1m/juiscript/internal/system"
)

// ServiceName identifies a systemd service.
type ServiceName string

// Known LEMP stack services (whitelist for security).
const (
	ServiceNginx   ServiceName = "nginx"
	ServiceMariaDB ServiceName = "mariadb"
	ServiceRedis   ServiceName = "redis-server"
)

// PHPFPMService returns the service name for a PHP-FPM version.
// Example: PHPFPMService("8.3") -> "php8.3-fpm"
func PHPFPMService(version string) ServiceName {
	return ServiceName(fmt.Sprintf("php%s-fpm", version))
}

// allowedServices is the whitelist of services we can manage.
// Prevents arbitrary service names (security: no shell injection).
var allowedServices = map[ServiceName]bool{
	ServiceNginx:   true,
	ServiceMariaDB: true,
	ServiceRedis:   true,
}

// isAllowed checks if a service name is safe to operate on.
func isAllowed(name ServiceName) bool {
	if allowedServices[name] {
		return true
	}
	// PHP-FPM services are dynamic: must match "phpX.Y-fpm" exactly
	s := string(name)
	if !strings.HasPrefix(s, "php") || !strings.HasSuffix(s, "-fpm") {
		return false
	}
	// Extract and validate version segment between "php" and "-fpm"
	version := s[3 : len(s)-4] // strip "php" prefix and "-fpm" suffix
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return false
	}
	return isNumeric(parts[0]) && isNumeric(parts[1])
}

// Status holds detailed info about a service's current state.
type Status struct {
	Name     ServiceName
	Active   bool          // true if state == "active"
	State    string        // "active", "inactive", "failed"
	SubState string        // "running", "dead", "exited"
	PID      int     // main process ID (0 if not running)
	MemoryMB float64 // memory usage in MB (0 if unavailable)
}

// Manager wraps systemctl to control LEMP services.
type Manager struct {
	executor system.Executor
}

// NewManager creates a service manager.
func NewManager(exec system.Executor) *Manager {
	return &Manager{executor: exec}
}

// Start starts a service via systemctl.
func (m *Manager) Start(ctx context.Context, name ServiceName) error {
	return m.runAction(ctx, "start", name)
}

// Stop stops a service via systemctl.
func (m *Manager) Stop(ctx context.Context, name ServiceName) error {
	return m.runAction(ctx, "stop", name)
}

// Restart restarts a service via systemctl.
func (m *Manager) Restart(ctx context.Context, name ServiceName) error {
	return m.runAction(ctx, "restart", name)
}

// Reload sends a reload signal to the service (graceful config reload).
func (m *Manager) Reload(ctx context.Context, name ServiceName) error {
	return m.runAction(ctx, "reload", name)
}

// runAction executes a systemctl action on a validated service.
func (m *Manager) runAction(ctx context.Context, action string, name ServiceName) error {
	if !isAllowed(name) {
		return fmt.Errorf("service %q is not in the allowed list", name)
	}

	_, err := m.executor.Run(ctx, "systemctl", action, string(name))
	if err != nil {
		return fmt.Errorf("%s %s: %w", action, name, err)
	}
	return nil
}

// IsActive returns true if the service is currently running.
func (m *Manager) IsActive(ctx context.Context, name ServiceName) bool {
	if !isAllowed(name) {
		return false
	}
	out, err := m.executor.Run(ctx, "systemctl", "is-active", string(name))
	return err == nil && strings.TrimSpace(out) == "active"
}

// Status returns detailed status information for a service.
func (m *Manager) Status(ctx context.Context, name ServiceName) (*Status, error) {
	if !isAllowed(name) {
		return nil, fmt.Errorf("service %q is not in the allowed list", name)
	}

	// Query multiple properties in one call for efficiency
	props := "ActiveState,SubState,MainPID,MemoryCurrent"
	out, err := m.executor.Run(ctx, "systemctl", "show", string(name), "--property="+props)
	if err != nil {
		return nil, fmt.Errorf("status %s: %w", name, err)
	}

	return parseStatus(name, out), nil
}

// parseStatus parses systemctl show output into a Status struct.
func parseStatus(name ServiceName, raw string) *Status {
	s := &Status{Name: name}

	for _, line := range strings.Split(raw, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]

		switch key {
		case "ActiveState":
			s.State = val
			s.Active = val == "active"
		case "SubState":
			s.SubState = val
		case "MainPID":
			s.PID, _ = strconv.Atoi(val)
		case "MemoryCurrent":
			// MemoryCurrent is in bytes; convert to MB
			if bytes, err := strconv.ParseUint(val, 10, 64); err == nil {
				s.MemoryMB = float64(bytes) / (1024 * 1024)
			}
		}
	}

	return s
}

// ListAll returns status for all detected LEMP services.
// Scans for Nginx, MariaDB, Redis, and all installed PHP-FPM versions.
func (m *Manager) ListAll(ctx context.Context) ([]Status, error) {
	var results []Status

	// Static services
	for _, name := range []ServiceName{ServiceNginx, ServiceMariaDB, ServiceRedis} {
		st, err := m.Status(ctx, name)
		if err != nil {
			// Service may not be installed; report as inactive
			results = append(results, Status{Name: name, State: "not-found"})
			continue
		}
		results = append(results, *st)
	}

	// Dynamic: detect installed PHP-FPM versions from /etc/php/
	phpVersions := detectPHPVersions()
	for _, ver := range phpVersions {
		name := PHPFPMService(ver)
		st, err := m.Status(ctx, name)
		if err != nil {
			results = append(results, Status{Name: name, State: "not-found"})
			continue
		}
		results = append(results, *st)
	}

	return results, nil
}

// IsHealthy returns true if all critical services are active.
func (m *Manager) IsHealthy(ctx context.Context) bool {
	all, err := m.ListAll(ctx)
	if err != nil {
		return false
	}

	nginxOk := false
	mariadbOk := false
	phpOk := false

	for _, s := range all {
		if !s.Active {
			continue
		}
		switch {
		case s.Name == ServiceNginx:
			nginxOk = true
		case s.Name == ServiceMariaDB:
			mariadbOk = true
		case strings.HasPrefix(string(s.Name), "php") && strings.HasSuffix(string(s.Name), "-fpm"):
			phpOk = true
		}
	}

	return nginxOk && mariadbOk && phpOk
}

// detectPHPVersions scans /etc/php/ for installed PHP version directories.
func detectPHPVersions() []string {
	entries, err := os.ReadDir("/etc/php")
	if err != nil {
		return nil
	}

	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Validate version format (e.g. "8.3", "7.4")
		name := e.Name()
		parts := strings.Split(name, ".")
		if len(parts) == 2 && isNumeric(parts[0]) && isNumeric(parts[1]) {
			versions = append(versions, name)
		}
	}
	return versions
}

// isNumeric checks if a string contains only digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
