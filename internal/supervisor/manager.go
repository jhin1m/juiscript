package supervisor

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jhin1m/juiscript/internal/system"
	"github.com/jhin1m/juiscript/internal/template"
)

// Default config directory for Supervisor program files.
const defaultConfDir = "/etc/supervisor/conf.d"

// WorkerConfig holds all parameters needed to create a Supervisor queue worker.
type WorkerConfig struct {
	Domain     string // site domain (used as program name prefix)
	Username   string // Linux user the worker runs as
	SitePath   string // absolute path to the Laravel project root
	PHPBinary  string // e.g., /usr/bin/php8.3
	Connection string // queue connection: "redis", "database", "sqs" (default: "redis")
	Queue      string // queue name (default: "default")
	Processes  int    // number of worker processes (default: 1)
	Tries      int    // max retry attempts per job (default: 3)
	MaxTime    int    // max seconds a worker lives before restart (default: 3600)
	Sleep      int    // seconds to sleep when no jobs (default: 3)
}

// applyDefaults fills in zero-value fields with sensible defaults.
func (c *WorkerConfig) applyDefaults() {
	if c.Connection == "" {
		c.Connection = "redis"
	}
	if c.Queue == "" {
		c.Queue = "default"
	}
	if c.Processes <= 0 {
		c.Processes = 1
	}
	if c.Tries <= 0 {
		c.Tries = 3
	}
	if c.MaxTime <= 0 {
		c.MaxTime = 3600
	}
	if c.Sleep <= 0 {
		c.Sleep = 3
	}
}

// templateData converts WorkerConfig into the struct expected by the template.
func (c *WorkerConfig) templateData() map[string]any {
	return map[string]any{
		"ProgramName":  programName(c.Domain),
		"ArtisanPath":  filepath.Join(c.SitePath, "artisan"),
		"Connection":   c.Connection,
		"Queue":        c.Queue,
		"MaxTries":     c.Tries,
		"MaxTime":      c.MaxTime,
		"Sleep":        c.Sleep,
		"User":         c.Username,
		"NumProcs":     c.Processes,
		"PHPBinary":    c.PHPBinary,
		"StopWaitSecs": c.MaxTime + 60, // Allow MaxTime + 60s buffer for graceful shutdown
	}
}

// WorkerStatus represents the current state of a Supervisor program.
type WorkerStatus struct {
	Name   string        // program name (e.g., "example.com-worker")
	State  string        // RUNNING, STOPPED, FATAL, STARTING, etc.
	PID    int           // process ID (0 if not running)
	Uptime time.Duration // how long the process has been running
}

// Manager handles Supervisor queue worker lifecycle.
type Manager struct {
	executor system.Executor
	files    system.FileManager
	tpl      *template.Engine
	confDir  string // path to supervisor conf.d directory
}

// NewManager creates a Supervisor manager.
func NewManager(
	exec system.Executor,
	files system.FileManager,
	tpl *template.Engine,
) *Manager {
	return &Manager{
		executor: exec,
		files:    files,
		tpl:      tpl,
		confDir:  defaultConfDir,
	}
}

// NewManagerWithConfDir creates a Manager with a custom conf directory (for testing).
func NewManagerWithConfDir(
	exec system.Executor,
	files system.FileManager,
	tpl *template.Engine,
	confDir string,
) *Manager {
	return &Manager{
		executor: exec,
		files:    files,
		tpl:      tpl,
		confDir:  confDir,
	}
}

// Create renders the worker config template, writes it to conf.d, and reloads Supervisor.
// If reload fails, it rolls back by removing the config file.
func (m *Manager) Create(ctx context.Context, cfg WorkerConfig) error {
	if err := validateConfig(&cfg); err != nil {
		return err
	}
	cfg.applyDefaults()

	// Render the supervisor config from template
	rendered, err := m.tpl.Render("supervisor-worker.conf.tmpl", cfg.templateData())
	if err != nil {
		return fmt.Errorf("render supervisor template: %w", err)
	}

	// Write config atomically to conf.d
	confPath := m.confPath(cfg.Domain)
	if err := m.files.WriteAtomic(confPath, []byte(rendered), 0644); err != nil {
		return fmt.Errorf("write supervisor config: %w", err)
	}

	// Tell Supervisor to pick up the new config
	if err := m.reload(ctx); err != nil {
		m.files.Remove(confPath) // rollback: remove config on failure
		return fmt.Errorf("supervisor reload after create: %w", err)
	}

	return nil
}

// Delete removes a worker config and reloads Supervisor.
// Supervisor automatically stops programs whose configs are removed.
// No rollback on reload failure: re-running delete is idempotent and safe.
func (m *Manager) Delete(ctx context.Context, domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	confPath := m.confPath(domain)
	if err := m.files.Remove(confPath); err != nil {
		return fmt.Errorf("remove supervisor config: %w", err)
	}

	// Reread + update stops removed programs automatically
	if err := m.reload(ctx); err != nil {
		return fmt.Errorf("supervisor reload after delete: %w", err)
	}

	return nil
}

// Start starts all processes in the worker group.
func (m *Manager) Start(ctx context.Context, domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "supervisorctl", "start", programName(domain)+":*")
	if err != nil {
		return fmt.Errorf("start worker %s: %w", domain, err)
	}
	return nil
}

// Stop stops all processes in the worker group.
func (m *Manager) Stop(ctx context.Context, domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "supervisorctl", "stop", programName(domain)+":*")
	if err != nil {
		return fmt.Errorf("stop worker %s: %w", domain, err)
	}
	return nil
}

// Restart restarts all processes in the worker group.
func (m *Manager) Restart(ctx context.Context, domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	_, err := m.executor.Run(ctx, "supervisorctl", "restart", programName(domain)+":*")
	if err != nil {
		return fmt.Errorf("restart worker %s: %w", domain, err)
	}
	return nil
}

// Status returns the current status of the domain's worker.
func (m *Manager) Status(ctx context.Context, domain string) (*WorkerStatus, error) {
	if err := validateDomain(domain); err != nil {
		return nil, err
	}

	progName := programName(domain)
	out, err := m.executor.Run(ctx, "supervisorctl", "status", progName+":*")
	if err != nil {
		// supervisorctl returns exit code 3 for STOPPED/FATAL - still parse output
		if out == "" {
			return nil, fmt.Errorf("status worker %s: %w", domain, err)
		}
	}

	statuses := parseStatusOutput(out)
	if len(statuses) == 0 {
		return nil, fmt.Errorf("worker %s not found in supervisor", domain)
	}

	// Return first process as representative (multi-process workers share the same state)
	return &statuses[0], nil
}

// ListAll returns status for all Supervisor-managed workers.
func (m *Manager) ListAll(ctx context.Context) ([]WorkerStatus, error) {
	out, err := m.executor.Run(ctx, "supervisorctl", "status")
	if err != nil {
		// supervisorctl exits non-zero if any program is STOPPED/FATAL
		if out == "" {
			return nil, fmt.Errorf("list workers: %w", err)
		}
	}

	return parseStatusOutput(out), nil
}

// reload tells Supervisor to reread configs and apply changes.
// `reread` discovers new/changed/removed configs.
// `update` starts new, stops removed, and restarts changed programs.
func (m *Manager) reload(ctx context.Context) error {
	if _, err := m.executor.Run(ctx, "supervisorctl", "reread"); err != nil {
		return fmt.Errorf("supervisorctl reread: %w", err)
	}
	if _, err := m.executor.Run(ctx, "supervisorctl", "update"); err != nil {
		return fmt.Errorf("supervisorctl update: %w", err)
	}
	return nil
}

// confPath returns the config file path for a domain's worker.
func (m *Manager) confPath(domain string) string {
	return filepath.Join(m.confDir, domain+"-worker.conf")
}

// programName converts a domain to a Supervisor program name.
// Example: "example.com" -> "example.com-worker"
func programName(domain string) string {
	return domain + "-worker"
}

// validateConfig checks all required fields and constraints.
func validateConfig(cfg *WorkerConfig) error {
	if err := validateDomain(cfg.Domain); err != nil {
		return err
	}
	if cfg.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if cfg.SitePath == "" {
		return fmt.Errorf("site path cannot be empty")
	}
	if cfg.PHPBinary == "" {
		return fmt.Errorf("PHP binary path cannot be empty")
	}
	if cfg.Processes > 8 {
		return fmt.Errorf("processes cannot exceed 8 (got %d)", cfg.Processes)
	}
	return nil
}

// domainRegex validates domain format (alphanumeric, dots, hyphens).
var domainRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.\-]+[a-zA-Z0-9]$`)

// validateDomain guards against path traversal and invalid domain names.
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if strings.Contains(domain, "/") || strings.Contains(domain, "..") {
		return fmt.Errorf("invalid domain: %q", domain)
	}
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %q", domain)
	}
	return nil
}

// parseStatusOutput parses `supervisorctl status` output into WorkerStatus slices.
//
// Example output format:
//
//	example.com-worker:example.com-worker_00   RUNNING   pid 1234, uptime 0:05:30
//	example.com-worker:example.com-worker_01   STOPPED   Not started
//	myapp.com-worker:myapp.com-worker_00       FATAL     Exited too quickly
func parseStatusOutput(raw string) []WorkerStatus {
	var results []WorkerStatus

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		status := parseSingleStatus(line)
		if status != nil {
			results = append(results, *status)
		}
	}

	return results
}

// parseSingleStatus parses one line of supervisorctl status output.
func parseSingleStatus(line string) *WorkerStatus {
	// Split into fields by whitespace
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}

	ws := &WorkerStatus{
		Name:  fields[0],
		State: fields[1],
	}

	// Parse PID from "pid 1234," if present
	for i, f := range fields {
		if f == "pid" && i+1 < len(fields) {
			pidStr := strings.TrimSuffix(fields[i+1], ",")
			ws.PID, _ = strconv.Atoi(pidStr)
		}
		if f == "uptime" && i+1 < len(fields) {
			ws.Uptime = parseUptime(fields[i+1])
		}
	}

	return ws
}

// parseUptime converts "H:MM:SS" format to time.Duration.
func parseUptime(s string) time.Duration {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0
	}

	hours, _ := strconv.Atoi(parts[0])
	mins, _ := strconv.Atoi(parts[1])
	secs, _ := strconv.Atoi(parts[2])

	return time.Duration(hours)*time.Hour +
		time.Duration(mins)*time.Minute +
		time.Duration(secs)*time.Second
}
