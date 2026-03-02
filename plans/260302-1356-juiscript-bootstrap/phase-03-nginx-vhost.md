# Phase 03: Nginx/Vhost Management

## Context
Nginx vhost configuration generation and management. Each site gets a server block with proper PHP-FPM upstream, security headers, and project-type-specific rules (Laravel rewrite vs WordPress permalink support).

## Overview
- **Effort**: 4h
- **Priority**: P1
- **Status**: pending
- **Depends on**: Phase 01, Phase 02

## Key Insights
- `sites-available` / `sites-enabled` symlink pattern is standard on Ubuntu Nginx.
- Always run `nginx -t` before reload to catch config errors.
- Laravel needs `try_files $uri $uri/ /index.php?$query_string` rewrite.
- WordPress needs `try_files $uri $uri/ /index.php?$args` plus specific location blocks for wp-admin.
- Separate access/error logs per site for debugging.

## Requirements
1. Generate vhost config from template per project type (Laravel, WordPress)
2. Enable/disable vhosts via symlinks
3. Test config before reload (`nginx -t`)
4. Support SSL and non-SSL variants (SSL added later in Phase 06)
5. Custom Nginx directives per site (e.g., client_max_body_size)
6. TUI screen for viewing/editing vhost status

## Architecture

### Nginx Package (`internal/nginx/`)
```go
type VhostConfig struct {
    Domain       string
    WebRoot      string
    PHPSocket    string   // e.g., /run/php/php8.3-fpm-{user}.sock
    AccessLog    string
    ErrorLog     string
    SSLEnabled   bool
    SSLCertPath  string
    SSLKeyPath   string
    ProjectType  site.ProjectType
    MaxBodySize  string   // default: "64m"
    ExtraConfig  string   // raw Nginx directives
}

type Manager struct {
    executor system.Executor
    files    system.FileManager
    tpl      *template.Engine
}

func (m *Manager) Create(cfg VhostConfig) error   // render + write to sites-available
func (m *Manager) Delete(domain string) error      // remove from both dirs
func (m *Manager) Enable(domain string) error      // symlink sites-available -> sites-enabled
func (m *Manager) Disable(domain string) error     // remove symlink from sites-enabled
func (m *Manager) Test() error                     // nginx -t
func (m *Manager) Reload() error                   // systemctl reload nginx
func (m *Manager) List() ([]VhostInfo, error)      // list all vhosts with enabled status
```

### Nginx Vhost Template (Laravel)
```nginx
server {
    listen 80;
    server_name {{ .Domain }};
    root {{ .WebRoot }};
    index index.php index.html;

    access_log {{ .AccessLog }};
    error_log {{ .ErrorLog }};

    client_max_body_size {{ .MaxBodySize }};

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:{{ .PHPSocket }};
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.(?!well-known) {
        deny all;
    }
}
```

### Nginx Vhost Template (WordPress)
```nginx
server {
    listen 80;
    server_name {{ .Domain }} www.{{ .Domain }};
    root {{ .WebRoot }};
    index index.php index.html;

    access_log {{ .AccessLog }};
    error_log {{ .ErrorLog }};

    client_max_body_size {{ .MaxBodySize }};

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \.php$ {
        fastcgi_pass unix:{{ .PHPSocket }};
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location = /favicon.ico { log_not_found off; access_log off; }
    location = /robots.txt  { log_not_found off; access_log off; allow all; }

    location ~* \.(css|gif|ico|jpeg|jpg|js|png)$ {
        expires max;
        log_not_found off;
    }

    location ~ /\.(?!well-known) {
        deny all;
    }
}
```

## Related Files
```
internal/nginx/manager.go
internal/nginx/manager_test.go
internal/tui/screens/nginx.go
templates/nginx-laravel.conf.tmpl
templates/nginx-wordpress.conf.tmpl
templates/nginx-ssl.conf.tmpl       # SSL snippet included conditionally
```

## Implementation Steps

1. **Create Nginx templates**: Laravel vhost, WordPress vhost, SSL snippet (for later inclusion)
2. **VhostConfig struct**: All fields needed by templates
3. **Manager.Create()**: Render template, `WriteAtomic` to `/etc/nginx/sites-available/{domain}`, enable, test, reload
4. **Manager.Delete()**: Disable first, remove config file, reload
5. **Manager.Enable/Disable()**: Symlink management
6. **Manager.Test()**: Run `nginx -t`, parse output for errors
7. **Manager.Reload()**: `systemctl reload nginx`, check exit code
8. **Manager.List()**: Read `sites-available/`, check symlink existence in `sites-enabled/`
9. **TUI nginx screen**: List vhosts with status, actions to enable/disable/delete
10. **Error handling**: If `nginx -t` fails after create, rollback (remove config, restore previous)

## Todo
- [ ] Laravel vhost template
- [ ] WordPress vhost template
- [ ] SSL snippet template (placeholder for Phase 06)
- [ ] Nginx Manager CRUD
- [ ] `nginx -t` integration with error parsing
- [ ] Rollback on failed config test
- [ ] TUI vhost list screen
- [ ] Unit tests with mock executor

## Success Criteria
- Generated vhost passes `nginx -t`
- Enable/disable toggles symlink correctly
- Failed config test triggers automatic rollback
- Both Laravel and WordPress templates produce valid configs

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Config syntax error breaks all sites | Critical | Always `nginx -t` before reload; rollback on failure |
| Conflicting server_name | High | Check existing configs for duplicate domains |
| Socket path mismatch | Medium | Derive socket path from PHP version + username consistently |

## Security Considerations
- Deny access to dotfiles (`location ~ /\.`) except `.well-known` (for Let's Encrypt)
- No directory listing (`autoindex off` is Nginx default)
- Restrict `fastcgi_pass` to Unix socket (no TCP)
- Per-site log files for audit trail

## Next Steps
Phase 04 (PHP Management) creates the PHP-FPM pool configs that this phase references via socket paths.
