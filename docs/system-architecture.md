# System Architecture

## High-Level Overview

juiscript is a single-binary LEMP management tool with three layers:

```
┌─────────────────────────────────┐
│  Cobra CLI / Bubble Tea TUI      │ User Interface
├─────────────────────────────────┤
│ Site / Nginx / PHP / DB / SSL    │ Domain Logic
├─────────────────────────────────┤
│ System / Config / Template       │ OS Abstractions
└─────────────────────────────────┘
```

## Component Architecture

### Layer 1: Entry Point

**cmd/juiscript/main.go**
- Cobra CLI root with version command
- Launches Bubble Tea TUI as default action
- Injects version/commit from build-time ldflags

```
juiscript              → TUI (default)
suiscript version      → Print version
```

### Layer 2: User Interface

**internal/tui/app.go** (Root Model)
- Screen router for TUI
- Manages current screen state (Dashboard, Sites, Nginx, etc)
- Delegates updates and views to active screen
- Keyboard navigation: 'j'/'k' move, 'enter' select, 'q' quit
- Screen transitions via NavigateMsg/GoBackMsg

**internal/tui/screens/** (Full-Screen Views)
- `dashboard.go`: Main menu with 8 feature links
- Other screens implement Bubble Tea Model interface

**internal/tui/components/** (Reusable Parts)
- `header.go`: App title and current screen name
- `statusbar.go`: Keyboard shortcuts at bottom
- `theme.go`: Centralized colors and styles (Lip Gloss)

### Layer 3: System Abstractions

**internal/system/**

Three core interfaces for testability (no root in tests):

1. **Executor** - Command execution with logging
   ```go
   Run(ctx, "systemctl", "restart", "nginx")
   RunWithInput(ctx, password, "chpasswd", "-e")
   ```
   - Wraps os/exec with context, timeouts, audit logging
   - Default timeout: 30 seconds
   - Logs all commands (cmd, args, duration, exit code)

2. **FileManager** - Safe filesystem operations
   ```go
   WriteAtomic(path, data, 0640)    // Temp file + atomic rename
   Symlink(target, link)             // Safe symlink creation
   RemoveSymlink(path)               // Validate before delete
   Remove(path)                      // Delete file/directory
   Exists(path), ReadFile(path)      // Info operations
   ```
   - Atomic writes prevent config corruption
   - Symlink ops validate type before deletion
   - Restrictive permissions enforced

3. **UserManager** - Linux user isolation
   ```go
   Create(username, homeDir)        // Create site user
   Delete(username)                  // Remove and cleanup
   Exists(username), LookupUID()     // Info queries
   ```
   - Each site gets dedicated Linux user
   - Backed by useradd/userdel commands
   - Returns UID/GID for permission setup

### Layer 4: Configuration & Templates

**internal/config/**
- `config.go`: TOML config structure with defaults
- Default paths: `/etc/juiscript/config.toml`
- Sections: General, Nginx, PHP, Database, Backup, Redis
- Load: Fallback to defaults if file missing
- Save: Create parent dirs, restrictive permissions (0640)
- EnsureDirs: Guarantee required directories exist

**internal/template/**
- `engine.go`: Template engine with embedded files
- Embedded via `//go:embed templates/*`
- Parse all templates at startup (fail-fast validation)
- Render with custom data struct per template
- Available: List all loaded template names

**templates/** (Embedded)
- `nginx.vhost.tmpl`: Virtual host config generation
- `php-fpm.pool.tmpl`: PHP-FPM pool config
- `supervisor.worker.tmpl`: Queue worker config
- All embedded in binary (no file dependency)

### Layer 5: Domain Logic

**Implemented Packages**:

1. **nginx/** (Phase 03 - Complete)
   ```go
   Manager {
     Create(cfg VhostConfig) error        // Render template, write, enable, test, reload
     Delete(domain string) error           // Remove config, disable, reload
     Enable(domain string) error           // Enable symlink, test, reload
     Disable(domain string) error          // Disable symlink, reload
     Test() error                          // nginx -t with error parsing
     Reload() error                        // systemctl reload nginx
     List() ([]VhostInfo, error)          // All vhosts with status
   }
   ```
   - Supports Laravel and WordPress templates
   - SSL configuration support
   - Atomic operations with rollback on failure
   - Comprehensive config validation
   - Error recovery with detailed messages

2. **site/** (Phase 02 - Partial)
   - Site creation/deletion lifecycle
   - Integrates with nginx.Manager for vhost setup
   - Linux user account management
   - Metadata storage

**Planned Packages**:

- **php/**: Multi-version support, pool management per site
- **database/**: MariaDB user/database operations
- **ssl/**: Let's Encrypt integration via certbot
- **backup/**: Full/partial backups, retention, restore
- **service/**: systemctl wrapper (Nginx, PHP-FPM, MariaDB, Redis)
- **supervisor/**: Queue worker management

## Data Flow

### Vhost Creation Sequence (Phase 03)
```
TUI Nginx Screen [Triggered from Site Manager]
    ↓
nginx.Manager.Create(VhostConfig)
    ↓
Template.Render [Select template based on ProjectType]
    ↓
FileManager.WriteAtomic [Write config to sites-available]
    ↓
FileManager.Symlink [Enable in sites-enabled]
    ↓
Manager.Test() [nginx -t validation]
    ↓
On Failure: Rollback (disable + remove)
    ↓
Manager.Reload() [systemctl reload nginx]
    ↓
Success: Config live on all workers
```

### Site Creation Sequence (Phase 02)
```
TUI Screen [Input site name, domain, PHP version]
    ↓
Site Service [Validate, check conflicts]
    ↓
UserManager.Create [Create Linux user account]
    ↓
nginx.Manager.Create [Write Nginx vhost config]
    ↓
Config.Save [Update config metadata]
```

### Configuration Loading
```
App Start
    ↓
Config.Load("/etc/juiscript/config.toml")
    ↓
If missing → Use Default()
    ↓
Template.Engine.New() [Parse all templates]
    ↓
Initialize Executor, FileManager, UserManager
    ↓
Launch TUI
```

## Key Design Patterns

### Interface-Based Abstraction
Every OS operation behind interface:
```go
type Operation interface { Execute() error }
```
Enables mocking, testing without root, dependency injection.

### Atomic File Operations
Critical configs use atomic writes:
```go
1. Write to temp file in same directory
2. Set permissions
3. Atomic rename to target
→ Zero chance of partial updates on crash
```

### Transaction-Like Vhost Creation
Nginx manager ensures consistent state:
```go
1. Render template from VhostConfig
2. WriteAtomic to sites-available
3. Enable symlink to sites-enabled
4. Test with `nginx -t`
   If test fails → Rollback (disable + remove file)
5. Reload only if all steps succeed
→ Live config always valid, or unchanged if error
```

### Error Wrapping
Always wrap with context:
```go
return fmt.Errorf("operation failed: %w", err)
```
Preserves error chain, helps debugging.

### Embedded Static Assets
All templates compiled into binary:
```go
//go:embed templates/*
var templateFS embed.FS
```
Single executable, no file dependencies. Vhost templates choose framework variant.

## Security Architecture

### User Isolation
- Each site has dedicated Linux user (not www-data)
- Prevents cross-site access
- File permissions restrict access to owner

### File Permissions
- Configs: 0640 (owner r/w, group r, others none)
- Dirs: 0750 (owner r/w/x, group r/x, others none)
- Home dirs: 0750 (owner has full access)

### Command Execution
- Context timeouts prevent hangs
- All commands logged for audit
- Error output captured separately
- Exit codes validated

### Root Requirement
- CLI enforces root-only execution
- Prevents accidental privilege escalation
- Clear error if non-root attempted

## Deployment Architecture

### Single Binary Model
```
Build Phase:
  Go source + embedded templates → Static ELF binary

Deployment:
  scp binary to /usr/local/bin/juiscript
  chmod +x /usr/local/bin/juiscript
  sudo juiscript      ← Launches TUI

Result:
  - Zero runtime dependencies
  - Works on any Ubuntu 22/24 box with systemd
  - No installation required beyond binary
```

### Configuration & Template Model
```
/etc/juiscript/
├── config.toml          ← User config (optional, loads defaults)
├── sites/               ← Site metadata per site
│   ├── example.com.json
│   └── blog.io.json
└── ssl/                 ← SSL certs (future)
    └── example.com/

/etc/nginx/
├── sites-available/     ← All vhost configs
│   ├── example.com.conf
│   └── blog.io.conf
└── sites-enabled/       ← Enabled vhosts (symlinks)
    ├── example.com.conf → ../sites-available/example.com.conf
    └── blog.io.conf → ../sites-available/blog.io.conf

Templates (Embedded):
├── nginx-laravel.conf.tmpl      ← Laravel-specific Nginx config
├── nginx-wordpress.conf.tmpl    ← WordPress-specific Nginx config
├── nginx-ssl.conf.tmpl          ← SSL certificate configuration
├── php-fpm-pool.conf.tmpl       ← PHP-FPM pool per site
└── supervisor-worker.conf.tmpl  ← Queue worker process management
```

## Scalability Considerations

### Current
- Suitable for single server with 1-100 sites
- TUI responsive with < 200ms latency
- All operations async where possible

### Future
- Multi-server management (future roadmap)
- CLI-only mode for automation
- API for programmatic access
- Batch operations for bulk site management

## Testing Strategy

### Unit Tests (internal/system/*, internal/config/*)
- No root required (interfaces mocked)
- Table-driven tests for coverage
- Target: 70%+ coverage

### Integration Tests (future)
- Docker containers for Ubuntu environment
- Real command execution in isolated VMs
- Test backup/restore, SSL, DB operations

### Manual QA (future)
- Ubuntu 22.04 and 24.04 both tested
- Real Nginx, PHP, MariaDB deployments
- Create/delete 10+ sites, verify isolation

## Performance Targets

- CLI startup: < 100ms
- TUI first paint: < 200ms
- Screen transitions: < 50ms
- Config load/save: < 10ms
- Site creation: < 10s (depends on system)
- Binary size: ~50MB (statically compiled)

## Logging & Monitoring

### Audit Log
- Location: `/var/log/juiscript.log` (configurable)
- Every command execution logged with duration
- Format: Structured JSON via slog
- Retention: 30 days (configurable)

### Future: Monitoring
- Health check endpoints
- Metrics export for Prometheus
- Alert integration with monitoring systems
