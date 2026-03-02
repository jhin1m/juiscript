package site

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/system"
	"github.com/jhin1m/juiscript/internal/template"
)

// Manager orchestrates site lifecycle operations.
// It coordinates user creation, directory setup, config generation, and service reload.
type Manager struct {
	config   *config.Config
	executor system.Executor
	files    system.FileManager
	users    system.UserManager
	tpl      *template.Engine
}

// NewManager creates a site manager with all required dependencies.
func NewManager(cfg *config.Config, exec system.Executor, files system.FileManager, users system.UserManager, tpl *template.Engine) *Manager {
	return &Manager{
		config:   cfg,
		executor: exec,
		files:    files,
		users:    users,
		tpl:      tpl,
	}
}

// Create provisions a new site with user isolation.
// On any failure, it rolls back all created resources.
func (m *Manager) Create(opts CreateOptions) (*Site, error) {
	// Step 1: Validate inputs
	if err := ValidateDomain(opts.Domain); err != nil {
		return nil, err
	}
	if err := ValidateProjectType(opts.ProjectType); err != nil {
		return nil, err
	}
	if err := ValidatePHPVersion(opts.PHPVersion); err != nil {
		return nil, err
	}

	username := DeriveUsername(opts.Domain)

	// Check if site already exists
	if m.users.Exists(username) {
		return nil, fmt.Errorf("site user already exists: %s", username)
	}

	// Build site struct
	site := &Site{
		Domain:      opts.Domain,
		User:        username,
		ProjectType: opts.ProjectType,
		PHPVersion:  opts.PHPVersion,
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	// Set web root based on project type
	switch opts.ProjectType {
	case ProjectLaravel:
		site.WebRoot = site.SiteDir(m.config.General.SitesRoot) + "/public"
	case ProjectWordPress:
		site.WebRoot = site.SiteDir(m.config.General.SitesRoot)
	}

	// Track what we've created for rollback on failure
	var rollbacks []func()
	rollback := func() {
		// Execute rollbacks in reverse order
		for i := len(rollbacks) - 1; i >= 0; i-- {
			rollbacks[i]()
		}
	}

	// Step 2: Create Linux user
	homeDir := site.HomeDir(m.config.General.SitesRoot)
	if err := m.users.Create(username, homeDir); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	rollbacks = append(rollbacks, func() { m.users.Delete(username) })

	// Step 3: Create directory structure
	if err := m.createDirs(site); err != nil {
		rollback()
		return nil, fmt.Errorf("create dirs: %w", err)
	}

	// Step 4: Generate and write PHP-FPM pool config
	if err := m.createFPMPool(site); err != nil {
		rollback()
		return nil, fmt.Errorf("create FPM pool: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		m.files.Remove(site.FPMPoolConfigPath())
	})

	// Step 5: Generate and write Nginx vhost
	if err := m.createNginxVhost(site); err != nil {
		rollback()
		return nil, fmt.Errorf("create nginx vhost: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		m.files.Remove(site.NginxConfigPath(m.config.Nginx.SitesAvailable))
		m.files.RemoveSymlink(site.NginxEnabledPath(m.config.Nginx.SitesEnabled))
	})

	// Step 6: Test Nginx config
	if err := m.testNginx(); err != nil {
		rollback()
		return nil, fmt.Errorf("nginx config test failed: %w", err)
	}

	// Step 7: Reload services
	if err := m.reloadServices(site.PHPVersion); err != nil {
		rollback()
		return nil, fmt.Errorf("reload services: %w", err)
	}

	// Step 8: Save metadata (last step - everything else succeeded)
	if err := SaveMetadata(config.SitesPath(), site); err != nil {
		rollback()
		return nil, fmt.Errorf("save metadata: %w", err)
	}

	return site, nil
}

// Delete removes a site and all its resources.
func (m *Manager) Delete(domain string, removeDB bool) error {
	site, err := m.Get(domain)
	if err != nil {
		return fmt.Errorf("get site: %w", err)
	}

	// 1. Disable site first (remove nginx symlink)
	_ = m.Disable(domain)

	// 2. Remove PHP-FPM pool config
	m.files.Remove(site.FPMPoolConfigPath())

	// 3. Remove Nginx vhost config
	m.files.Remove(site.NginxConfigPath(m.config.Nginx.SitesAvailable))

	// 4. Reload services
	_ = m.reloadServices(site.PHPVersion)

	// 5. Delete Linux user (and home dir with -r)
	if err := m.users.Delete(site.User); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	// 6. Remove metadata
	return DeleteMetadata(config.SitesPath(), domain)
}

// List returns all managed sites sorted by domain.
func (m *Manager) List() ([]*Site, error) {
	return LoadAllMetadata(config.SitesPath())
}

// Get retrieves a single site by domain.
func (m *Manager) Get(domain string) (*Site, error) {
	return LoadMetadata(config.SitesPath(), domain)
}

// Enable activates a site by creating the Nginx symlink and reloading.
func (m *Manager) Enable(domain string) error {
	site, err := m.Get(domain)
	if err != nil {
		return err
	}

	available := site.NginxConfigPath(m.config.Nginx.SitesAvailable)
	enabled := site.NginxEnabledPath(m.config.Nginx.SitesEnabled)

	if err := m.files.Symlink(available, enabled); err != nil {
		return fmt.Errorf("create symlink: %w", err)
	}

	if err := m.testNginx(); err != nil {
		m.files.RemoveSymlink(enabled) // rollback
		return fmt.Errorf("nginx test failed after enable: %w", err)
	}

	m.reloadNginx()

	site.Enabled = true
	return SaveMetadata(config.SitesPath(), site)
}

// Disable deactivates a site by removing the Nginx symlink.
func (m *Manager) Disable(domain string) error {
	site, err := m.Get(domain)
	if err != nil {
		return err
	}

	enabled := site.NginxEnabledPath(m.config.Nginx.SitesEnabled)
	if err := m.files.RemoveSymlink(enabled); err != nil {
		return fmt.Errorf("remove symlink: %w", err)
	}

	m.reloadNginx()

	site.Enabled = false
	return SaveMetadata(config.SitesPath(), site)
}

// createDirs sets up the directory structure for a site.
func (m *Manager) createDirs(site *Site) error {
	homeDir := site.HomeDir(m.config.General.SitesRoot)
	ctx := context.Background()

	// Directories to create
	dirs := []string{
		site.SiteDir(m.config.General.SitesRoot),
		homeDir + "/logs",
		homeDir + "/tmp",
	}

	// Laravel needs extra dirs
	if site.ProjectType == ProjectLaravel {
		siteDir := site.SiteDir(m.config.General.SitesRoot)
		dirs = append(dirs,
			siteDir+"/public",
			siteDir+"/storage",
			siteDir+"/bootstrap/cache",
		)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// Set ownership to site user
	uid, gid, err := m.users.LookupUID(site.User)
	if err != nil {
		return fmt.Errorf("lookup user uid: %w", err)
	}

	// Recursively chown the home directory
	_, err = m.executor.Run(ctx, "chown", "-R",
		fmt.Sprintf("%d:%d", uid, gid), homeDir)
	if err != nil {
		return fmt.Errorf("chown home dir: %w", err)
	}

	return nil
}

// createFPMPool generates and writes the PHP-FPM pool config.
func (m *Manager) createFPMPool(site *Site) error {
	data := struct {
		PoolName     string
		User         string
		SocketPath   string
		MaxChildren  int
		StartServers int
		MinSpare     int
		MaxSpare     int
	}{
		PoolName:     site.User,
		User:         site.User,
		SocketPath:   site.PHPSocket(),
		MaxChildren:  5,
		StartServers: 2,
		MinSpare:     1,
		MaxSpare:     3,
	}

	rendered, err := m.tpl.Render("php-fpm-pool.conf.tmpl", data)
	if err != nil {
		return err
	}

	return m.files.WriteAtomic(site.FPMPoolConfigPath(), []byte(rendered), 0644)
}

// createNginxVhost generates the Nginx vhost config and enables it.
func (m *Manager) createNginxVhost(site *Site) error {
	data := struct {
		Domain      string
		Aliases     string
		WebRoot     string
		User        string
		ProjectType string
		PHPSocket   string
	}{
		Domain:      site.Domain,
		Aliases:     "www." + site.Domain,
		WebRoot:     site.WebRoot,
		User:        site.User,
		ProjectType: string(site.ProjectType),
		PHPSocket:   site.PHPSocket(),
	}

	rendered, err := m.tpl.Render("nginx-vhost.conf.tmpl", data)
	if err != nil {
		return err
	}

	available := site.NginxConfigPath(m.config.Nginx.SitesAvailable)
	if err := m.files.WriteAtomic(available, []byte(rendered), 0644); err != nil {
		return err
	}

	// Create symlink to enable
	enabled := site.NginxEnabledPath(m.config.Nginx.SitesEnabled)
	return m.files.Symlink(available, enabled)
}

func (m *Manager) testNginx() error {
	_, err := m.executor.Run(context.Background(), "nginx", "-t")
	return err
}

func (m *Manager) reloadNginx() {
	m.executor.Run(context.Background(), "systemctl", "reload", "nginx")
}

func (m *Manager) reloadServices(phpVersion string) error {
	ctx := context.Background()

	fpmService := "php" + phpVersion + "-fpm"
	if _, err := m.executor.Run(ctx, "systemctl", "reload", fpmService); err != nil {
		return fmt.Errorf("reload %s: %w", fpmService, err)
	}

	if _, err := m.executor.Run(ctx, "systemctl", "reload", "nginx"); err != nil {
		return fmt.Errorf("reload nginx: %w", err)
	}

	return nil
}
