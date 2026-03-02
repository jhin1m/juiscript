package supervisor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jhin1m/juiscript/internal/template"
)

// --- Mock implementations ---

type mockExecutor struct {
	commands []string
	outputs  map[string]string
	failOn   map[string]error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		outputs: make(map[string]string),
		failOn:  make(map[string]error),
	}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)

	// Check failOn by command name
	if err, ok := m.failOn[name]; ok {
		return m.outputs[cmd], err
	}
	// Check failOn by full command
	if err, ok := m.failOn[cmd]; ok {
		return m.outputs[cmd], err
	}

	if out, ok := m.outputs[cmd]; ok {
		return out, nil
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

type mockFileManager struct {
	written map[string][]byte
	exists  map[string]bool
	failOn  map[string]error
}

func newMockFileManager() *mockFileManager {
	return &mockFileManager{
		written: make(map[string][]byte),
		exists:  make(map[string]bool),
		failOn:  make(map[string]error),
	}
}

func (f *mockFileManager) WriteAtomic(path string, data []byte, _ os.FileMode) error {
	if err, ok := f.failOn["write"]; ok {
		return err
	}
	f.written[path] = data
	f.exists[path] = true
	return nil
}

func (f *mockFileManager) Symlink(target, link string) error { return nil }
func (f *mockFileManager) RemoveSymlink(path string) error   { return nil }

func (f *mockFileManager) Remove(path string) error {
	if err, ok := f.failOn["remove"]; ok {
		return err
	}
	delete(f.written, path)
	delete(f.exists, path)
	return nil
}

func (f *mockFileManager) Exists(path string) bool {
	return f.exists[path]
}

func (f *mockFileManager) ReadFile(path string) ([]byte, error) {
	if data, ok := f.written[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// --- Test helpers ---

func setupTestManager(t *testing.T) (*Manager, *mockExecutor, *mockFileManager) {
	t.Helper()

	tpl, err := template.New()
	if err != nil {
		t.Fatalf("failed to create template engine: %v", err)
	}

	exec := newMockExecutor()
	files := newMockFileManager()
	mgr := NewManagerWithConfDir(exec, files, tpl, "/etc/supervisor/conf.d")

	return mgr, exec, files
}

func validConfig() WorkerConfig {
	return WorkerConfig{
		Domain:    "app.example.com",
		Username:  "site_app",
		SitePath:  "/home/site_app/app.example.com",
		PHPBinary: "/usr/bin/php8.3",
	}
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	mgr, exec, files := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.Create(ctx, validConfig()); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify config was written
	confPath := "/etc/supervisor/conf.d/app.example.com-worker.conf"
	content, ok := files.written[confPath]
	if !ok {
		t.Fatal("expected supervisor config to be written")
	}

	// Verify template rendered with correct values
	contentStr := string(content)
	checks := []string{
		"[program:app.example.com-worker]",
		"/usr/bin/php8.3",
		"queue:work redis",
		"--queue=default",
		"--sleep=3",
		"--tries=3",
		"--max-time=3600",
		"user=site_app",
		"numprocs=1",
		"stopasgroup=true",
		"killasgroup=true",
		"stopwaitsecs=3660", // MaxTime(3600) + 60s buffer
	}
	for _, check := range checks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("expected config to contain %q", check)
		}
	}

	// Verify supervisorctl reread && update were called
	foundReread := false
	foundUpdate := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "reread") {
			foundReread = true
		}
		if strings.Contains(cmd, "update") {
			foundUpdate = true
		}
	}
	if !foundReread || !foundUpdate {
		t.Error("expected supervisorctl reread and update to be called")
	}
}

func TestCreate_CustomParams(t *testing.T) {
	mgr, _, files := setupTestManager(t)
	ctx := context.Background()

	cfg := validConfig()
	cfg.Connection = "database"
	cfg.Queue = "emails"
	cfg.Processes = 4
	cfg.Tries = 5
	cfg.MaxTime = 1800
	cfg.Sleep = 5

	if err := mgr.Create(ctx, cfg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	confPath := "/etc/supervisor/conf.d/app.example.com-worker.conf"
	content := string(files.written[confPath])

	checks := []string{
		"queue:work database",
		"--queue=emails",
		"--sleep=5",
		"--tries=5",
		"--max-time=1800",
		"numprocs=4",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected config to contain %q", check)
		}
	}
}

func TestCreate_RollbackOnReloadFailure(t *testing.T) {
	mgr, exec, files := setupTestManager(t)
	ctx := context.Background()

	// Make supervisorctl reread fail
	exec.failOn["supervisorctl"] = fmt.Errorf("supervisor not running")

	err := mgr.Create(ctx, validConfig())
	if err == nil {
		t.Fatal("expected Create to fail when reload fails")
	}

	// Config should be rolled back (removed)
	confPath := "/etc/supervisor/conf.d/app.example.com-worker.conf"
	if _, ok := files.written[confPath]; ok {
		t.Error("expected config to be removed after rollback")
	}
}

func TestCreate_ValidationErrors(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	tests := []struct {
		name string
		cfg  WorkerConfig
	}{
		{"empty domain", WorkerConfig{Username: "u", SitePath: "/p", PHPBinary: "/php"}},
		{"empty username", WorkerConfig{Domain: "x.com", SitePath: "/p", PHPBinary: "/php"}},
		{"empty site path", WorkerConfig{Domain: "x.com", Username: "u", PHPBinary: "/php"}},
		{"empty php binary", WorkerConfig{Domain: "x.com", Username: "u", SitePath: "/p"}},
		{"too many processes", WorkerConfig{Domain: "x.com", Username: "u", SitePath: "/p", PHPBinary: "/php", Processes: 9}},
		{"path traversal", WorkerConfig{Domain: "../etc", Username: "u", SitePath: "/p", PHPBinary: "/php"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mgr.Create(ctx, tt.cfg); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestDelete_Success(t *testing.T) {
	mgr, exec, files := setupTestManager(t)
	ctx := context.Background()

	// Pre-create config file
	confPath := "/etc/supervisor/conf.d/app.example.com-worker.conf"
	files.written[confPath] = []byte("[program:app.example.com-worker]")
	files.exists[confPath] = true

	if err := mgr.Delete(ctx, "app.example.com"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Config should be removed
	if _, ok := files.written[confPath]; ok {
		t.Error("expected config to be removed")
	}

	// Verify reread+update were called
	foundReread := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "reread") {
			foundReread = true
		}
	}
	if !foundReread {
		t.Error("expected supervisorctl reread to be called after delete")
	}
}

func TestStart(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.Start(ctx, "app.example.com"); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	found := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "start app.example.com-worker:*") {
			found = true
		}
	}
	if !found {
		t.Error("expected supervisorctl start command")
	}
}

func TestStop(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.Stop(ctx, "app.example.com"); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	found := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "stop app.example.com-worker:*") {
			found = true
		}
	}
	if !found {
		t.Error("expected supervisorctl stop command")
	}
}

func TestRestart(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.Restart(ctx, "app.example.com"); err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	found := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "restart app.example.com-worker:*") {
			found = true
		}
	}
	if !found {
		t.Error("expected supervisorctl restart command")
	}
}

func TestStatus_Running(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	exec.outputs["supervisorctl status app.example.com-worker:*"] =
		"app.example.com-worker:app.example.com-worker_00   RUNNING   pid 1234, uptime 1:30:45"

	status, err := mgr.Status(ctx, "app.example.com")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status.State != "RUNNING" {
		t.Errorf("expected RUNNING, got %s", status.State)
	}
	if status.PID != 1234 {
		t.Errorf("expected PID 1234, got %d", status.PID)
	}
	if status.Uptime != 1*time.Hour+30*time.Minute+45*time.Second {
		t.Errorf("expected 1h30m45s uptime, got %v", status.Uptime)
	}
}

func TestStatus_Stopped(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	// supervisorctl returns exit 3 for non-running programs but still outputs text
	exec.outputs["supervisorctl status app.example.com-worker:*"] =
		"app.example.com-worker:app.example.com-worker_00   STOPPED   Not started"
	exec.failOn["supervisorctl status app.example.com-worker:*"] = fmt.Errorf("exit status 3")

	status, err := mgr.Status(ctx, "app.example.com")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status.State != "STOPPED" {
		t.Errorf("expected STOPPED, got %s", status.State)
	}
}

func TestListAll(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	exec.outputs["supervisorctl status"] = `app.example.com-worker:app.example.com-worker_00   RUNNING   pid 1234, uptime 0:05:30
mysite.com-worker:mysite.com-worker_00   FATAL     Exited too quickly`

	statuses, err := mgr.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 workers, got %d", len(statuses))
	}

	if statuses[0].State != "RUNNING" {
		t.Errorf("expected first worker RUNNING, got %s", statuses[0].State)
	}
	if statuses[1].State != "FATAL" {
		t.Errorf("expected second worker FATAL, got %s", statuses[1].State)
	}
}

func TestParseUptime(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"0:05:30", 5*time.Minute + 30*time.Second},
		{"1:00:00", 1 * time.Hour},
		{"24:30:15", 24*time.Hour + 30*time.Minute + 15*time.Second},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseUptime(tt.input)
			if got != tt.want {
				t.Errorf("parseUptime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateDomain(t *testing.T) {
	valid := []string{"example.com", "my-site.example.com", "a1.test.io"}
	for _, d := range valid {
		if err := validateDomain(d); err != nil {
			t.Errorf("expected %q to be valid, got: %v", d, err)
		}
	}

	invalid := []string{"", "../etc", "foo/bar", "..", "a", "-bad.com"}
	for _, d := range invalid {
		if err := validateDomain(d); err == nil {
			t.Errorf("expected %q to be invalid", d)
		}
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &WorkerConfig{}
	cfg.applyDefaults()

	if cfg.Connection != "redis" {
		t.Errorf("expected default connection redis, got %s", cfg.Connection)
	}
	if cfg.Queue != "default" {
		t.Errorf("expected default queue, got %s", cfg.Queue)
	}
	if cfg.Processes != 1 {
		t.Errorf("expected 1 process, got %d", cfg.Processes)
	}
	if cfg.Tries != 3 {
		t.Errorf("expected 3 tries, got %d", cfg.Tries)
	}
	if cfg.MaxTime != 3600 {
		t.Errorf("expected 3600 max time, got %d", cfg.MaxTime)
	}
	if cfg.Sleep != 3 {
		t.Errorf("expected 3 sleep, got %d", cfg.Sleep)
	}
}

func TestProgramName(t *testing.T) {
	if got := programName("example.com"); got != "example.com-worker" {
		t.Errorf("expected example.com-worker, got %s", got)
	}
}
