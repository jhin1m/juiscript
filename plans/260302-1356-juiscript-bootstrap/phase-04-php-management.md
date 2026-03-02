# Phase 04: PHP Management

## Context
Multi-version PHP-FPM management via ondrej/php PPA. Each site has its own FPM pool running as the site's Linux user, with its own socket. Supports installing/removing PHP versions and switching sites between versions.

## Overview
- **Effort**: 5h
- **Priority**: P1
- **Status**: pending
- **Depends on**: Phase 01, Phase 02

## Key Insights
- ondrej/php PPA (`ppa:ondrej/php`) provides PHP 7.4 through 8.4+ on Ubuntu.
- Each PHP version has its own FPM service: `php8.3-fpm`, `php8.2-fpm`, etc.
- Pool configs live at `/etc/php/{version}/fpm/pool.d/{site}.conf`.
- Socket path convention: `/run/php/php{version}-fpm-{username}.sock`.
- Switching PHP version = new pool config + update Nginx vhost socket path + reload both.

## Requirements
1. Install/remove PHP versions with common extensions
2. Create/delete per-site FPM pool configs
3. Switch site PHP version (update pool + vhost + reload)
4. List installed PHP versions and their status
5. Configure pool settings (memory, workers, timeouts)
6. TUI screen for PHP version management and per-site switching

## Architecture

### PHP Package (`internal/php/`)
```go
var CommonExtensions = []string{
    "cli", "fpm", "common", "mysql", "xml", "mbstring",
    "curl", "zip", "gd", "bcmath", "intl", "readline",
    "opcache", "redis", "imagick",
}

type PoolConfig struct {
    SiteDomain    string
    Username      string
    PHPVersion    string
    ListenSocket  string   // /run/php/php{ver}-fpm-{user}.sock
    MaxChildren   int      // default: 5
    StartServers  int      // default: 2
    MinSpare      int      // default: 1
    MaxSpare      int      // default: 3
    MaxRequests   int      // default: 500
    MemoryLimit   string   // default: "256M"
    UploadMaxSize string   // default: "64M"
    Timezone      string   // default: "UTC"
}

type Manager struct {
    executor system.Executor
    files    system.FileManager
    tpl      *template.Engine
}

func (m *Manager) InstallVersion(version string) error
func (m *Manager) RemoveVersion(version string) error
func (m *Manager) ListVersions() ([]VersionInfo, error)
func (m *Manager) CreatePool(cfg PoolConfig) error
func (m *Manager) DeletePool(domain, version string) error
func (m *Manager) SwitchVersion(domain, fromVer, toVer string) error
func (m *Manager) ReloadFPM(version string) error
```

### PHP-FPM Pool Template
```ini
[{{ .SiteDomain }}]
user = {{ .Username }}
group = {{ .Username }}
listen = {{ .ListenSocket }}
listen.owner = www-data
listen.group = www-data
listen.mode = 0660

pm = dynamic
pm.max_children = {{ .MaxChildren }}
pm.start_servers = {{ .StartServers }}
pm.min_spare_servers = {{ .MinSpare }}
pm.max_spare_servers = {{ .MaxSpare }}
pm.max_requests = {{ .MaxRequests }}

php_admin_value[memory_limit] = {{ .MemoryLimit }}
php_admin_value[upload_max_filesize] = {{ .UploadMaxSize }}
php_admin_value[post_max_size] = {{ .UploadMaxSize }}
php_admin_value[date.timezone] = {{ .Timezone }}
php_admin_value[error_log] = /home/{{ .Username }}/logs/php-error.log
php_admin_flag[log_errors] = on

security.limit_extensions = .php
```

## Related Files
```
internal/php/manager.go
internal/php/pool.go
internal/php/manager_test.go
internal/tui/screens/php.go
templates/php-fpm-pool.conf.tmpl
```

## Implementation Steps

1. **Version detection**: Parse `ls /etc/php/` or `dpkg -l php*-fpm` to find installed versions
2. **InstallVersion()**: `apt-get install -y php{ver}-{ext}` for all common extensions; ensure PPA added
3. **RemoveVersion()**: Check no sites using it first; `apt-get remove php{ver}-*`
4. **PoolConfig struct**: All pool settings with sensible defaults
5. **CreatePool()**: Render template, atomic write to pool dir, reload FPM
6. **DeletePool()**: Remove config, reload FPM
7. **SwitchVersion()**: Create new pool (new version), update Nginx vhost socket path, delete old pool, reload both services
8. **ListVersions()**: Scan `/etc/php/` dirs, check service status for each
9. **TUI PHP screen**: Table of installed versions + status; per-site version switcher
10. **PPA management**: `add-apt-repository ppa:ondrej/php -y` on first install

## Todo
- [ ] PHP-FPM pool template
- [ ] Manager.InstallVersion with PPA check
- [ ] Manager.RemoveVersion with safety check
- [ ] Manager.ListVersions
- [ ] Manager.CreatePool
- [ ] Manager.DeletePool
- [ ] Manager.SwitchVersion (orchestration)
- [ ] TUI PHP screen
- [ ] Unit tests

## Success Criteria
- Can install PHP 8.3 and see it listed with status
- Creating a pool produces valid FPM config that passes `php-fpm{ver} -t`
- Switching a site from 8.2 to 8.3 updates both pool and vhost seamlessly
- No downtime during version switch (create new before removing old)

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| PPA not available | High | Check PPA exists before install; clear error message |
| Extension install failure | Medium | Continue with available extensions, report missing ones |
| Remove version with active sites | Critical | Refuse removal if any site uses the version |

## Security Considerations
- FPM pool runs as site user, not www-data
- Socket permissions: 0660, owner www-data (Nginx must read)
- `security.limit_extensions = .php` prevents arbitrary file execution
- `php_admin_value` (not `php_value`) prevents overriding via .user.ini
- Per-site error log in user's home directory

## Next Steps
Phase 05 (Database) adds MariaDB management; Phase 03 (Nginx) consumes the socket paths defined here.
