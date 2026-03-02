package nginx

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
	// commands records all executed commands for assertion
	commands []string
	// failOn makes specific commands fail (key: command name)
	failOn map[string]error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{failOn: make(map[string]error)}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)
	if err, ok := m.failOn[name]; ok {
		return "", err
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

// mockFileManager simulates filesystem operations.
type mockFileManager struct {
	written  map[string][]byte // path -> content
	symlinks map[string]string // link -> target
	exists   map[string]bool
	failOn   map[string]error
}

func newMockFileManager() *mockFileManager {
	return &mockFileManager{
		written:  make(map[string][]byte),
		symlinks: make(map[string]string),
		exists:   make(map[string]bool),
		failOn:   make(map[string]error),
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

func (f *mockFileManager) Symlink(target, link string) error {
	if err, ok := f.failOn["symlink"]; ok {
		return err
	}
	f.symlinks[link] = target
	f.exists[link] = true
	return nil
}

func (f *mockFileManager) RemoveSymlink(path string) error {
	delete(f.symlinks, path)
	delete(f.exists, path)
	return nil
}

func (f *mockFileManager) Remove(path string) error {
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

// helper to create a Manager with mocks and real template engine
func setupTestManager(t *testing.T) (*Manager, *mockExecutor, *mockFileManager) {
	t.Helper()

	tpl, err := template.New()
	if err != nil {
		t.Fatalf("failed to create template engine: %v", err)
	}

	exec := newMockExecutor()
	files := newMockFileManager()

	mgr := NewManager(exec, files, tpl, "/etc/nginx/sites-available", "/etc/nginx/sites-enabled")
	return mgr, exec, files
}

func TestCreate_Laravel(t *testing.T) {
	mgr, exec, files := setupTestManager(t)

	cfg := VhostConfig{
		Domain:      "app.example.com",
		WebRoot:     "/home/site_app/app.example.com/public",
		PHPSocket:   "/run/php/php8.3-fpm-site_app.sock",
		AccessLog:   "/home/site_app/logs/nginx-access.log",
		ErrorLog:    "/home/site_app/logs/nginx-error.log",
		ProjectType: ProjectLaravel,
		MaxBodySize: "64m",
	}

	if err := mgr.Create(cfg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify config was written
	availPath := "/etc/nginx/sites-available/app.example.com.conf"
	content, ok := files.written[availPath]
	if !ok {
		t.Fatal("expected vhost config to be written")
	}

	// Check template rendered correctly
	contentStr := string(content)
	if !strings.Contains(contentStr, "server_name app.example.com") {
		t.Error("expected server_name in rendered config")
	}
	if !strings.Contains(contentStr, "try_files $uri $uri/ /index.php?$query_string") {
		t.Error("expected Laravel rewrite rule")
	}

	// Verify symlink was created
	enabledPath := "/etc/nginx/sites-enabled/app.example.com.conf"
	if _, ok := files.symlinks[enabledPath]; !ok {
		t.Error("expected symlink to be created")
	}

	// Verify nginx -t was called
	foundTest := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "nginx -t") {
			foundTest = true
			break
		}
	}
	if !foundTest {
		t.Error("expected nginx -t to be called")
	}
}

func TestCreate_WordPress(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	cfg := VhostConfig{
		Domain:      "blog.test.com",
		WebRoot:     "/home/site_blog/public_html/blog.test.com",
		PHPSocket:   "/run/php/php8.3-fpm-site_blog.sock",
		AccessLog:   "/home/site_blog/logs/nginx-access.log",
		ErrorLog:    "/home/site_blog/logs/nginx-error.log",
		ProjectType: ProjectWordPress,
		MaxBodySize: "128m",
	}

	if err := mgr.Create(cfg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	availPath := "/etc/nginx/sites-available/blog.test.com.conf"
	content := string(files.written[availPath])

	if !strings.Contains(content, "www.blog.test.com") {
		t.Error("expected www alias for WordPress")
	}
	if !strings.Contains(content, "try_files $uri $uri/ /index.php?$args") {
		t.Error("expected WordPress rewrite rule")
	}
	if !strings.Contains(content, "favicon.ico") {
		t.Error("expected WordPress static file rules")
	}
}

func TestCreate_DefaultMaxBodySize(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	cfg := VhostConfig{
		Domain:      "default.test.com",
		WebRoot:     "/var/www/html",
		PHPSocket:   "/run/php/php8.3-fpm.sock",
		AccessLog:   "/var/log/nginx/access.log",
		ErrorLog:    "/var/log/nginx/error.log",
		ProjectType: ProjectLaravel,
		// MaxBodySize intentionally empty - should default to 64m
	}

	if err := mgr.Create(cfg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	content := string(files.written["/etc/nginx/sites-available/default.test.com.conf"])
	if !strings.Contains(content, "client_max_body_size 64m") {
		t.Error("expected default max body size of 64m")
	}
}

func TestCreate_RollbackOnNginxTestFailure(t *testing.T) {
	mgr, exec, files := setupTestManager(t)

	// Make nginx -t fail
	exec.failOn["nginx"] = fmt.Errorf("config syntax error")

	cfg := VhostConfig{
		Domain:      "broken.test.com",
		WebRoot:     "/var/www/html",
		PHPSocket:   "/run/php/php8.3-fpm.sock",
		AccessLog:   "/var/log/access.log",
		ErrorLog:    "/var/log/error.log",
		ProjectType: ProjectLaravel,
	}

	err := mgr.Create(cfg)
	if err == nil {
		t.Fatal("expected Create to fail when nginx -t fails")
	}

	// Verify rollback: config should be removed
	availPath := "/etc/nginx/sites-available/broken.test.com.conf"
	if _, ok := files.written[availPath]; ok {
		t.Error("expected config to be removed after rollback")
	}

	// Verify rollback: symlink should be removed
	enabledPath := "/etc/nginx/sites-enabled/broken.test.com.conf"
	if _, ok := files.symlinks[enabledPath]; ok {
		t.Error("expected symlink to be removed after rollback")
	}
}

func TestDelete(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	// Pre-create a vhost
	availPath := "/etc/nginx/sites-available/remove.test.com.conf"
	enabledPath := "/etc/nginx/sites-enabled/remove.test.com.conf"
	files.written[availPath] = []byte("server {}")
	files.exists[availPath] = true
	files.symlinks[enabledPath] = availPath
	files.exists[enabledPath] = true

	if err := mgr.Delete("remove.test.com"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, ok := files.written[availPath]; ok {
		t.Error("expected config file to be removed")
	}
	if _, ok := files.symlinks[enabledPath]; ok {
		t.Error("expected symlink to be removed")
	}
}

func TestEnable(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	// Pre-create config file
	availPath := "/etc/nginx/sites-available/mysite.com.conf"
	files.written[availPath] = []byte("server {}")
	files.exists[availPath] = true

	if err := mgr.Enable("mysite.com"); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	enabledPath := "/etc/nginx/sites-enabled/mysite.com.conf"
	if _, ok := files.symlinks[enabledPath]; !ok {
		t.Error("expected symlink to exist after enable")
	}
}

func TestEnable_RollbackOnTestFailure(t *testing.T) {
	mgr, exec, files := setupTestManager(t)

	availPath := "/etc/nginx/sites-available/bad.com.conf"
	files.written[availPath] = []byte("server {}")
	files.exists[availPath] = true

	exec.failOn["nginx"] = fmt.Errorf("syntax error")

	err := mgr.Enable("bad.com")
	if err == nil {
		t.Fatal("expected Enable to fail when nginx -t fails")
	}

	// Symlink should be rolled back
	enabledPath := "/etc/nginx/sites-enabled/bad.com.conf"
	if _, ok := files.symlinks[enabledPath]; ok {
		t.Error("expected symlink to be removed after failed test")
	}
}

func TestDisable(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	enabledPath := "/etc/nginx/sites-enabled/off.com.conf"
	files.symlinks[enabledPath] = "/etc/nginx/sites-available/off.com.conf"
	files.exists[enabledPath] = true

	if err := mgr.Disable("off.com"); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	if _, ok := files.symlinks[enabledPath]; ok {
		t.Error("expected symlink to be removed after disable")
	}
}

func TestList(t *testing.T) {
	// Create real temp directories to test List with os.ReadDir
	tmpDir := t.TempDir()
	sitesAvailable := filepath.Join(tmpDir, "sites-available")
	sitesEnabled := filepath.Join(tmpDir, "sites-enabled")
	os.MkdirAll(sitesAvailable, 0755)
	os.MkdirAll(sitesEnabled, 0755)

	// Create config files
	os.WriteFile(filepath.Join(sitesAvailable, "one.com.conf"), []byte(""), 0644)
	os.WriteFile(filepath.Join(sitesAvailable, "two.com.conf"), []byte(""), 0644)

	// Enable one.com only
	files := newMockFileManager()
	files.exists[filepath.Join(sitesEnabled, "one.com.conf")] = true

	tpl, _ := template.New()
	exec := newMockExecutor()
	mgr := NewManager(exec, files, tpl, sitesAvailable, sitesEnabled)

	vhosts, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(vhosts) != 2 {
		t.Fatalf("expected 2 vhosts, got %d", len(vhosts))
	}

	// Find one.com and two.com
	var one, two *VhostInfo
	for i := range vhosts {
		switch vhosts[i].Domain {
		case "one.com":
			one = &vhosts[i]
		case "two.com":
			two = &vhosts[i]
		}
	}

	if one == nil || two == nil {
		t.Fatal("expected both vhosts to be listed")
	}
	if !one.Enabled {
		t.Error("expected one.com to be enabled")
	}
	if two.Enabled {
		t.Error("expected two.com to be disabled")
	}
}

func TestCreate_UnsupportedProjectType(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	cfg := VhostConfig{
		Domain:      "unknown.com",
		WebRoot:     "/var/www",
		PHPSocket:   "/run/php/php8.3-fpm.sock",
		AccessLog:   "/var/log/access.log",
		ErrorLog:    "/var/log/error.log",
		ProjectType: "django", // unsupported
	}

	err := mgr.Create(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported project type")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported error, got: %v", err)
	}
}

func TestParseNginxTestError(t *testing.T) {
	// Test with emerg error
	output := `nginx: [emerg] unknown directive "servr" in /etc/nginx/sites-enabled/test.conf:2
nginx: configuration file /etc/nginx/nginx.conf test failed`
	err := parseNginxTestError(output, fmt.Errorf("exit status 1"))

	if !strings.Contains(err.Error(), "[emerg]") {
		t.Errorf("expected parsed error with [emerg], got: %v", err)
	}

	// Test without recognizable pattern
	err2 := parseNginxTestError("", fmt.Errorf("exit status 1"))
	if !strings.Contains(err2.Error(), "nginx -t failed") {
		t.Errorf("expected fallback error, got: %v", err2)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	malicious := []string{"../../../etc/passwd", "foo/bar", ".."}
	for _, domain := range malicious {
		cfg := VhostConfig{
			Domain: domain, WebRoot: "/var/www", PHPSocket: "/tmp/php.sock",
			AccessLog: "/dev/null", ErrorLog: "/dev/null", ProjectType: ProjectLaravel,
		}
		if err := mgr.Create(cfg); err == nil {
			t.Errorf("expected error for domain %q", domain)
		}
		if err := mgr.Enable(domain); err == nil {
			t.Errorf("expected Enable error for domain %q", domain)
		}
		if err := mgr.Disable(domain); err == nil {
			t.Errorf("expected Disable error for domain %q", domain)
		}
		if err := mgr.Delete(domain); err == nil {
			t.Errorf("expected Delete error for domain %q", domain)
		}
	}
}

func TestCreate_ExtraConfig(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	cfg := VhostConfig{
		Domain:      "extra.test.com",
		WebRoot:     "/var/www/html",
		PHPSocket:   "/run/php/php8.3-fpm.sock",
		AccessLog:   "/var/log/access.log",
		ErrorLog:    "/var/log/error.log",
		ProjectType: ProjectLaravel,
		ExtraConfig: "    proxy_read_timeout 300s;",
	}

	if err := mgr.Create(cfg); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	content := string(files.written["/etc/nginx/sites-available/extra.test.com.conf"])
	if !strings.Contains(content, "proxy_read_timeout 300s") {
		t.Error("expected ExtraConfig to be included in rendered output")
	}
}
