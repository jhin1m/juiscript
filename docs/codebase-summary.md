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
│   ├── site/
│   │   └── manager.go          # Site lifecycle (uses nginx.Manager)
│   └── tui/
│       ├── app.go              # Root model & screen router
│       ├── components/
│       │   ├── header.go       # Header component
│       │   ├── statusbar.go    # Status bar component
│       │   └── theme/
│       │       └── theme.go    # Color scheme & styles
│       └── screens/
│           ├── dashboard.go    # Dashboard screen
│           └── nginx.go        # Nginx vhost management screen
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

**Phase 01 - Infrastructure**: Config, system abstractions, template engine, basic TUI ✓
**Phase 02 - Site Management**: Site lifecycle manager, site creation/deletion ✓
**Phase 03 - Nginx/Vhost**: Manager CRUD, templates, TUI screen, enable/disable ✓

## Future Additions

- PHP-FPM pool per site
- MariaDB user/database management
- SSL certificate automation via certbot
- Backup scheduling & execution
- Supervisor queue worker management
- Service control screens (stop/start/restart)
- System monitoring and health checks
