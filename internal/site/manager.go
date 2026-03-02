package site

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/nginx"
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
	nginx    *nginx.Manager // dedicated nginx vhost manager
}

// NewManager creates a site manager with all required dependencies.
func NewManager(cfg *config.Config, exec system.Executor, files system.FileManager, users system.UserManager, tpl *template.Engine) *Manager {
	return &Manager{
		config:   cfg,
		executor: exec,
		files:    files,
		users:    users,
		tpl:      tpl,
		nginx: nginx.NewManager(
			exec, files, tpl,
			cfg.Nginx.SitesAvailable,
			cfg.Nginx.SitesEnabled,
		),
	}
}

// Nginx returns the underlying nginx manager for direct vhost operations.
func (m *Manager) Nginx() *nginx.Manager {
	return m.nginx
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

	// Step 5: Create Nginx vhost via nginx.Manager (handles test + reload internally)
	vhostCfg := m.buildVhostConfig(site)
	if err := m.nginx.Create(vhostCfg); err != nil {
		rollback()
		return nil, fmt.Errorf("create nginx vhost: %w", err)
	}
	rollbacks = append(rollbacks, func() {
		m.nginx.Delete(site.Domain)
	})

	// Step 6: Reload PHP-FPM (nginx already reloaded by nginx.Manager)
	if err := m.reloadPHPFPM(site.PHPVersion); err != nil {
		rollback()
		return nil, fmt.Errorf("reload php-fpm: %w", err)
	}

	// Step 7: Save metadata (last step - everything else succeeded)
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

	// 1. Delete Nginx vhost (disable + remove config + reload)
	_ = m.nginx.Delete(domain)

	// 2. Remove PHP-FPM pool config
	m.files.Remove(site.FPMPoolConfigPath())

	// 3. Reload PHP-FPM
	_ = m.reloadPHPFPM(site.PHPVersion)

	// 4. Delete Linux user (and home dir with -r)
	if err := m.users.Delete(site.User); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	// 5. Remove metadata
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

// Enable activates a site via nginx.Manager and updates metadata.
func (m *Manager) Enable(domain string) error {
	if err := m.nginx.Enable(domain); err != nil {
		return err
	}

	site, err := m.Get(domain)
	if err != nil {
		return err
	}
	site.Enabled = true
	return SaveMetadata(config.SitesPath(), site)
}

// Disable deactivates a site via nginx.Manager and updates metadata.
func (m *Manager) Disable(domain string) error {
	if err := m.nginx.Disable(domain); err != nil {
		return err
	}

	site, err := m.Get(domain)
	if err != nil {
		return err
	}
	site.Enabled = false
	return SaveMetadata(config.SitesPath(), site)
}

// buildVhostConfig converts a Site into an nginx.VhostConfig.
func (m *Manager) buildVhostConfig(s *Site) nginx.VhostConfig {
	homeDir := s.HomeDir(m.config.General.SitesRoot)
	return nginx.VhostConfig{
		Domain:      s.Domain,
		WebRoot:     s.WebRoot,
		PHPSocket:   s.PHPSocket(),
		AccessLog:   homeDir + "/logs/nginx-access.log",
		ErrorLog:    homeDir + "/logs/nginx-error.log",
		ProjectType: nginx.ProjectType(s.ProjectType),
		MaxBodySize: "64m",
	}
}

// createDirs sets up the directory structure for a site.
func (m *Manager) createDirs(site *Site) error {
	homeDir := site.HomeDir(m.config.General.SitesRoot)
	ctx := context.Background()

	dirs := []string{
		site.SiteDir(m.config.General.SitesRoot),
		homeDir + "/logs",
		homeDir + "/tmp",
	}

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

	uid, gid, err := m.users.LookupUID(site.User)
	if err != nil {
		return fmt.Errorf("lookup user uid: %w", err)
	}

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

func (m *Manager) reloadPHPFPM(phpVersion string) error {
	ctx := context.Background()
	fpmService := "php" + phpVersion + "-fpm"
	if _, err := m.executor.Run(ctx, "systemctl", "reload", fpmService); err != nil {
		return fmt.Errorf("reload %s: %w", fpmService, err)
	}
	return nil
}
