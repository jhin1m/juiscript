package php

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jhin1m/juiscript/internal/template"
)

// mockExecutor simulates system command execution for testing.
type mockExecutor struct {
	commands []string
	failOn   map[string]error
	outputs  map[string]string
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		failOn:  make(map[string]error),
		outputs: make(map[string]string),
	}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)

	// Check for specific command output overrides
	if output, ok := m.outputs[cmd]; ok {
		return output, nil
	}

	if err, ok := m.failOn[name]; ok {
		return "", err
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

func (m *mockExecutor) hasCommand(substr string) bool {
	for _, cmd := range m.commands {
		if strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}

// mockFileManager simulates filesystem operations.
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

// helper to create Manager with mocks
func setupTestManager(t *testing.T) (*Manager, *mockExecutor, *mockFileManager) {
	t.Helper()

	tpl, err := template.New()
	if err != nil {
		t.Fatalf("failed to create template engine: %v", err)
	}

	exec := newMockExecutor()
	files := newMockFileManager()

	mgr := NewManager(exec, files, tpl)
	return mgr, exec, files
}

// --- Version validation tests ---

func TestValidateVersion(t *testing.T) {
	valid := []string{"8.3", "8.2", "7.4", "8.4"}
	for _, v := range valid {
		if err := validateVersion(v); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", v, err)
		}
	}

	invalid := []string{"", "abc", "8.3.1", "../etc", "8.3;rm -rf /", "."}
	for _, v := range invalid {
		if err := validateVersion(v); err == nil {
			t.Errorf("expected %q to be invalid", v)
		}
	}
}

func TestIsVersionDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"8.3", true},
		{"7.4", true},
		{"8.10", true},
		{"abc", false},
		{"8.3.1", false},
		{"", false},
		{"..", false},
		{".", false},
	}

	for _, tc := range tests {
		if got := isVersionDir(tc.name); got != tc.want {
			t.Errorf("isVersionDir(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// --- InstallVersion tests ---

func TestInstallVersion_Success(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.InstallVersion(ctx, "8.3"); err != nil {
		t.Fatalf("InstallVersion failed: %v", err)
	}

	// Should have called apt-get install with common extensions
	if !exec.hasCommand("apt-get install -y") {
		t.Error("expected apt-get install to be called")
	}

	if !exec.hasCommand("php8.3-fpm") {
		t.Error("expected php8.3-fpm package in install command")
	}

	// Should start the FPM service
	if !exec.hasCommand("systemctl start php8.3-fpm") {
		t.Error("expected FPM service to be started")
	}
}

func TestInstallVersion_InvalidVersion(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.InstallVersion(ctx, "../etc"); err == nil {
		t.Error("expected error for path traversal version")
	}
}

func TestInstallVersion_AptFailure(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	exec.failOn["apt-get"] = fmt.Errorf("package not found")

	err := mgr.InstallVersion(ctx, "8.3")
	if err == nil {
		t.Fatal("expected error when apt-get fails")
	}
}

// --- RemoveVersion tests ---

func TestRemoveVersion_Success(t *testing.T) {
	mgr, exec, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.RemoveVersion(ctx, "8.2", nil); err != nil {
		t.Fatalf("RemoveVersion failed: %v", err)
	}

	if !exec.hasCommand("systemctl stop php8.2-fpm") {
		t.Error("expected FPM service to be stopped")
	}

	if !exec.hasCommand("apt-get remove --purge -y php8.2-*") {
		t.Error("expected apt-get remove to be called")
	}
}

func TestRemoveVersion_RefusedWhenSitesActive(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	err := mgr.RemoveVersion(ctx, "8.3", []string{"example.com", "blog.com"})
	if err == nil {
		t.Fatal("expected error when sites are using this version")
	}

	if !strings.Contains(err.Error(), "example.com") {
		t.Errorf("error should mention active sites, got: %v", err)
	}
}

// --- ListVersions tests ---

func TestListVersions_ScansDirectory(t *testing.T) {
	// Create temp /etc/php-like directory
	tmpDir := t.TempDir()
	phpDir := filepath.Join(tmpDir, "etc", "php")

	// Create version dirs with FPM pool.d
	for _, ver := range []string{"8.2", "8.3"} {
		poolDir := filepath.Join(phpDir, ver, "fpm", "pool.d")
		os.MkdirAll(poolDir, 0755)
	}

	// Create a non-version dir (should be ignored)
	os.MkdirAll(filepath.Join(phpDir, "mods-available"), 0755)

	// We can't easily test ListVersions since it reads /etc/php directly.
	// Instead, test the helper function.
	if !isVersionDir("8.3") {
		t.Error("8.3 should be a valid version dir")
	}
	if isVersionDir("mods-available") {
		t.Error("mods-available should not be a valid version dir")
	}
}

// --- Pool tests ---

func TestCreatePool_Success(t *testing.T) {
	mgr, _, files := setupTestManager(t)
	ctx := context.Background()

	cfg := DefaultPool("example.com", "site_example", "8.3")

	if err := mgr.CreatePool(ctx, cfg); err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	// Check pool config was written
	poolPath := "/etc/php/8.3/fpm/pool.d/example.com.conf"
	content, ok := files.written[poolPath]
	if !ok {
		t.Fatal("expected pool config to be written")
	}

	contentStr := string(content)

	// Verify template rendered correctly
	if !strings.Contains(contentStr, "[example.com]") {
		t.Error("expected pool name in rendered config")
	}
	if !strings.Contains(contentStr, "user = site_example") {
		t.Error("expected user in rendered config")
	}
	if !strings.Contains(contentStr, "/run/php/php8.3-fpm-site_example.sock") {
		t.Error("expected socket path in rendered config")
	}
	if !strings.Contains(contentStr, "memory_limit] = 256M") {
		t.Error("expected memory_limit in rendered config")
	}
	if !strings.Contains(contentStr, "upload_max_filesize] = 64M") {
		t.Error("expected upload_max_filesize in rendered config")
	}
	if !strings.Contains(contentStr, "date.timezone] = UTC") {
		t.Error("expected timezone in rendered config")
	}
	if !strings.Contains(contentStr, "security.limit_extensions = .php") {
		t.Error("expected security.limit_extensions in rendered config")
	}
}

func TestCreatePool_DefaultValues(t *testing.T) {
	mgr, _, files := setupTestManager(t)
	ctx := context.Background()

	// Create with zero/empty values to test defaults
	cfg := PoolConfig{
		SiteDomain: "test.com",
		Username:   "site_test",
		PHPVersion: "8.3",
	}

	if err := mgr.CreatePool(ctx, cfg); err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	content := string(files.written["/etc/php/8.3/fpm/pool.d/test.com.conf"])

	if !strings.Contains(content, "pm.max_children = 5") {
		t.Error("expected default max_children = 5")
	}
	if !strings.Contains(content, "pm.max_requests = 500") {
		t.Error("expected default max_requests = 500")
	}
}

func TestCreatePool_InvalidVersion(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	cfg := DefaultPool("example.com", "user", "invalid!")
	if err := mgr.CreatePool(ctx, cfg); err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestDeletePool_Success(t *testing.T) {
	mgr, _, files := setupTestManager(t)
	ctx := context.Background()

	// Pre-create pool config
	poolPath := "/etc/php/8.3/fpm/pool.d/example.com.conf"
	files.written[poolPath] = []byte("pool config")
	files.exists[poolPath] = true

	if err := mgr.DeletePool(ctx, "example.com", "8.3"); err != nil {
		t.Fatalf("DeletePool failed: %v", err)
	}

	if _, ok := files.written[poolPath]; ok {
		t.Error("expected pool config to be removed")
	}
}

func TestSwitchVersion_Success(t *testing.T) {
	mgr, _, files := setupTestManager(t)
	ctx := context.Background()

	cfg := DefaultPool("example.com", "site_example", "8.3")

	// Pre-create old pool config
	oldPoolPath := "/etc/php/8.2/fpm/pool.d/example.com.conf"
	files.written[oldPoolPath] = []byte("old pool")
	files.exists[oldPoolPath] = true

	nginxReloaded := false
	nginxReloadFn := func() error {
		nginxReloaded = true
		return nil
	}

	if err := mgr.SwitchVersion(ctx, cfg, "8.2", nginxReloadFn); err != nil {
		t.Fatalf("SwitchVersion failed: %v", err)
	}

	// New pool should be created
	newPoolPath := "/etc/php/8.3/fpm/pool.d/example.com.conf"
	if _, ok := files.written[newPoolPath]; !ok {
		t.Error("expected new pool config to be written")
	}

	// Nginx should have been reloaded
	if !nginxReloaded {
		t.Error("expected nginx reload to be called")
	}

	// Old pool should be removed
	if _, ok := files.written[oldPoolPath]; ok {
		t.Error("expected old pool config to be removed")
	}
}

func TestSwitchVersion_SameVersion(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	cfg := DefaultPool("example.com", "user", "8.3")

	err := mgr.SwitchVersion(ctx, cfg, "8.3", nil)
	if err == nil {
		t.Error("expected error when switching to same version")
	}
}

func TestSwitchVersion_RollbackOnNginxFailure(t *testing.T) {
	mgr, _, files := setupTestManager(t)
	ctx := context.Background()

	cfg := DefaultPool("example.com", "site_example", "8.3")

	nginxReloadFn := func() error {
		return fmt.Errorf("nginx config test failed")
	}

	err := mgr.SwitchVersion(ctx, cfg, "8.2", nginxReloadFn)
	if err == nil {
		t.Fatal("expected error when nginx reload fails")
	}

	// New pool should be rolled back (deleted)
	newPoolPath := "/etc/php/8.3/fpm/pool.d/example.com.conf"
	if _, ok := files.written[newPoolPath]; ok {
		t.Error("expected new pool to be rolled back after nginx failure")
	}
}

func TestFpmServiceName(t *testing.T) {
	if got := fpmServiceName("8.3"); got != "php8.3-fpm" {
		t.Errorf("fpmServiceName(8.3) = %q, want php8.3-fpm", got)
	}
}

func TestCreatePool_PathTraversalDomain(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	malicious := []string{"../../../etc/cron.d/evil", "foo/bar", ".."}
	for _, domain := range malicious {
		cfg := DefaultPool(domain, "user", "8.3")
		if err := mgr.CreatePool(ctx, cfg); err == nil {
			t.Errorf("expected error for domain %q", domain)
		}
	}
}

func TestDeletePool_PathTraversalDomain(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	if err := mgr.DeletePool(ctx, "../../../etc/passwd", "8.3"); err == nil {
		t.Error("expected error for path traversal domain")
	}
}

func TestDefaultPool(t *testing.T) {
	cfg := DefaultPool("example.com", "site_example", "8.3")

	if cfg.SiteDomain != "example.com" {
		t.Errorf("expected domain example.com, got %s", cfg.SiteDomain)
	}
	if cfg.ListenSocket != "/run/php/php8.3-fpm-site_example.sock" {
		t.Errorf("unexpected socket path: %s", cfg.ListenSocket)
	}
	if cfg.MaxChildren != 5 {
		t.Errorf("expected MaxChildren=5, got %d", cfg.MaxChildren)
	}
	if cfg.MemoryLimit != "256M" {
		t.Errorf("expected MemoryLimit=256M, got %s", cfg.MemoryLimit)
	}
}
