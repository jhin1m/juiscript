# Phase 02: Site Management

## Context
Core feature: creating and deleting isolated sites. Each site gets a dedicated Linux user, home directory structure, PHP-FPM pool, Nginx vhost, and optionally a database. Supports Laravel and WordPress project types.

## Overview
- **Effort**: 5h
- **Priority**: P1
- **Status**: pending
- **Depends on**: Phase 01

## Key Insights
- User isolation is the security foundation: each site runs under its own Linux user with its own PHP-FPM pool.
- Directory structure differs between Laravel (`public/` as web root) and WordPress (`public_html/` convention).
- Site metadata stored in `/etc/juiscript/sites/` as individual TOML files per site.
- Deletion must be reversible-ish: disable first, delete after confirmation.

## Requirements
1. Create site: Linux user, directory structure, PHP-FPM pool, Nginx vhost, optional DB
2. Delete site: Remove user, directories, configs, optional DB
3. List sites with status (active/disabled)
4. Site metadata persistence in TOML
5. Support Laravel and WordPress project types
6. TUI screen: site list, create form, detail view, delete confirmation

## Architecture

### Site Package (`internal/site/`)
```go
type ProjectType string
const (
    ProjectLaravel   ProjectType = "laravel"
    ProjectWordPress ProjectType = "wordpress"
)

type Site struct {
    Domain      string      `toml:"domain"`
    User        string      `toml:"user"`        // linux username
    ProjectType ProjectType `toml:"project_type"`
    PHPVersion  string      `toml:"php_version"` // e.g., "8.3"
    WebRoot     string      `toml:"web_root"`    // computed from project type
    DBName      string      `toml:"db_name"`
    DBUser      string      `toml:"db_user"`
    SSLEnabled  bool        `toml:"ssl_enabled"`
    Enabled     bool        `toml:"enabled"`
    CreatedAt   time.Time   `toml:"created_at"`
}

type Manager struct {
    config   *config.Config
    executor system.Executor
    files    system.FileManager
    users    system.UserManager
    tpl      *template.Engine
}

func (m *Manager) Create(opts CreateOptions) (*Site, error)
func (m *Manager) Delete(domain string, removeDB bool) error
func (m *Manager) List() ([]*Site, error)
func (m *Manager) Get(domain string) (*Site, error)
func (m *Manager) Enable(domain string) error
func (m *Manager) Disable(domain string) error
```

### Directory Structure Per Site
```
# Laravel
/home/{user}/
├── {domain}/
│   ├── public/          <- web root (Nginx points here)
│   ├── storage/
│   ├── bootstrap/cache/
│   └── .env
├── logs/
│   ├── nginx-access.log
│   └── nginx-error.log
└── tmp/

# WordPress
/home/{user}/
├── public_html/{domain}/  <- web root
├── logs/
└── tmp/
```

### Create Flow
1. Validate domain (regex: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z]{2,})+$`)
2. Derive username from domain: `site_{sanitized_domain}` (max 32 chars)
3. `useradd -m -d /home/{user} -s /bin/bash {user}`
4. Create directory structure with correct ownership (`chown -R {user}:{user}`)
5. Set directory permissions: 750 for dirs, 640 for files
6. Generate PHP-FPM pool config from template -> `/etc/php/{ver}/fpm/pool.d/{domain}.conf`
7. Generate Nginx vhost from template -> `/etc/nginx/sites-available/{domain}`
8. Symlink to sites-enabled
9. Test Nginx config (`nginx -t`)
10. Reload PHP-FPM and Nginx
11. Save site metadata to `/etc/juiscript/sites/{domain}.toml`

### Delete Flow
1. Disable site (remove symlink, reload Nginx)
2. Remove PHP-FPM pool config, reload
3. Remove Nginx vhost config
4. Optionally drop DB + DB user
5. `userdel -r {user}` (removes home dir)
6. Remove site metadata file

## Related Files
```
internal/site/site.go        # Site struct, constants
internal/site/manager.go     # Manager with CRUD operations
internal/site/validate.go    # Domain/username validation
internal/site/manager_test.go
internal/tui/screens/sites.go       # Site list screen
internal/tui/screens/sitecreate.go  # Create site form
internal/tui/screens/sitedetail.go  # Site detail view
```

## Implementation Steps

1. **Site struct + validation**: Define `Site`, `CreateOptions`; domain regex validator; username derivation function
2. **Site metadata CRUD**: `Save()`, `Load()`, `LoadAll()` for TOML files in `/etc/juiscript/sites/`
3. **Manager.Create()**: Orchestrate user creation, dir setup, config generation, service reload
4. **Manager.Delete()**: Reverse of create with confirmation support
5. **Manager.Enable/Disable**: Symlink management + Nginx reload
6. **Manager.List()**: Read all site TOML files, return sorted list
7. **TUI site list screen**: Table with domain, type, PHP version, SSL, status columns
8. **TUI create form**: Huh form with domain, project type, PHP version, DB toggle fields
9. **TUI site detail**: Show site info, actions (enable/disable/delete/view logs)
10. **Integration**: Wire site manager into TUI app, add navigation

## Todo
- [ ] Site struct and TOML serialization
- [ ] Domain validation and username derivation
- [ ] Manager.Create with full orchestration
- [ ] Manager.Delete with cleanup
- [ ] Manager.Enable / Disable
- [ ] Manager.List
- [ ] TUI site list screen
- [ ] TUI create site form (Huh)
- [ ] TUI site detail screen
- [ ] Unit tests for validation and metadata

## Success Criteria
- `juiscript site create example.com --type laravel --php 8.3` creates full site with user, dirs, configs
- `juiscript site list` shows all sites with status
- `juiscript site delete example.com` cleanly removes everything
- TUI allows creating/managing sites through forms
- No orphaned resources on failure (rollback on error)

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Partial creation failure | High | Implement rollback: delete created resources on error |
| Username collision | Medium | Check user exists before creation; use domain-derived unique names |
| Home dir already exists | Medium | Check existence, fail with clear error |

## Security Considerations
- Validate domain against injection (no shell metacharacters)
- Username derived deterministically, sanitized (alphanumeric + underscore only)
- Site directories owned by site user, not root
- PHP-FPM pool runs as site user, not www-data
- File permissions: 750 dirs, 640 files (owner+group read, no world access)

## Next Steps
Phase 03 (Nginx Vhost) provides the detailed vhost template and management that this phase stubs out.
