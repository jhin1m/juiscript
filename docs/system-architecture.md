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

**cmd/juiscript/main.go** (Phase 1: Manager Injection)
- Cobra CLI root with version command
- Launches Bubble Tea TUI as default action
- Injects version/commit from build-time ldflags
- Phase 1: Constructs all 9 backend managers and passes via AppDeps
- Manager construction order:
  1. System abstractions: Executor, FileManager, UserManager
  2. Core managers: PHPManager, ServiceManager, Provisioner
  3. Domain managers: NginxManager, DatabaseManager, SiteManager, SSLManager, SupervisorManager, BackupManager
- All managers initialized with appropriate dependencies
- Passes cfg + AppDeps to tui.NewApp for injection

```
juiscript              → TUI (default) with all managers wired
juiscript version      → Print version
```

### Layer 2: User Interface

**internal/tui/app.go** (Root Model - Phase 1: Backend Wiring)
- Screen router for TUI with injected backend managers
- AppDeps struct: Encapsulates all 9 backend managers for dependency injection
- All managers are optional (nil-safe) for graceful degradation
- App struct fields:
  - `svcMgr` (service.Manager): Service control
  - `prov` (provisioner.Provisioner): Package detection & installation
  - `phpMgr` (php.Manager): PHP version management
  - `siteMgr` (site.Manager): Site lifecycle
  - `nginxMgr` (nginx.Manager): Virtual host management
  - `dbMgr` (database.Manager): Database operations
  - `sslMgr` (ssl.Manager): SSL certificate management
  - `supervisorMgr` (supervisor.Manager): Queue worker management
  - `backupMgr` (backup.Manager): Backup/restore operations
- NewApp(cfg, AppDeps) constructor for clean initialization
- Keyboard navigation: 'j'/'k' move, 'enter' select, 'q' quit
- Screen transitions via NavigateMsg/GoBackMsg
- Async operation pattern via result message types

**internal/tui/app_messages.go** (Phase 1: Result Messages)
- 28 result/error message types for async backend operations
- Site operations: SiteListMsg, SiteListErrMsg, SiteCreatedMsg, SiteDetailMsg, SiteOpDoneMsg, SiteOpErrMsg
- Nginx operations: VhostListMsg, VhostListErrMsg, NginxOpDoneMsg, NginxOpErrMsg, NginxTestOkMsg
- Database operations: DBListMsg, DBListErrMsg, DBOpDoneMsg, DBOpErrMsg
- SSL operations: CertListMsg, CertListErrMsg, SSLOpDoneMsg, SSLOpErrMsg
- Service operations: ServiceOpDoneMsg, ServiceOpErrMsg (StatusMsg already in app.go)
- Queue operations: WorkerListMsg, WorkerListErrMsg, QueueOpDoneMsg, QueueOpErrMsg
- Backup operations: BackupListMsg, BackupListErrMsg, BackupOpDoneMsg, BackupOpErrMsg
- Pattern: Each operation has success (msg) and failure (errMsg) variants

**internal/tui/screens/** (Full-Screen Views - Phase 5-6: Forms + Feedback Complete)
- `dashboard.go`: Main menu with 8 feature links
- `php.go`: Version list with install/remove forms, spinner for install, confirmation for remove
- `database.go`: DB list with create/import forms, confirmation for drop
- `ssl.go`: Certificate list with obtain form, spinner for certbot, confirmation for revoke
- `backup.go`: Backup list with create form, spinners for create/restore, confirmation for delete/restore
- `sitedetail.go`: Site details with confirmation for delete
- All screens implement Bubble Tea Model interface with form/spinner/confirmation integration

**internal/tui/components/** (Reusable Parts - Phase 5-6 Complete)
- `header.go`: App title and current screen name
- `statusbar.go`: Keyboard shortcuts at bottom
- `form.go` (Phase 5-6): Generic step-by-step form component (FieldText, FieldSelect, FieldConfirm) - fully integrated
- `confirm.go` (Phase 5-6): Modal confirmation dialog for destructive actions
- `spinner.go` (Phase 5-6): Loading spinner for long-running operations
- `toast.go` (Phase 5-6): Toast notifications for operation results (success/error)
- `service-status-bar.go`: Horizontal health indicator for LEMP services (Phase 01)
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

0. **provisioner/** (Phase 01 - Complete: Detector; Phase 02 - Complete: Installer)

   **Detector**:
   ```go
   Detector {
     DetectAll(ctx) → ([]PackageInfo, error)  // Detect all LEMP packages
   }
   ```
   - Scans system for installed packages: Nginx, MariaDB, Redis, PHP versions
   - Uses dpkg-query for Ubuntu package status
   - Dynamic PHP version detection from /etc/php/ filesystem scan
   - Returns comprehensive package information with installed status and versions
   - Used for system health checks and provisioning decisions

   **Installer** (Phase 02):
   ```go
   Installer {
     AptUpdate(ctx) error
     InstallNginx(ctx) → (*InstallResult, error)
     InstallRedis(ctx) → (*InstallResult, error)
     InstallMariaDB(ctx) → (*InstallResult, error)    // + hardening
     InstallPHP(ctx, version) → (*InstallResult, error)
   }
   ```
   - Idempotent package installation via apt-get
   - Service management: systemctl enable + start
   - MariaDB hardening: Removes test DB, anonymous users, remote root access
   - Shared isPackageInstalled function with Detector (DRY)
   - 5-minute timeout per package install
   - Returns InstallResult with status (installed/skipped/failed)

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

3. **php/** (Phase 04 - Complete)
   ```go
   Manager {
     EnsurePPA() error                                    // Add ondrej/php PPA
     InstallVersion(ctx, version string) error           // Install PHP + extensions
     RemoveVersion(ctx, version string, sites []string) error  // Remove if no sites use it
     ListVersions(ctx) ([]VersionInfo, error)            // All versions with FPM status
     ReloadFPM(ctx, version string) error                // Reload FPM service
     CreatePool(ctx, cfg PoolConfig) error               // Create per-site FPM pool
     DeletePool(ctx, domain, version string) error       // Remove pool config
     SwitchVersion(ctx, cfg PoolConfig, fromVersion string, reloadFn) error
   }
   ```
   - Multi-version PHP support (ondrej/php PPA)
   - Dynamic FPM process manager per site
   - Zero-downtime version switching with rollback
   - Per-site user/group isolation
   - Security constraints (open_basedir, extension restrictions)

7. **service/** (Phase 07 - Complete)
   ```go
   Manager {
     Start(ctx, name ServiceName) error            // systemctl start
     Stop(ctx, name ServiceName) error             // systemctl stop
     Restart(ctx, name ServiceName) error          // systemctl restart
     Reload(ctx, name ServiceName) error           // systemctl reload (graceful)
     IsActive(ctx, name ServiceName) bool          // Check if running
     Status(ctx, name ServiceName) (*Status, error) // Detailed status
     ListAll(ctx) ([]Status, error)                // All LEMP services
     IsHealthy(ctx) bool                           // Critical services check
   }
   ```
   - Whitelist-based security (no shell injection)
   - Dynamic PHP-FPM detection from /etc/php/
   - Service health monitoring for app startup
   - Memory usage and PID tracking
   - Graceful reload support (nginx, php-fpm)

8. **supervisor/** (Phase 08 - Complete)
   ```go
   Manager {
     Create(ctx, cfg WorkerConfig) error    // Render template, write config, reload
     Delete(ctx, domain string) error       // Remove config, reload
     Start(ctx, domain string) error        // Start worker processes
     Stop(ctx, domain string) error         // Stop worker processes
     Restart(ctx, domain string) error      // Restart worker processes
     Status(ctx, domain string) (*WorkerStatus, error) // Worker state & uptime
     ListAll(ctx) ([]WorkerStatus, error)   // All workers status
   }
   ```
   - Supervisor-managed Laravel queue workers
   - Per-site configuration with supervisorctl automation
   - Worker process management and monitoring
   - Uptime tracking and state reporting
   - Atomic config writes with rollback on reload failure

9. **backup/** (Phase 09 - Complete)
   ```go
   Manager {
     Create(ctx, opts Options) (*BackupInfo, error)     // Create full/partial backup archive
     Restore(ctx, backupPath, domain string) error      // Restore from archive
     List(domain string) ([]BackupInfo, error)          // All backups for domain
     Delete(backupPath string) error                    // Remove backup file
     Cleanup(domain string, keepLast int) error         // Retention policy
     SetupCron(domain, schedule string) error           // Scheduled backups via cron
     RemoveCron(domain string) error                    // Remove cron job
   }
   ```
   - Full site backups (files + database)
   - Partial backups (files-only or database-only)
   - Portable archives with embedded metadata
   - Retention policy with cleanup automation
   - Cron-based scheduled backups
   - Security: path traversal prevention, domain validation, restrictive permissions

## Data Flow

### PHP-FPM Pool Creation Sequence (Phase 04)
```
TUI PHP Screen OR Site Manager [CreatePool(PoolConfig)]
    ↓
Validate: PHPVersion format, Domain path traversal check
    ↓
Apply defaults for zero values (MaxChildren=5, etc)
    ↓
Map PoolConfig → poolTemplateData
    ↓
Template.Render [php-fpm-pool.conf.tmpl with data]
    ↓
FileManager.WriteAtomic [Write to /etc/php/{ver}/fpm/pool.d/{domain}.conf]
    ↓
Manager.ReloadFPM [systemctl reload php{ver}-fpm]
    ↓
Success: Pool serving with new socket path
```

### PHP Version Switch Sequence (Phase 04 - Zero Downtime)
```
TUI PHP Screen [SwitchVersion(cfg, fromVersion, reloadNginx)]
    ↓
Validate: fromVersion ≠ toVersion
    ↓
Step 1: CreatePool [Create new pool with target PHP version]
    ↓
Step 2: ReloadNginx [Caller updates Nginx vhost to new socket]
    ↓
On Failure → Rollback: DeletePool(cfg.SiteDomain, cfg.PHPVersion)
    ↓
Step 3: DeletePool [Remove old pool config]
    ↓
Success: Site now on new PHP version, zero downtime
```

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
│   - User/group isolation per site
│   - Dynamic process manager (pm=dynamic)
│   - Socket at /run/php/php{version}-fpm-{user}.sock
│   - Security: open_basedir, upload restrictions, extension limits
│   - Logging: php-error.log per site
└── supervisor-worker.conf.tmpl  ← Queue worker process management
    - Program name per site domain
    - Multiple process instances with supervisor group notation
    - Graceful shutdown timeout (MaxTime + 60s buffer)
    - Connection type: redis/database/sqs
    - Queue name and retry/sleep configuration
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

## Supervisor Queue Worker Implementation (Phase 08)

### Worker Configuration Flow
```
WorkerConfig {Domain, Username, SitePath, PHPBinary, Connection, Queue, Processes, Tries, MaxTime, Sleep}
    ↓
applyDefaults (Connection: redis, Queue: default, Processes: 1, etc.)
    ↓
templateData mapping → supervisor-worker.conf.tmpl
    ↓
supervisorctl reread + update
    ↓
Processes started/managed by Supervisor
```

### Configuration Path
- File: `/etc/supervisor/conf.d/{domain}-worker.conf`
- Program name: `{domain}-worker`
- Processes: `{domain}-worker:00`, `{domain}-worker:01`, etc. (group notation)
- Graceful shutdown: MaxTime + 60 seconds buffer

### Worker State Management
- **RUNNING**: Process active and healthy
- **STOPPED**: Process manually stopped or removed config
- **FATAL**: Process failed repeatedly (exceeded retry limit)
- **STARTING**: Process initializing (transient state)

### Status Parsing
- Parses `supervisorctl status` output
- Extracts: program name, state, PID, uptime (H:MM:SS format)
- Uptime conversion: Hours + Minutes + Seconds → time.Duration
- Multi-process workers: Returns first process as representative

### Integration with TUI
- QueuesScreen displays workers in table: NAME | STATE | PID | UPTIME
- Color-coded states: RUNNING (green), FATAL (red), STOPPED (yellow)
- Actions: start, stop, restart, delete
- Reload timing: ~2s for supervisor to apply changes

## Backup System Implementation (Phase 09)

### Archive Structure
```
backup-archive.tar.gz
├── metadata.toml          ← Domain, type, PHP version, DB creds
├── files.tar.gz           ← Site files (if full or files-only)
└── database.sql.gz        ← Database dump (if full or database-only)
```

### Backup Types
- **BackupFull**: Files + database (complete site snapshot)
- **BackupFiles**: Site files only (public_html, config, etc.)
- **BackupDatabase**: Database dump only (data snapshot)

### Backup Flow (Create)
```
Manager.Create(domain, type)
    ↓
Load site metadata → PHP version, DB name, site user
    ↓
Step 1: Database export (if full/database)
  → mysqldump {domain} | gzip → database.sql.gz
    ↓
Step 2: Files archive (if full/files)
  → tar czf files.tar.gz -C {site-root} .
    ↓
Step 3: Write metadata.toml
  → Stores domain, type, PHP version, DB user for restore
    ↓
Step 4: Final archive
  → tar czf {domain}_{timestamp}.tar.gz {temp-contents}
    ↓
Permissions 0600 (restrictive: contains DB dumps)
```

### Restore Flow
```
Manager.Restore(backup-path, domain)
    ↓
Path validation (must be in backup directory)
    ↓
Extract to temp dir
    ↓
Read metadata.toml
    ↓
Step 1: Restore files (if archive contains)
  → tar xzf files.tar.gz -C {site-root}
  → chown -R {site-user} {site-root}
    ↓
Step 2: Restore database (if archive contains)
  → gunzip database.sql.gz | mysql {db-name}
```

### Cron Scheduling
- Location: `/etc/cron.d/juiscript-{domain}`
- Validation: 5-field format (`minute hour day month weekday`)
- Prevents injection: Regex whitelist + command validation
- Example: `0 2 * * *` = Daily at 2 AM full backup

### Retention & Cleanup
- Keep N most recent backups per domain
- Older backups automatically deleted
- Idempotent: Safe to run multiple times

### Security Model
- Path traversal: isInsideBackupDir validates all operations
- Domain validation: safeNameRegex (alphanumeric + dot/dash/underscore)
- Archive permissions: 0600 (readable only by backup user)
- Directory permissions: 0750
- Timeout: 15 minutes for large sites

## Package Installation Implementation (Phase 02)

### Installation Strategy
```
Installer struct (Executor + optional php.Manager)
    ↓
For each package (Nginx, Redis, MariaDB, PHP):
    ↓
Check if already installed via dpkg-query
    ↓
If installed → Return StatusSkipped
    ↓
If not → apt-get install with noninteractive flags
    ↓
systemctl enable + start service
    ↓
Special: MariaDB → Run SQL hardening after startup
    ↓
Return InstallResult (status + message)
```

### Idempotency Pattern
All installation methods are idempotent:
- Check package status first (dpkg-query)
- Skip if already installed (no-op)
- Install only if needed
- Can safely re-run multiple times
- Useful for infrastructure automation

### apt-get Configuration
- **Environment**: DEBIAN_FRONTEND=noninteractive (no prompts)
- **Flags**:
  - `--force-confdef`: Use default for changed config files
  - `--force-confold`: Keep existing config files (don't ask)
  - `DPkg::Lock::Timeout=120`: Wait up to 120s for apt lock
- **Timeout**: 5 minutes per package install
- **Error handling**: Wraps errors with context

### MariaDB Hardening
Executed after successful installation via `mysql --user=root`:
```sql
DELETE FROM mysql.user WHERE User='';
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');
DROP DATABASE IF EXISTS test;
DELETE FROM mysql.db WHERE Db='test' OR Db='test\_%';
FLUSH PRIVILEGES;
```

**Security improvements**:
- Removes anonymous login accounts
- Removes remote root access (keeps local socket auth only)
- Deletes test database
- Applies changes immediately

### Shared isPackageInstalled Function
Located in installer.go, used by both Detector and Installer:
```go
isPackageInstalled(ctx, executor, pkg) → (bool, version)
  ↓
dpkg-query -W --showformat='${Status}\n${Version}' {pkg}
  ↓
Parse Status line: must contain "install ok installed"
  ↓
Extract Version from second line if installed
  ↓
Return (installed, version)
```

Benefits:
- Single source of truth for package status
- Both Detector (detection) and Installer (installation check) reuse same logic
- Consistent behavior across codebase

### PHP Installation Delegation
When Installer.InstallPHP called:
```
InstallPHP(ctx, version)
    ↓
Check if phpMgr is nil → Return error if unavailable
    ↓
Delegate to php.Manager.InstallVersion(ctx, version)
    ↓
PHP Manager handles: PPA setup, version-specific packages, extensions
    ↓
Return InstallResult with status from PHP Manager
```

Rationale: PHP has complex multi-version setup (PPA, extensions, FPM pools)
→ Leverage existing php.Manager rather than duplicate logic

### Service Management
After each apt install (except PHP):
```
systemctl enable {service}   ← Start on boot
systemctl start {service}    ← Start immediately
```

Service mappings:
- nginx → "nginx"
- redis-server → "redis-server"
- mariadb-server → "mariadb"

### Test Coverage (12 tests)
1. **AptUpdate** (2 tests): Success, network failure
2. **InstallNginx** (3 tests): Success, already-installed, apt-failure
3. **InstallRedis** (1 test): Success case
4. **InstallMariaDB** (3 tests): Success (with hardening), already-installed, hardening-failure
5. **InstallPHP** (1 test): nil manager error handling
6. **Mock patterns**: Command verification, error injection, SQL input validation

## Package Detection Implementation (Phase 01)

### Detection Strategy
```
System Package Detection
    ↓
Static Packages: Nginx, MariaDB, Redis (fixed list)
    ↓
dpkg-query for each package → [Installed, Version]
    ↓
Dynamic PHP: Scan /etc/php/ for versions (X.Y format)
    ↓
For each PHP version: Check php{version}-fpm package
    ↓
Return: Consolidated PackageInfo list
```

### Detection Methods
- **dpkg-query**: Ubuntu package status and version
  - Command: `dpkg-query -W --showformat='${Status}\n${Version}' {package}`
  - Status validation: Must contain "install ok installed"
  - Returns version if installed, empty string if not

- **Filesystem Scan**: PHP version discovery
  - Directory: `/etc/php/`
  - Pattern: Version directories match X.Y (e.g., "8.3", "7.4")
  - Creates PackageInfo for each detected version
  - Validates directory name format strictly

### Error Handling
- dpkg-query failure → Package treated as not installed
- /etc/php/ missing/unreadable → Returns PHP placeholder
- Returns (false, "") for any package detection error
- No exceptions thrown, graceful degradation

## TUI Input Forms Implementation (Phases 5-6)

### Form Input Workflow

**Phase 5 - Form Wiring**: Screens embed FormModel for data collection
```
User presses action key (i=install, c=create, o=obtain)
    ↓
Screen creates form with fields and displays it
    ↓
User fills form fields (text input, dropdown select)
    ↓
FormSubmitMsg emitted with collected values
    ↓
Screen converts form data → operation message
    ↓
App receives message and calls handler with parameters
    ↓
Handler executes backend manager operation
```

**Phase 6 - Feedback**: Confirmation, spinner, toast complete flow
```
Handler starts async operation
    ↓
Screen shows spinner ("Installing PHP 8.3...")
    ↓
Operation completes in background
    ↓
App receives OpDoneMsg or OpErrMsg
    ↓
Toast notification shows result (success or error)
    ↓
Screen updates with refreshed data
```

### Form Fields per Screen

**PHP Screen** (internal/tui/screens/php.go)
- Install form: `version` (FieldSelect) → InstallPHPMsg{Version}
- Keypress: `i` to install, `r` to remove (with confirmation)

**Database Screen** (internal/tui/screens/database.go)
- Create form: `name` (FieldText, validateDBName) → CreateDBMsg{Name}
- Import form: `path` (FieldText, validateFilePath) → ImportDBMsg{Name, Path}
- Keypress: `c` to create, `i` to import, `d` to drop (with confirmation)

**SSL Screen** (internal/tui/screens/ssl.go)
- Obtain form: `domain`, `email` (FieldText) → ObtainCertMsg{Domain, Email}
- Spinner during certbot execution
- Keypress: `o` to obtain, `r` to revoke (with confirmation)

**Backup Screen** (internal/tui/screens/backup.go)
- Create form: `domain` (FieldText), `type` (FieldSelect: full/files/database) → CreateBackupMsg{Domain, Type}
- Restore: Selected from list, confirmation required
- Spinners for create and restore operations
- Keypress: `c` to create, `r` to restore (confirm), `d` to delete (confirm)

### Feedback Components

**Toast Notifications** (app.go)
- Single instance in App struct
- Shows for all OpDoneMsg/OpErrMsg (operation results)
- Auto-dismisses after 3 seconds
- Types: ToastSuccess (green), ToastError (red)
- Examples: "PHP 8.3 installed", "Database 'mydb' dropped", "Error: Domain exists"

**Spinners** (php.go, ssl.go, backup.go)
- Per-screen instance for long operations
- Animated loading indicator while operation in progress
- Stopped when app receives operation result
- Shows contextual message: "Installing PHP 8.3...", "Creating backup..."

**Confirmations** (php.go, database.go, ssl.go, backup.go, sitedetail.go)
- Gate destructive actions: remove PHP, drop DB, delete site, revoke cert, delete/restore backup
- Modal dialog with [Y/n] prompt
- Prevents accidental data loss
- Examples:
  - "Remove PHP 8.2? Sites using it will break."
  - "Drop database 'olddb'? This cannot be undone."
  - "Delete backup 'site_2026-03-06.tar.gz'?"

### Message Type Updates (Phase 5)

| Message | Phase 5 Fields | Sender | Receiver |
|---------|---|---|---|
| `InstallPHPMsg` | `Version string` | PHP screen form → App | app_handlers_php |
| `RemovePHPMsg` | `Version string` | PHP confirm → App | app_handlers_php |
| `CreateDBMsg` | `Name string` | DB screen form → App | app_handlers_db |
| `ImportDBMsg` | `Name, Path string` | DB screen form → App | app_handlers_db |
| `DropDBMsg` | `Name string` | DB confirm → App | app_handlers_db |
| `ObtainCertMsg` | `Domain, Email string` | SSL form → App | app_handlers_ssl |
| `RevokeCertMsg` | `Domain string` | SSL confirm → App | app_handlers_ssl |
| `CreateBackupMsg` | `Domain, Type string` | Backup form → App | app_handlers_backup |
| `RestoreBackupMsg` | `Path, Domain string` | Backup confirm → App | app_handlers_backup |
| `DeleteBackupMsg` | `Path string` | Backup confirm → App | app_handlers_backup |
| `DeleteSiteMsg` | `Domain string` | SiteDetail confirm → App | app_handlers_site |

### Handler Implementation (Phase 5)

All handlers converted from stubs to real implementations:

**app_handlers_php.go**
```go
func (a *App) handleInstallPHP(version string) tea.Cmd {
    return func() tea.Msg {
        // a.phpMgr.InstallVersion(ctx, version)
        return PHPInstalledMsg{Version: version}
    }
}
```

**app_handlers_db.go**
```go
func (a *App) handleCreateDB(name string) tea.Cmd { /* ... */ }
func (a *App) handleImportDB(name, path string) tea.Cmd { /* ... */ }
func (a *App) handleDropDB(name string) tea.Cmd { /* ... */ }
```

**app_handlers_ssl.go**
```go
func (a *App) handleObtainCert(domain, email string) tea.Cmd {
    // Derives webRoot from config: cfg.SitesRoot + "/" + domain
    return func() tea.Msg { /* ... */ }
}
```

**app_handlers_backup.go**
```go
func (a *App) handleCreateBackup(domain, backupType string) tea.Cmd {
    // Converts string type to backup.BackupType enum
    return func() tea.Msg { /* ... */ }
}
func (a *App) handleRestoreBackup(path, domain string) tea.Cmd { /* ... */ }
```

### Key Architectural Patterns

**Form Priority** (screens)
```
if form.Active() { delegate keys to form }
else if confirm.Active() { delegate keys to confirm }
else if spinner.Active() { forward tick message }
else { normal key handling }
```

**Operation Result Handling** (app.go)
```
Receive OpDoneMsg/OpErrMsg
    ↓
Show toast (success/error message)
    ↓
Call screen.StopSpinner()
    ↓
Fetch updated data (list refresh)
    ↓
Re-render screen with new state
```

**Spinner Control**
- Started by screen when form submitted
- Stopped by App when result message received
- Example: `s.spinner.Start("Installing...")` → handler → `a.phpScreen.StopSpinner()`

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
