package site

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/template"
)

// --- Mock Infrastructure (follows nginx/manager_test.go pattern) ---

// mockExecutor records commands and can inject failures.
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
	if err, ok := m.failOn[name]; ok {
		return "", err
	}
	if out, ok := m.outputs[name]; ok {
		return out, nil
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

// mockFileManager tracks written files, symlinks, and existence.
type mockFileManager struct {
	written  map[string][]byte
	symlinks map[string]string
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

// mockUserManager simulates Linux user operations.
type mockUserManager struct {
	users    map[string]string // username -> homeDir
	failOn   map[string]error
	uid, gid int
}

func newMockUserManager() *mockUserManager {
	return &mockUserManager{
		users:  make(map[string]string),
		failOn: make(map[string]error),
		uid:    1001,
		gid:    1001,
	}
}

func (u *mockUserManager) Create(username, homeDir string) error {
	if err, ok := u.failOn["create"]; ok {
		return err
	}
	u.users[username] = homeDir
	return nil
}

func (u *mockUserManager) Delete(username string) error {
	if err, ok := u.failOn["delete"]; ok {
		return err
	}
	delete(u.users, username)
	return nil
}

func (u *mockUserManager) Exists(username string) bool {
	_, ok := u.users[username]
	return ok
}

func (u *mockUserManager) LookupUID(username string) (int, int, error) {
	if _, ok := u.users[username]; !ok {
		return 0, 0, fmt.Errorf("user not found: %s", username)
	}
	return u.uid, u.gid, nil
}

// setupTestSiteManager creates a Manager wired to mocks with temp dirs.
func setupTestSiteManager(t *testing.T) (*Manager, *mockExecutor, *mockFileManager, *mockUserManager) {
	t.Helper()

	tpl, err := template.New()
	if err != nil {
		t.Fatalf("template: %v", err)
	}

	exec := newMockExecutor()
	files := newMockFileManager()
	users := newMockUserManager()

	tmpDir := t.TempDir()
	sitesDir := t.TempDir()

	cfg := &config.Config{
		General: config.GeneralConfig{
			SitesRoot: tmpDir,
			SitesDir:  sitesDir, // override metadata path for tests
		},
		Nginx: config.NginxConfig{
			SitesAvailable: "/etc/nginx/sites-available",
			SitesEnabled:   "/etc/nginx/sites-enabled",
		},
	}

	mgr := NewManager(cfg, exec, files, users, tpl)
	return mgr, exec, files, users
}

// --- Create: Happy Path ---

func TestCreate_Laravel_Success(t *testing.T) {
	mgr, exec, files, users := setupTestSiteManager(t)

	site, err := mgr.Create(CreateOptions{
		Domain:      "app.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify user was created
	username := DeriveUsername("app.example.com")
	if _, ok := users.users[username]; !ok {
		t.Error("expected user to be created")
	}

	// Verify directories were created via real FS (t.TempDir)
	sitesRoot := mgr.config.General.SitesRoot
	homeDir := site.HomeDir(sitesRoot)
	for _, dir := range []string{
		site.SiteDir(sitesRoot),
		homeDir + "/logs",
		homeDir + "/tmp",
		site.SiteDir(sitesRoot) + "/public",
		site.SiteDir(sitesRoot) + "/storage",
		site.SiteDir(sitesRoot) + "/bootstrap/cache",
	} {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected dir %s to exist", dir)
		}
	}

	// Verify FPM pool config was written
	fpmPath := site.FPMPoolConfigPath()
	if _, ok := files.written[fpmPath]; !ok {
		t.Error("expected FPM pool config to be written")
	}

	// Verify nginx vhost was created (symlink exists)
	enabledPath := site.NginxEnabledPath(mgr.config.Nginx.SitesEnabled)
	if _, ok := files.symlinks[enabledPath]; !ok {
		t.Error("expected nginx symlink to exist")
	}

	// Verify chown and php-fpm reload were called
	foundChown := false
	foundReload := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "chown") {
			foundChown = true
		}
		if strings.Contains(cmd, "systemctl reload php8.3-fpm") {
			foundReload = true
		}
	}
	if !foundChown {
		t.Error("expected chown to be called")
	}
	if !foundReload {
		t.Error("expected php-fpm reload to be called")
	}

	// Verify metadata was saved
	metaPath := site.MetadataPath(mgr.config.SitesPath())
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Error("expected metadata file to be saved")
	}

	// Verify site fields
	if site.Domain != "app.example.com" {
		t.Errorf("Domain = %q", site.Domain)
	}
	if site.ProjectType != ProjectLaravel {
		t.Errorf("ProjectType = %q", site.ProjectType)
	}
	if !site.Enabled {
		t.Error("expected site to be enabled")
	}
	if !strings.HasSuffix(site.WebRoot, "/public") {
		t.Errorf("WebRoot should end with /public, got %q", site.WebRoot)
	}
}

func TestCreate_WordPress_Success(t *testing.T) {
	mgr, _, files, _ := setupTestSiteManager(t)

	site, err := mgr.Create(CreateOptions{
		Domain:      "blog.example.com",
		ProjectType: ProjectWordPress,
		PHPVersion:  "8.3",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// WordPress WebRoot should NOT end with /public
	if strings.HasSuffix(site.WebRoot, "/public") {
		t.Error("WordPress WebRoot should not end with /public")
	}
	if !strings.Contains(site.WebRoot, "public_html") {
		t.Error("WordPress WebRoot should contain public_html")
	}

	// Verify nginx vhost was written
	availPath := site.NginxConfigPath(mgr.config.Nginx.SitesAvailable)
	content, ok := files.written[availPath]
	if !ok {
		t.Fatal("expected vhost config to be written")
	}
	if !strings.Contains(string(content), "www.blog.example.com") {
		t.Error("expected www alias in WordPress vhost")
	}
}

// --- Create: Validation Errors ---

func TestCreate_InvalidDomain(t *testing.T) {
	mgr, _, _, _ := setupTestSiteManager(t)

	tests := []struct {
		name   string
		domain string
	}{
		{"empty", ""},
		{"no TLD", "localhost"},
		{"starts with hyphen", "-bad.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.Create(CreateOptions{
				Domain:      tt.domain,
				ProjectType: ProjectLaravel,
				PHPVersion:  "8.3",
			})
			if err == nil {
				t.Errorf("expected error for domain %q", tt.domain)
			}
		})
	}
}

func TestCreate_InvalidProjectType(t *testing.T) {
	mgr, _, _, _ := setupTestSiteManager(t)

	_, err := mgr.Create(CreateOptions{
		Domain:      "test.example.com",
		ProjectType: "flask",
		PHPVersion:  "8.3",
	})
	if err == nil {
		t.Fatal("expected error for invalid project type")
	}
}

func TestCreate_InvalidPHPVersion(t *testing.T) {
	mgr, _, _, _ := setupTestSiteManager(t)

	tests := []string{"abc", "", "83"}
	for _, ver := range tests {
		t.Run(ver, func(t *testing.T) {
			_, err := mgr.Create(CreateOptions{
				Domain:      "test.example.com",
				ProjectType: ProjectLaravel,
				PHPVersion:  ver,
			})
			if err == nil {
				t.Errorf("expected error for PHP version %q", ver)
			}
		})
	}
}

func TestCreate_UserAlreadyExists(t *testing.T) {
	mgr, _, _, users := setupTestSiteManager(t)

	username := DeriveUsername("test.example.com")
	users.users[username] = "/home/" + username

	_, err := mgr.Create(CreateOptions{
		Domain:      "test.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err == nil {
		t.Fatal("expected error when user already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

// --- Create: Rollback Scenarios ---

func TestCreate_RollbackOnUserCreateFail(t *testing.T) {
	mgr, _, files, users := setupTestSiteManager(t)

	users.failOn["create"] = fmt.Errorf("useradd failed")

	_, err := mgr.Create(CreateOptions{
		Domain:      "fail.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	// No files should be written, no users created
	if len(files.written) > 0 {
		t.Error("no files should be written after user create failure")
	}
	if len(users.users) > 0 {
		t.Error("no users should exist after failure")
	}
}

func TestCreate_RollbackOnFPMPoolFail(t *testing.T) {
	mgr, _, files, users := setupTestSiteManager(t)

	files.failOn["write"] = fmt.Errorf("disk full")

	_, err := mgr.Create(CreateOptions{
		Domain:      "fail.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	// User should be rolled back (deleted)
	username := DeriveUsername("fail.example.com")
	if _, ok := users.users[username]; ok {
		t.Error("user should be deleted during rollback")
	}
}

func TestCreate_RollbackOnNginxFail(t *testing.T) {
	mgr, exec, _, users := setupTestSiteManager(t)

	// nginx -t will fail, causing nginx.Create to fail
	exec.failOn["nginx"] = fmt.Errorf("config syntax error")

	_, err := mgr.Create(CreateOptions{
		Domain:      "fail.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	// User should be rolled back
	username := DeriveUsername("fail.example.com")
	if _, ok := users.users[username]; ok {
		t.Error("user should be deleted during rollback")
	}
}

func TestCreate_RollbackOnPHPReloadFail(t *testing.T) {
	mgr, exec, _, users := setupTestSiteManager(t)

	exec.failOn["systemctl"] = fmt.Errorf("service not found")

	_, err := mgr.Create(CreateOptions{
		Domain:      "fail.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	// User should be rolled back
	username := DeriveUsername("fail.example.com")
	if _, ok := users.users[username]; ok {
		t.Error("user should be deleted during rollback")
	}
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	mgr, _, _, users := setupTestSiteManager(t)

	// First create a site (saves metadata to temp dir)
	site, err := mgr.Create(CreateOptions{
		Domain:      "delete.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	username := site.User

	// Pre-add nginx config files so nginx.Delete works
	err = mgr.Delete("delete.example.com", false)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// User should be deleted
	if _, ok := users.users[username]; ok {
		t.Error("user should be deleted")
	}

	// Metadata should be removed
	metaPath := site.MetadataPath(mgr.config.SitesPath())
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Error("metadata file should be removed")
	}
}

func TestDelete_SiteNotFound(t *testing.T) {
	mgr, _, _, _ := setupTestSiteManager(t)

	err := mgr.Delete("nonexistent.example.com", false)
	if err == nil {
		t.Fatal("expected error when site not found")
	}
}

// --- Enable / Disable ---

func TestEnable_Success(t *testing.T) {
	mgr, _, files, _ := setupTestSiteManager(t)

	// Create site first
	site, err := mgr.Create(CreateOptions{
		Domain:      "enable.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	// Disable it first (remove symlink)
	availPath := site.NginxConfigPath(mgr.config.Nginx.SitesAvailable)
	files.exists[availPath] = true
	_ = mgr.Disable("enable.example.com")

	// Now enable
	err = mgr.Enable("enable.example.com")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	// Verify metadata shows enabled
	loaded, err := mgr.Get("enable.example.com")
	if err != nil {
		t.Fatalf("Get after Enable: %v", err)
	}
	if !loaded.Enabled {
		t.Error("expected site to be enabled after Enable()")
	}
}

func TestDisable_Success(t *testing.T) {
	mgr, _, files, _ := setupTestSiteManager(t)

	// Create site first
	site, err := mgr.Create(CreateOptions{
		Domain:      "disable.example.com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
	})
	if err != nil {
		t.Fatalf("setup Create failed: %v", err)
	}

	// Pre-set the symlink as existing
	enabledPath := site.NginxEnabledPath(mgr.config.Nginx.SitesEnabled)
	files.exists[enabledPath] = true

	err = mgr.Disable("disable.example.com")
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	// Verify metadata shows disabled
	loaded, err := mgr.Get("disable.example.com")
	if err != nil {
		t.Fatalf("Get after Disable: %v", err)
	}
	if loaded.Enabled {
		t.Error("expected site to be disabled after Disable()")
	}
}

// --- buildVhostConfig ---

func TestBuildVhostConfig(t *testing.T) {
	mgr, _, _, _ := setupTestSiteManager(t)

	site := &Site{
		Domain:      "vhost.example.com",
		User:        "site_vhost_example_com",
		ProjectType: ProjectLaravel,
		PHPVersion:  "8.3",
		WebRoot:     "/home/site_vhost_example_com/vhost.example.com/public",
	}

	cfg := mgr.buildVhostConfig(site)

	if cfg.Domain != site.Domain {
		t.Errorf("Domain = %q, want %q", cfg.Domain, site.Domain)
	}
	if cfg.WebRoot != site.WebRoot {
		t.Errorf("WebRoot = %q", cfg.WebRoot)
	}
	if cfg.PHPSocket != site.PHPSocket() {
		t.Errorf("PHPSocket = %q, want %q", cfg.PHPSocket, site.PHPSocket())
	}
	if cfg.MaxBodySize != "64m" {
		t.Errorf("MaxBodySize = %q, want 64m", cfg.MaxBodySize)
	}
	sitesRoot := mgr.config.General.SitesRoot
	homeDir := site.HomeDir(sitesRoot)
	if cfg.AccessLog != homeDir+"/logs/nginx-access.log" {
		t.Errorf("AccessLog = %q", cfg.AccessLog)
	}
	if cfg.ErrorLog != homeDir+"/logs/nginx-error.log" {
		t.Errorf("ErrorLog = %q", cfg.ErrorLog)
	}
}
