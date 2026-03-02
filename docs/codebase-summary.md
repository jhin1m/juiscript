# Codebase Summary

## Project Structure

```
juiscript/
├── cmd/juiscript/
│   └── main.go                 # CLI entry, Cobra root, TUI launcher
├── internal/
│   ├── config/
│   │   ├── config.go           # TOML config struct, Load/Save, defaults
│   │   └── config_test.go
│   ├── system/
│   │   ├── executor.go         # Command execution interface & impl
│   │   ├── fileops.go          # File ops interface & atomic writes
│   │   ├── usermgmt.go         # Linux user management
│   │   └── fileops_test.go
│   ├── template/
│   │   ├── engine.go           # Template engine, embedded fs
│   │   └── engine_test.go
│   ├── nginx/
│   │   ├── manager.go          # Vhost CRUD, test, reload, enable/disable
│   │   └── manager_test.go
│   ├── php/
│   │   ├── manager.go          # PHP version install/remove/list, FPM reload
│   │   ├── pool.go             # PoolConfig, CreatePool, DeletePool, SwitchVersion
│   │   └── manager_test.go     # 19 unit tests
│   ├── site/
│   │   └── manager.go          # Site lifecycle (uses nginx.Manager)
│   ├── database/
│   │   ├── manager.go          # Manager struct, validation, password gen
│   │   ├── database-operations.go # CreateDB/DropDB/ListDBs
│   │   ├── user-operations.go  # CreateUser/DropUser/ResetPassword
│   │   ├── import-export.go    # Import/Export with streaming
│   │   └── manager_test.go     # 20 unit tests
│   ├── ssl/
│   │   ├── manager.go          # Let's Encrypt automation via certbot
│   │   └── manager_test.go     # Unit tests
│   ├── nginx/ssl-operations.go # EnableSSL/DisableSSL vhost operations
│   ├── nginx/ssl-operations_test.go # Tests
│   ├── supervisor/
│   │   ├── manager.go          # Queue worker lifecycle (create/delete/start/stop/status)
│   │   └── manager_test.go     # Unit tests
│   ├── backup/
│   │   ├── manager.go          # Backup lifecycle (create/restore/list/delete/cleanup/cron)
│   │   └── manager_test.go     # 34 unit tests
│   ├── provisioner/
│   │   ├── detector.go         # Package detection (Phase 01)
│   │   ├── detector_test.go    # 8 unit tests
│   │   ├── installer.go        # Package installation (Phase 02, 196 lines)
│   │   └── installer_test.go   # 12 unit tests
│   └── tui/
│       ├── app.go              # Root model & screen router
│       ├── components/
│       │   ├── header.go       # Header component
│       │   ├── statusbar.go    # Status bar component
│       │   └── theme/
│       │       └── theme.go    # Color scheme & styles
│       └── screens/
│           ├── dashboard.go    # Dashboard screen
│           ├── nginx.go        # Nginx vhost management screen
│           ├── php.go          # PHP version management screen
│           ├── database.go     # Database management screen
│           ├── queues.go       # Queue worker management screen (Phase 08)
│           ├── services.go     # Service control screen
│           └── backup.go       # Backup/restore management screen (Phase 09)
├── templates/
│   ├── nginx-laravel.conf.tmpl     # Laravel vhost template
│   ├── nginx-wordpress.conf.tmpl   # WordPress vhost template
│   ├── nginx-ssl.conf.tmpl         # SSL vhost template
│   ├── php-fpm-pool.conf.tmpl      # PHP-FPM pool template
│   └── supervisor-worker.conf.tmpl # Queue worker template
├── Makefile                    # Build targets
└── README.md                   # Quick start guide
```

## Key Files & Responsibilities

### cmd/juiscript/main.go (57 lines)
**Purpose**: CLI entry point
- Cobra root command with TUI as default action
- Version command for build info
- Launches Bubble Tea program
- Build-time ldflags: version, commit

### internal/config/config.go (157 lines)
**Purpose**: Configuration management
- Config struct with 6 sections (General, Nginx, PHP, Database, Backup, Redis)
- Sensible Ubuntu defaults
- Load from TOML (fallback to defaults if missing)
- Save with restrictive permissions (0640)
- Helper: EnsureDirs creates required directories

**Key Types**:
```go
Config
├── General (sites_root, log_file, backup_dir)
├── Nginx (sites_available, sites_enabled, conf_dir)
├── PHP (default_version, versions[])
├── Database (root_user, socket_path)
├── Backup (dir, retention_days, compress_level)
└── Redis (max_databases)
```

### internal/system/executor.go (80 lines)
**Purpose**: Command execution abstraction
- Executor interface: Run, RunWithInput
- Wraps os/exec with context, timeouts, logging
- Default 30s timeout (configurable per call)
- Logs all commands: name, args, duration, exit code
- Returns stdout on success, stderr on failure

### internal/system/fileops.go (116 lines)
**Purpose**: Safe filesystem operations
- FileManager interface: 8 methods
- WriteAtomic: Temp file + atomic rename (prevents corruption)
- Symlink: Safe creation with existing link removal
- RemoveSymlink: Validates symlink before delete
- Remove, Exists, ReadFile: Basic file ops

### internal/system/usermgmt.go (91 lines)
**Purpose**: Linux user creation & deletion
- UserManager interface: Create, Delete, Exists, LookupUID
- Create: useradd with home dir, bash shell
- Delete: userdel with home dir removal
- LookupUID: Returns numeric UID/GID for site user
- Backed by system commands (useradd/userdel)

### internal/template/engine.go (50 lines)
**Purpose**: Template rendering engine
- Embeds templates via //go:embed
- Parse all templates at startup (fail-fast)
- Render: Execute named template with data
- Available: List all loaded template names
- Used for Nginx vhost, PHP-FPM, Supervisor configs

### internal/tui/app.go (171 lines)
**Purpose**: Root Bubble Tea model & screen router
- App struct: Theme, Header, StatusBar, current screen, dimensions
- NewApp: Initializes with Dashboard screen
- Update: Handles keyboard input, window resize, navigation messages
- View: Renders header + content + status bar
- Navigation: 'j'/'k' to move, 'enter' to select, 'q' to quit
- Custom messages: NavigateMsg (screen change), GoBackMsg (return to dashboard)

### internal/tui/components/header.go (27 lines)
**Purpose**: App header component
- Shows app title and current screen name
- Centered, styled with theme
- SetWidth: Updates width for responsive layout

### internal/tui/components/statusbar.go (29 lines)
**Purpose**: Bottom status bar with key bindings
- Displays active key bindings based on screen
- Layout: "Key: Description" format
- Right-aligned at bottom

### internal/tui/components/service-status-bar.go (128 lines) - Phase 01
**Purpose**: Horizontal health indicator bar for LEMP services
- Renders one-line service overview (green dot=active, red dot=failed, hollow=inactive)
- Shows memory usage for active services (e.g., "● nginx 45MB")
- Displays between header and content on all TUI screens
- Smart truncation for narrow terminals ("+N more" indicator)
- Service name formatting: Shortens "php8.3-fpm" → "php8.3", "redis-server" → "redis"

**Key Methods**:
- NewServiceStatusBar(t *Theme): Constructor
- SetServices([]service.Status): Update service list
- SetWidth(w int): Set available terminal width for truncation
- SetError(msg string): Display error instead of services (shows ⚠ icon)
- View(): Renders single-line status bar

**Truncation Logic**:
- Full width: All services + memory (if fits)
- Narrow: Shows progressively fewer services + "+N more" suffix
- Extreme: Only "+N more" fallback when can't fit any service
- Uses lipgloss.Width() for accurate rendering width (accounts for ANSI codes)

### internal/tui/components/helpers.go (NEW) - Phase 01
**Purpose**: Shared statusIndicator helper for service components
- Returns colored dot symbol + style based on service state
- Used by ServicePanel and ServiceStatusBar for consistent rendering
- Mapping: "active"→●(green), "failed"→●(red), default→○(gray)

### internal/tui/theme/theme.go (varies)
**Purpose**: Centralized styling
- Lip Gloss styles for colors/formatting
- Title, Subtitle, Selected, Disabled, Error styles
- Theme struct: All styles together
- Passed to components for consistency

### internal/tui/screens/dashboard.go (varies)
**Purpose**: Main menu screen
- 8 menu items (Sites, Nginx, PHP, Database, SSL, Services, Queues, Backup)
- List selection with 'j'/'k' navigation
- 'enter' emits NavigateMsg to router
- Shows feature descriptions

### internal/nginx/manager.go (268 lines)
**Purpose**: Nginx vhost CRUD and reload management
- Vhost creation from templates with validation
- Enable/disable via symlinks in sites-available/sites-enabled
- Config testing via `nginx -t` with error parsing
- Rollback on failure (atomic operations)
- List vhosts with enabled status
- Delete vhosts with cleanup
- Reload Nginx via systemctl

**Key Types**:
```go
ProjectType = "laravel" | "wordpress"

VhostConfig {
  Domain, WebRoot, PHPSocket, AccessLog, ErrorLog
  SSLEnabled, SSLCertPath, SSLKeyPath
  ProjectType, MaxBodySize, ExtraConfig
}

VhostInfo { Domain, Enabled, Path }

Manager {
  executor, files, tpl
  sitesAvailable, sitesEnabled
}
```

### internal/tui/screens/nginx.go (143 lines)
**Purpose**: TUI screen for vhost management
- List all vhosts with enabled/disabled status
- Keyboard: 'k'/'j' navigate, 'e' toggle, 'd' delete, 't' test config
- Error display and empty state handling
- Table view with Domain, Status, Path columns

### internal/php/manager.go (239 lines)
**Purpose**: PHP version installation, removal, and FPM service management
- EnsurePPA: Adds ondrej/php PPA if not already present
- InstallVersion: Installs PHP version with common & optional extensions
- RemoveVersion: Removes PHP version (validates no sites use it)
- ListVersions: Scans /etc/php/ and retrieves FPM service status
- ReloadFPM: Reloads specific PHP-FPM version service
- Extension management: Common (cli, fpm, mysql, xml, etc) & optional (redis, imagick)
- Atomic service control via systemctl
- 5-minute timeout for apt-get install, 3-minute for remove

**Key Types**:
```go
VersionInfo {
  Version string   // e.g. "8.3"
  Active bool      // FPM service running
  Enabled bool     // FPM service enabled at boot
}
```

### internal/php/pool.go (194 lines)
**Purpose**: PHP-FPM pool configuration per site
- PoolConfig: Settings for a single site's FPM pool
- DefaultPool: Returns PoolConfig with sensible defaults
- CreatePool: Renders template, writes atomically, reloads FPM
- DeletePool: Removes pool config and reloads FPM
- SwitchVersion: Zero-downtime PHP version switch for sites
  - Creates new pool → reloads Nginx → deletes old pool
  - Rollback on failure: removes new pool if Nginx reload fails

**Key Types**:
```go
PoolConfig {
  SiteDomain, Username, PHPVersion, ListenSocket
  MaxChildren, StartServers, MinSpare, MaxSpare, MaxRequests
  MemoryLimit, UploadMaxSize, Timezone
}
```

### internal/tui/screens/php.go (141 lines)
**Purpose**: TUI screen for PHP version management
- Displays installed PHP versions with FPM status & boot status
- Keyboard: 'i' install, 'r' remove, 'k'/'j' navigate, 'esc' back
- Table view: VERSION | FPM STATUS | BOOT columns
- Color-coded status: running/stopped (green/red), enabled/disabled
- Messages: InstallPHPMsg, RemovePHPMsg for app routing

### internal/database/manager.go (80 lines)
**Purpose**: Database management foundation
- Manager struct wraps Executor for MariaDB operations
- Validation regex: `^[a-z][a-z0-9_]{0,63}$` (lowercase, alphanumeric, underscore)
- GeneratePassword: Cryptographically secure 24-char passwords (default length)
- Socket authentication (no password needed as root)
- System database protection: information_schema, mysql, performance_schema, sys

**Key Types**:
```go
DBInfo {
  Name string       // Database name
  SizeMB float64    // Total size in MB
  Tables int        // Table count
}

Manager {
  executor Executor  // For running mysql commands
}
```

### internal/database/database-operations.go (71 lines)
**Purpose**: Database CRUD operations
- CreateDB: Creates UTF-8MB4 database with unicode collation
- DropDB: Drops user database (prevents system DB deletion)
- ListDBs: Returns all databases with size/table metadata
- Uses information_schema for size/table metrics
- Error wrapping for context

### internal/database/user-operations.go (81 lines)
**Purpose**: Database user lifecycle management
- CreateUser: Creates user with full privileges on database, returns 24-char password
- DropUser: Revokes privileges, drops user, flushes privileges
- ResetPassword: Generates new password, updates user, returns password
- All operations scoped to 'localhost' for security
- Batch operations to prevent partial state

### internal/database/import-export.go (89 lines)
**Purpose**: Database backup/restore via streaming
- Import: Loads SQL file (uncompressed or gzip) into database
- Export: Dumps database via mysqldump (uncompressed or gzip)
- Streaming commands to handle large files without memory issues
- 10-minute timeout for import/export operations
- Path validation: Rejects shell metacharacters, prevents injection
- Export uses --single-transaction for consistent snapshots without locks

### internal/database/manager_test.go (varies)
**Purpose**: 20 unit tests for database operations
- Tests for validation (name format, system DB protection)
- Password generation tests
- CRUD tests with mocked executor
- Import/export path validation
- Edge cases: empty names, invalid characters, system databases

### internal/tui/screens/database.go (144 lines)
**Purpose**: TUI screen for database management
- Lists databases with size (MB) and table count
- Table view: DATABASE | SIZE | TABLES columns
- Keyboard: 'k'/'j' navigate, 'c' create, 'd' drop, 'i' import, 'e' export
- Error display for failed operations
- Messages: CreateDBMsg, DropDBMsg, ImportDBMsg, ExportDBMsg for app routing
- Empty state with help text

### internal/ssl/manager.go (291 lines)
**Purpose**: Let's Encrypt SSL certificate automation via certbot
- Manager struct: wraps Executor, Nginx Manager, FileManager
- Obtain(): Requests cert via certbot webroot, updates Nginx vhost with SSL directives, reloads
- Revoke(): Revokes cert, deletes certbot files, disables SSL in vhost
- Renew(): Forces certificate renewal
- Status(): Parses certificate with openssl, returns expiry/issuer/validity
- ListCerts(): Parses certbot output for all managed certificates
- Validation: validateDomain (DNS chars only), validateEmail (basic format check)
- Certificate parsing: Handles openssl x509 output and certbot certificates listing

**Key Types**:
```go
CertInfo {
  Domain   string      // Certificate domain
  Expiry   time.Time   // Certificate expiration datetime
  Issuer   string      // Issuer CN (e.g., "Let's Encrypt")
  Valid    bool        // Current validity status
  DaysLeft int         // Days until expiration
}

Manager {
  executor system.Executor
  nginx    *nginx.Manager
  files    system.FileManager
}
```

### internal/nginx/ssl-operations.go (202 lines)
**Purpose**: Nginx vhost SSL enable/disable operations
- EnableSSL(): Injects SSL directives, prepends HTTP-to-HTTPS redirect block
- DisableSSL(): Removes SSL sections from vhost config
- injectSSLDirectives(): Adds listen 443, certificate paths, TLS settings, OCSP stapling
- buildRedirectBlock(): Creates server block for HTTP→HTTPS redirect with ACME challenge support
- removeSSLSections(): Uses marker comments to cleanly strip SSL config
- Atomic with rollback: Tests config and restores original on failure

**SSL Markers for Clean Injection**:
```
# BEGIN SSL REDIRECT ... # END SSL REDIRECT
# BEGIN SSL ... # END SSL
```

**TLS Configuration Injected**:
- Protocols: TLSv1.2, TLSv1.3
- Ciphers: ECDHE-ECDSA-AES128/256-GCM-SHA256/384, ECDHE-RSA-AES128/256-GCM-SHA256/384
- OCSP stapling enabled
- HSTS header available (commented, require manual activation)

### internal/tui/screens/ssl.go (165 lines)
**Purpose**: TUI screen for SSL certificate management
- Displays certificates in table: DOMAIN | DAYS LEFT | STATUS | ISSUER
- Keyboard: 'k'/'j' navigate, 'o' obtain, 'r' revoke, 'f' force-renew, 'esc' back
- Color-coded status: VALID (green), EXPIRING (yellow, ≤30 days), CRITICAL (red, ≤7 days), EXPIRED (red)
- Messages: ObtainCertMsg, RevokeCertMsg, RenewCertMsg for app routing
- Empty state with obtain hint
- Cursor selection and error display

**Screen Title**: SSL

### Makefile (50 lines)
**Purpose**: Build automation
- build: Current platform
- build-linux: Linux AMD64 for servers
- test: Run all tests
- cover: Coverage report (HTML)
- fmt: gofmt + govet
- dev: Build and run
- clean: Remove artifacts

## Core Data Structures

### Config (TOML → Go)
```toml
[general]
sites_root = "/home"
log_file = "/var/log/juiscript.log"
backup_dir = "/var/backups/juiscript"

[nginx]
sites_available = "/etc/nginx/sites-available"
sites_enabled = "/etc/nginx/sites-enabled"

[php]
default_version = "8.3"
versions = ["8.3"]

[database]
root_user = "root"
socket_path = "/var/run/mysqld/mysqld.sock"

[backup]
dir = "/var/backups/juiscript"
retention_days = 30
compress_level = 6

[redis]
max_databases = 16
```

## Interfaces (Abstractions for Testing)

```go
Executor {
  Run(ctx, name, args) → (string, error)
  RunWithInput(ctx, input, name, args) → (string, error)
}

FileManager {
  WriteAtomic(path, data, perm) → error
  Symlink(target, link) → error
  RemoveSymlink(path) → error
  Remove(path) → error
  Exists(path) → bool
  ReadFile(path) → ([]byte, error)
}

UserManager {
  Create(username, homeDir) → error
  Delete(username) → error
  Exists(username) → bool
  LookupUID(username) → (uid, gid int, error)
}
```

## Design Patterns in Use

1. **Interface Abstraction**: All OS operations behind interfaces
2. **Dependency Injection**: Pass interfaces to constructors
3. **Atomic Operations**: Temp file + rename for configs
4. **Error Wrapping**: Always use %w in fmt.Errorf
5. **Fail-Fast**: Parse templates at startup, validate config early
6. **Structured Logging**: slog for audit trail
7. **Table-Driven Tests**: Multiple scenarios per test
8. **Screen Router**: TUI app delegates to screen models

## Dependencies

### Go Modules
- charmbracelet/bubbletea: TUI framework
- charmbracelet/bubbles: Form/list components
- charmbracelet/lipgloss: Styling
- charmbracelet/huh: Advanced forms
- spf13/cobra: CLI framework
- BurntSushi/toml: Config parsing

### System Requirements
- Ubuntu 22.04 or 24.04
- Nginx, PHP-FPM, MariaDB, Redis
- Supervisor (for queue workers)
- certbot (for SSL)

## Code Quality Metrics

- **Coverage**: Tests for system, config, template packages
- **Linting**: gofmt + govet in Makefile
- **Error Handling**: Comprehensive with wrapped errors
- **Testing**: No root required (interfaces mocked)
- **Logging**: Structured slog throughout

## Phase Completion Status

**Phase 01 - Infrastructure & Package Detection**: Config, system abstractions, template engine, basic TUI, Service Status Bar component, Package Detector ✓
  - ServiceStatusBar (horizontal LEMP health indicator, 128 lines)
  - statusIndicator helper (shared with ServicePanel, 20 lines)
  - 15 comprehensive unit tests for ServiceStatusBar
  - Detector: Package detection for Nginx, MariaDB, Redis, PHP versions (148 lines)
  - 8 unit tests for package detection (isInstalled, DetectAll, isVersionDir)
**Phase 02 - Package Installation**: Installer with apt-get, service management, MariaDB hardening ✓
  - Installer: Idempotent package installation (InstallNginx, InstallRedis, InstallMariaDB, InstallPHP, AptUpdate)
  - 12 unit tests covering success, already-installed, apt-failure, hardening-failure scenarios
  - Shared isPackageInstalled function with Detector (DRY)
  - MariaDB hardening: Remove test DB, anonymous users, remote root access
**Phase 03 - Site Management**: Site lifecycle manager, site creation/deletion (future TUI integration)
**Phase 03 - Nginx/Vhost**: Manager CRUD, templates, TUI screen, enable/disable ✓
**Phase 04 - PHP Management**: Version install/remove/list, FPM pool CRUD, version switch, TUI screen ✓
**Phase 05 - Database Management**: Manager CRUD, user ops, import/export, TUI screen, 20 unit tests ✓
**Phase 06 - SSL Management**: Certbot automation, Nginx SSL injection, TUI screen, full unit tests ✓
**Phase 07 - Service Control**: Systemctl wrapper, start/stop/restart/reload/status/health, TUI screen, 16 unit tests ✓
**Phase 08 - Supervisor/Queue Workers**: Worker lifecycle, template full params, TUI screen, supervisorctl integration ✓
**Phase 09 - Backup System**: Full/partial backups, restore, list, delete, cleanup, cron scheduling, TUI screen, 34 unit tests ✓

## Service Control Implementation Details (Phase 07)

### Service Manager (internal/service/manager.go - 256 lines)
**Purpose**: Systemd service control for LEMP stack
- ServiceName type with whitelist (Nginx, MariaDB, Redis, PHP-FPM versions)
- Manager struct wraps Executor for systemctl operations
- Validation: Prevents arbitrary service names via allowedServices map + regex for PHP-FPM

**Key Operations**:
- **Start/Stop/Restart/Reload**: Execute systemctl actions with context
- **IsActive**: Check if service is currently running (systemctl is-active)
- **Status**: Detailed status parsing (state, substate, PID, memory in MB)
- **ListAll**: Enumerate all LEMP services (static + dynamic PHP versions)
- **IsHealthy**: Health check - verifies critical services (Nginx, MariaDB, PHP)

**Service Discovery**:
- Static: Nginx, MariaDB, Redis (always checked)
- Dynamic: Scans /etc/php/ for installed PHP versions (e.g., "8.3", "7.4")
- PHP-FPM service names auto-generated: PHPFPMService("8.3") → "php8.3-fpm"

**Status Structure**:
```go
Status {
  Name ServiceName       // e.g., "nginx", "php8.3-fpm"
  Active bool            // state == "active"
  State string           // "active", "inactive", "failed", "not-found"
  SubState string        // "running", "dead", "exited"
  PID int                // Main process ID (0 if stopped)
  MemoryMB float64       // Converted from bytes via systemctl show
}
```

**Security**: Whitelist-based validation prevents command injection

### Service Manager Tests (internal/service/manager_test.go - 286 lines)
**Coverage**: 16 test cases
- isAllowed: Tests whitelist (nginx, mariadb, redis-server, php8.3-fpm, invalid services)
- Start/Stop/Restart: Basic operation execution
- Reload: Graceful reload signal (tested with php8.3-fpm)
- Blocked services: Rejection of non-whitelisted services
- Systemctl failure: Error handling and propagation
- IsActive: True/false states and blocked service handling
- Status parsing: Property extraction (ActiveState, SubState, MainPID, MemoryCurrent)
- Inactive services: PID=0 and memory=0 handling
- PHPFPMService: Service name generation for versions
- isNumeric: Validation helper for version parsing

### Services TUI Screen (internal/tui/screens/services.go - 183 lines)
**Purpose**: Full-screen service management interface
- Service list display with table: SERVICE | STATE | SUBSTATE | PID | MEMORY
- Cursor selection with 'j'/'k' navigation
- Action keys: 's' (start), 'x' (stop), 'r' (restart), 'l' (reload)
- Color-coded states: active (green), failed (red), other (gray)
- Empty state handling with help text
- Error display and message routing

**Screen Components**:
- SetServices/SetError: Update displayed data
- stateDisplay: Returns display string + style for service state
- serviceCmd: Wraps tea.Msg into tea.Cmd for async execution
- Messages: StartServiceMsg, StopServiceMsg, RestartServiceMsg, ReloadServiceMsg

### Service Panel Component (internal/tui/components/servicepanel.go - REFACTORED)
**Purpose**: Compact service status overview for dashboard
- Reusable component for dashboard integration
- Shows service name + colored status indicator via shared statusIndicator helper
- Green dot (●) for active, red (●) for failed, hollow (○) for inactive
- Minimal footprint for dashboard display
- SetServices: Update service list
- **Refactored**: Now uses shared statusIndicator(t, state) from helpers.go

### internal/tui/components/service-status-bar_test.go (233 lines) - Phase 01
**Purpose**: 15 comprehensive unit tests for ServiceStatusBar component
**Test Coverage**:
1. NewServiceStatusBar: Constructor initialization
2. EmptyState: "No services detected" message when no services
3. ErrorState: Shows ⚠ icon + error message
4. ErrorClearedOnSetServices: Error clears when new services arrive
5. ActiveService: Green ● dot + name + memory for running services
6. InactiveService: Hollow ○ dot, no memory display
7. FailedService: Red ● dot for failed state
8. FormatServiceName: Name shortening (php8.3-fpm→php8.3, redis-server→redis)
9. MultipleServices: Pipe (│) separator, multiple service rendering
10. Truncation: "+N more" indicator when width constrained (40 chars)
11. NoTruncationWideEnough: All services shown when width sufficient (200 chars)
12. ActiveServiceWithZeroMemory: Omits 0MB for active services with no memory data
13. TruncationExtreme: Fallback to "+N more" when can't fit even one service (5 char width)
14. TruncationZeroWidth: Empty view when width ≤ padding (2 chars)
15. SetWidth: Width property setter

**Test Patterns**:
- Table-driven for FormatServiceName (5 scenarios)
- lipgloss.Width() for accurate ANSI code handling
- Mock theme.New() for all tests
- Multiple service combinations (4-service scenario)

### App Integration (internal/tui/app.go)
**Changes**:
- Added ScreenServices enum value
- Added "Services": ScreenServices to screenNames map
- Services screen instantiated and routed like other feature screens
- Dashboard menu includes "Services" option
- Service action messages integrated with app router

## PHP Management Implementation Details

### CommonExtensions (auto-installed with each version)
`cli`, `fpm`, `common`, `mysql`, `xml`, `mbstring`, `curl`, `zip`, `gd`, `bcmath`, `intl`, `readline`, `opcache`

### OptionalExtensions (skipped on install failure)
`redis`, `imagick`

### FPM Pool Settings
- **Process Manager**: Dynamic mode (adjusts workers based on traffic)
- **Socket Path**: `/run/php/php{version}-fpm-{username}.sock`
- **Security**: open_basedir, sys_temp_dir, upload_tmp_dir restrictions
- **PHP Admin Values**: memory_limit, upload_max_filesize, post_max_size, date.timezone
- **Logging**: Error logs at `/home/{user}/logs/php-error.log`

### PHP-FPM Template (php-fpm-pool.conf.tmpl)
- Pool name derived from site domain
- Per-site user/group isolation
- Socket with 0660 permissions, www-data owner
- Dynamic process manager with configurable limits
- PHP execution restricted via open_basedir
- Security extensions limited to .php files

## Database Management Implementation Details

### Validation & Security
- **Name Format**: `^[a-z][a-z0-9_]{0,63}$` (64 chars max, lowercase, alphanumeric, underscore)
- **System DB Protection**: Prevents dropping critical databases
- **SQL Injection Prevention**: Name validation + backtick escaping for identifiers
- **Password Security**: 24-char randomized with 74-char charset (letters, digits, symbols)
- **Authentication**: Socket auth as root (no password transmitted)

### User & Database Operations
- **CreateUser**: Single batch statement (create + grant + flush)
- **DropUser**: Revokes privileges before drop for safety
- **CreateDB**: UTF-8MB4 charset, Unicode collation for i18n
- **Password Reset**: Generates new password, updates in one statement

### Import/Export Features
- **Import**: Supports plain SQL and gzip-compressed files
- **Export**: Single-transaction snapshots (no table locks)
- **Streaming**: Pipes large files via shell to avoid memory bloat
- **Path Validation**: Regex rejects shell metacharacters (`/ . - _` allowed)
- **Timeout**: 10-minute limit for long-running operations

### Database Metadata
- **Size Calculation**: information_schema.TABLES (data + index length)
- **Table Count**: Per-database table enumeration
- **System DB Filtering**: Automatic exclusion in ListDBs output

## SSL Management Implementation Details

### Certificate Operations
- **Obtain**: Uses certbot webroot method for ACME challenge validation, then injects SSL into Nginx vhost
- **Revoke**: Revokes cert via certbot, deletes certbot files, removes SSL from vhost
- **Renew**: Forces certificate renewal (useful before auto-renewal)
- **Status**: Parses openssl x509 output to extract expiry, issuer, validity
- **List**: Parses certbot certificates output for all managed certs

### Certbot Configuration
- Method: `--webroot` for zero-downtime validation (no port 80/443 temporarily required)
- Options: `--non-interactive`, `--agree-tos` for automation
- Certificate path: `/etc/letsencrypt/live/{domain}/`
- Email: Required for ACME registration and expiry notifications

### Nginx SSL Injection
- **Location**: Inserts after "listen [::]:80;" line in vhost config
- **Redirect Block**: Prepended to vhost, handles HTTP→HTTPS with ACME challenge path exception
- **Markers**: Uses comments (# BEGIN SSL, # END SSL) for clean removal without parsing
- **Atomic**: Writes atomically, tests config, rolls back if `nginx -t` fails
- **TLS Version**: TLSv1.2 and TLSv1.3 only (no TLSv1.0/1.1)
- **OCSP Stapling**: Enabled for performance, requires resolver configuration

### Certificate Status Colors
- **VALID** (Green): Expires in >30 days
- **EXPIRING** (Yellow): Expires in 8-30 days
- **CRITICAL** (Red): Expires in ≤7 days
- **EXPIRED** (Red): Already expired

### Security Validations
- Domain: Allows only letters, digits, dots, hyphens (DNS-valid)
- Email: Requires @ symbol, allows alphanumeric, dots, hyphens, underscores, plus
- Path Traversal: Rejected via validation before passing to commands
- Command Injection: Input validation prevents shell metacharacters

### internal/supervisor/manager.go (368 lines)
**Purpose**: Supervisor queue worker lifecycle management
- WorkerConfig struct: Domain, Username, SitePath, PHPBinary, Connection, Queue, Processes, Tries, MaxTime, Sleep
- applyDefaults: Sensible defaults (redis, default queue, 1 process, 3 tries, 3600s max-time, 3s sleep)
- Create: Renders supervisor-worker template, writes atomically, reloads (rollback on failure)
- Delete: Removes config, reloads (idempotent)
- Start/Stop/Restart: Control worker processes via supervisorctl
- Status: Parses supervisorctl output for state, PID, uptime
- ListAll: Enumerate all managed workers
- reload: Executes reread + update for config discovery and application
- Validation: Domain (DNS chars), processes (≤8), required fields

**Key Types**:
```go
WorkerConfig {
  Domain, Username, SitePath, PHPBinary
  Connection (redis/database/sqs), Queue, Processes, Tries, MaxTime, Sleep
}

WorkerStatus {
  Name string           // program name (e.g., "example.com-worker")
  State string          // RUNNING, STOPPED, FATAL, STARTING, etc.
  PID int               // process ID
  Uptime time.Duration  // how long running
}
```

**Supervisor Template Parameters** (supervisor-worker.conf.tmpl):
- ProgramName: e.g., "example.com-worker"
- ArtisanPath: Full path to artisan executable
- Connection: Queue connection type
- Queue: Queue name
- MaxTries: Retry attempts
- MaxTime: Seconds before restart
- Sleep: Seconds between failed job polls
- User: Linux user running worker
- NumProcs: Number of parallel processes
- PHPBinary: PHP version binary path
- StopWaitSecs: Graceful shutdown timeout (MaxTime + 60s buffer)

### internal/tui/screens/queues.go (174 lines)
**Purpose**: Queue worker management screen
- List workers with state, PID, uptime in table format
- Cursor selection: 'k'/'j' navigate, 's' start, 'x' stop, 'r' restart, 'd' delete
- Color-coded state: RUNNING (green), FATAL (red), STOPPED (yellow)
- Uptime formatting: "Xh Ym" or "Xm Ys"
- Messages: StartWorkerMsg, StopWorkerMsg, RestartWorkerMsg, DeleteWorkerMsg
- Empty state handling with help text
- Error display
- Escape to go back to dashboard

**Screen Title**: Queues

### internal/backup/manager.go (494 lines)
**Purpose**: Backup and restore lifecycle management with scheduling
- Manager struct wraps Executor, FileManager, Config, Database manager
- BackupType enum: Full (files + database), Files (site files only), Database (DB dump only)
- BackupInfo: Path, Domain, Type, Size, CreatedAt metadata
- Metadata struct embedded in archives: Domain, Type, ProjectType, PHPVersion, DBName, DBUser, SiteUser, CreatedAt

**Key Operations**:
- **Create()**: Packages site files (tar.gz) + DB dump (mysqldump.sql.gz) into archive
  - Temp directory aggregation then final compression
  - Metadata.toml written for cross-server portability
  - Archive permissions 0600 (security: contains DB dumps)
  - 15-minute timeout for large sites
- **Restore()**: Extracts files + restores database from archive
  - Path traversal validation (must be in backup directory)
  - Atomic file extraction to temp dir
  - Restores files with original ownership via chown
  - Imports SQL dump into target database
- **List()**: Returns all backups for domain sorted newest first
  - Parses filename pattern: `{domain}_{YYYYMMdd_HHmmss}.tar.gz`
  - Filters by domain, returns with size + creation time
- **Delete()**: Removes single backup file (path must be in backup dir)
- **Cleanup()**: Retention policy - keeps N most recent backups, deletes older
- **SetupCron()**: Writes `/etc/cron.d/juiscript-{domain}` cron job
  - Validation: 5-field cron schedule format, domain name safe characters
  - Prevents command injection via newline/special char regex
- **RemoveCron()**: Deletes cron job file (idempotent)

**Security**:
- Path traversal prevention: isInsideBackupDir validates all paths
- Domain validation: safeNameRegex enforces alphanumeric + dot/dash/underscore only
- Restrictive permissions: 0600 on archives, 0750 on directories
- Regex validation for cron schedules (no shell injection)

### internal/backup/manager_test.go (34 unit tests)
**Coverage**: Backup creation, restore, list, cleanup, cron setup, path validation, filename parsing

### internal/provisioner/detector.go (148 lines) - Phase 01: Package Detection
**Purpose**: System package detection for LEMP stack initialization
- PackageInfo struct: Name, DisplayName, Package, Installed, Version
- Detector struct: Wraps system.Executor for command abstraction
- DetectAll(): Returns status for all LEMP packages (Nginx, MariaDB, Redis, PHP versions)

**Key Methods**:
- **isInstalled(ctx, pkg)**: Uses dpkg-query to check package status and retrieve version
  - Parses dpkg output: Status line must contain "install ok installed"
  - Returns (installed bool, version string)
  - On error returns (false, "")
- **detectPHPVersions()**: Scans /etc/php/ directory for installed PHP versions
  - Validates directory names match X.Y format
  - Reuses isVersionDir validation helper
- **isVersionDir(name)**: Validates PHP version directory names (e.g., "8.3", "7.4")
  - Strict validation: exactly 2 numeric parts separated by dot
  - Rejects invalid formats: "", single numbers, decimals, non-numeric

**Detection Logic**:
- Static packages: Nginx, MariaDB, Redis (fixed list)
- Dynamic PHP: Scans /etc/php/ for directories, one PackageInfo per detected version
- Placeholder: If no PHP found, returns single "PHP" entry with Installed=false
- All checks via dpkg-query for reliable Ubuntu package detection

**Key Types**:
```go
PackageInfo {
  Name string        // "nginx", "mariadb", "redis", "php"
  DisplayName string // "Nginx", "MariaDB", "Redis", "PHP 8.3"
  Package string     // apt package name or "" for PHP placeholder
  Installed bool     // Detected via dpkg
  Version string     // e.g. "1.24.0-2ubuntu1" or ""
}
```

### internal/provisioner/detector_test.go (269 lines) - Phase 01
**Purpose**: 8 comprehensive unit tests for package detection
**Test Coverage**:
1. **isInstalled tests** (3 tests):
   - InstalledPackage: Parses version from dpkg output
   - NotInstalled: Handles dpkg-query failure (package not found)
   - BadStatus: Deinstalled/removed packages return installed=false
2. **DetectAll tests** (4 tests):
   - AllInstalled: All static packages detected + PHP placeholder
   - NoneInstalled: All packages show Installed=false
   - PartialInstall: Mixed installation state (nginx+redis, no mariadb)
   - CorrectDisplayNames: Validates display name mapping (nginx→"Nginx", etc.)
   - PHPPlaceholderWhenNotInstalled: Single "PHP" entry when no /etc/php/
3. **isVersionDir tests** (1 test - table-driven):
   - Valid: "8.3", "7.4", "8.0" → true
   - Invalid: "", "8", "abc", "8.3.1", ".3", "8." → false

**Mock Executor Pattern**:
- Captures executed commands for verification
- Supports configurable outputs per command
- Supports configurable error injection per command
- dpkgCmd() helper builds expected command strings for matching

### internal/provisioner/installer.go (196 lines) - Phase 02: Package Installation
**Purpose**: Idempotent apt-get installation with service management and MariaDB hardening
- InstallStatus enum: installed, skipped (already present), failed
- InstallResult struct: Package, Status, Message for operation outcomes
- installTimeout: 5 minutes per package

**Installer Struct**:
- executor: system.Executor for command execution
- phpMgr: Optional *php.Manager for PHP version delegation

**Key Methods**:
- **AptUpdate(ctx)**: Runs apt-get update with DEBIAN_FRONTEND=noninteractive
- **InstallNginx(ctx)**: Installs nginx, enables/starts service
- **InstallRedis(ctx)**: Installs redis-server, enables/starts service
- **InstallMariaDB(ctx)**: Installs mariadb-server, enables/starts, hardens database
  - Removes test database, anonymous users, remote root access via SQL
  - Keeps unix_socket auth for local root (matches existing DB pattern)
- **InstallPHP(ctx, version)**: Delegates to php.Manager.InstallVersion(ctx, version)
- **installSimplePackage(ctx, pkg, serviceName)**: Common pattern - idempotent install
  - Checks package status via dpkg-query
  - Skips if already installed (StatusSkipped)
  - Installs via apt-get with force-conf flags
  - Enables and starts service via systemctl
- **aptInstall(ctx, pkg)**: Noninteractive apt-get with lock timeout
  - Flags: DEBIAN_FRONTEND=noninteractive, --force-confdef, --force-confold
  - DPkg::Lock::Timeout=120 (handles concurrent apt operations)
- **enableAndStart(ctx, service)**: systemctl enable + start (idempotent)
- **hardenMariaDB(ctx)**: SQL hardening via mysql CLI
  - DELETE anonymous users and remote root access
  - DROP test database
  - FLUSH PRIVILEGES to apply changes

**Idempotency**: All methods skip if package already installed (dpkg-query validation)

**Shared Function**:
- **isPackageInstalled(ctx, executor, pkg)**: Checks package status via dpkg-query
  - Returns (installed bool, version string)
  - Parses dpkg output for status + version
  - Shared with detector.go (DRY pattern)

### internal/provisioner/installer_test.go (198 lines) - Phase 02
**Purpose**: 12 comprehensive unit tests for Installer package operations
**Test Coverage**:

1. **AptUpdate tests** (2 tests):
   - Success: Executes apt-get update with DEBIAN_FRONTEND=noninteractive
   - Failure: Propagates network errors from apt-get

2. **InstallNginx tests** (3 tests):
   - Success: Installs, enables, and starts nginx service
   - AlreadyInstalled: Skips installation (StatusSkipped)
   - AptFailure: Returns StatusFailed with error wrapping

3. **InstallRedis tests** (1 test):
   - Success: Installs redis-server and starts service

4. **InstallMariaDB tests** (3 tests):
   - Success: Installs, enables, starts, and hardens MariaDB
     - Verifies DELETE/DROP/FLUSH SQL commands executed
     - Checks removal of anonymous users and test database
   - AlreadyInstalled: Skips installation (StatusSkipped)
   - HardeningFailure: Returns StatusFailed with "hardening failed" message

5. **InstallPHP tests** (1 test):
   - NilManager: Returns StatusFailed when php.Manager not available

6. **aptInstallCmd() helper**:
   - Builds expected apt-get command string with all flags for mock matching
   - Used across multiple tests for command verification

**Mock Executor Pattern**:
- Captures all executed commands for assertion
- Supports per-command output configuration
- Supports per-command failure injection
- lastInput field captures stdin for SQL command verification

### internal/tui/screens/backup.go (varies)
**Purpose**: Backup management screen
- Display: Table with DOMAIN | SIZE | CREATED columns
- List all backups for selected site
- Keyboard: 'k'/'j' navigate, 'c' create, 'r' restore, 'd' delete, 'esc' back
- Color-coded display with formatted sizes (KB/MB/GB)
- Messages: CreateBackupMsg, RestoreBackupMsg, DeleteBackupMsg for app routing
- Empty state with help text

**Screen Title**: Backup

### internal/tui/theme/theme.go (UPDATED)
**Addition**: WarnText style for warning states (amber color)
- Used by QueuesScreen for STOPPED worker display
- Consistent with existing theme palette

## Future Additions

- Service control screens (stop/start/restart)
- System monitoring and health checks
- Wildcard certificate support
