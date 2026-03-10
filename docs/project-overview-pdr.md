# juiscript - Project Overview & Product Development Requirements

## Product Vision

juiscript is a production-grade LEMP (Linux, Nginx, PHP-FPM, MariaDB) server management tool for hosting Laravel and WordPress applications on Ubuntu 22.04/24.04. It combines a powerful CLI with an intuitive Bubble Tea TUI to simplify DevOps workflows while enforcing security best practices through per-site user isolation.

## Core Objectives

- **Simplify LEMP Management**: Single binary replacing shell scripts and manual configuration
- **Security First**: Per-site Linux user isolation, atomic file writes, restrictive permissions
- **Developer Experience**: Beautiful TUI, fast single-binary deployment, sensible defaults
- **Production Ready**: Full audit logging, error handling, Ubuntu LTS support

## Functional Requirements

### Site Lifecycle Management
- Create isolated sites with dedicated Linux user accounts
- Support Laravel and WordPress detection/optimization
- Configure Nginx vhosts and PHP-FPM pools automatically
- Delete sites with full cleanup (users, files, databases)

### Web Server (Nginx)
- Generate vhost configs from templates
- Enable/disable sites via symlink management
- Reload Nginx safely
- SSL certificate management (Let's Encrypt via certbot)

### PHP Management
- Support multiple PHP versions (ondrej/php PPA)
- Install/remove PHP versions with common & optional extensions
- Auto-configure PHP-FPM pools per site with dynamic process manager
- Pool per site for isolation and resource control
- Zero-downtime PHP version switching with atomic rollback
- Per-site user/group isolation in FPM pools
- Security constraints: open_basedir, extension restrictions

### Database (MariaDB)
- Create/delete databases and users
- Manage per-site credentials
- Database backups integrated with backup system

### Caching & Queues
- Redis database and configuration
- Supervisor-managed Laravel queue workers
- Queue status monitoring

### Backup & Restore
- Full and partial backups (files, databases)
- Automatic retention policy enforcement
- Restore to point-in-time

### Service Control
- Start/stop/restart Nginx, PHP-FPM, MariaDB, Redis
- Service status monitoring
- Systemd integration

### Firewall Management
- UFW (Uncomplicated Firewall) rule management
- Open/close ports with protocol selection (TCP/UDP)
- Fail2ban IP blocking integration for brute-force protection
- View and manage blocked IPs by jail

## Non-Functional Requirements

### Architecture
- Single statically-compiled binary (no runtime dependencies)
- Config at `/etc/juiscript/config.toml` (TOML format)
- Embedded templates for all config generation
- Interface-based design for testability

### Security
- Root-only CLI execution enforcement
- Atomic file writes (temp + rename pattern)
- Restrictive file permissions (0640 configs, 0750 dirs)
- Command execution audit logging
- Per-site user/group isolation (no www-data for apps)

### Performance
- Context-aware timeouts (30s default)
- Parallel TUI updates
- Minimal startup overhead

### Reliability
- Comprehensive error handling and messaging
- Graceful fallback to defaults when config missing
- Safe symlink operations with validation
- Transaction-like operations (atomic file ops)

## Technical Constraints

- **Go 1.22+**: Required for embed and modern tooling
- **Ubuntu 22.04 / 24.04**: Target OS only
- **Root Access**: Required for system operations
- **Single Binary**: No runtime dependencies
- **Linux systemd**: For service management

## Acceptance Criteria

### MVP Functionality
- [x] CLI entry point with version command
- [x] TUI dashboard screen with navigation
- [x] TOML config loading/saving with defaults
- [x] Template engine with embedded files
- [x] System command execution with logging
- [x] Atomic file operations
- [x] Linux user management interface
- [x] Nginx vhost CRUD & enable/disable
- [x] PHP version install/remove/list with FPM status
- [x] PHP-FPM pool creation, deletion, version switching
- [x] TUI PHP management screen
- [x] Service control: start/stop/restart/reload services
- [x] Service status monitoring with health checks
- [x] TUI service management screen
- [x] Site creation and deletion (integrates Nginx + PHP)
- [x] MariaDB user/database management
- [x] SSL certificate automation
- [x] Backup scheduling and execution
- [x] Supervisor queue worker management
- [x] UFW firewall rule management (open/close ports)
- [x] Fail2ban IP blocking integration
- [x] TUI firewall management screen
- [x] Redis cache management and status monitoring
- [x] PHP Opcache reset functionality
- [x] Cache flush operations (per-database and all)
- [x] TUI cache management screen

### Quality Standards
- Unit test coverage > 70%
- All critical paths tested
- No root-required tests (interfaces for mocking)
- Gofmt/govet compliance
- Clear error messages for users

### Documentation
- README with quick start
- Architecture overview
- Code standards and patterns
- API/component documentation
- Troubleshooting guide

## Success Metrics

- Single 50MB binary deployable on any Ubuntu 22/24 box
- Site creation < 10s
- TUI response time < 200ms
- Configuration changes atomic (no partial updates)
- Audit log captures all system operations

## Version & Changes

- **v0.1.0-dev** (2026-03-02): MVP complete through Phase 07
  - Phases 01-07 implemented: Infrastructure, Nginx, PHP, Database, SSL, Service Control
  - All major LEMP management features operational
  - 80+ unit tests with high coverage
  - TUI screens for all feature areas

- **Phase 04** (Complete): PHP Management
  - PHP version install/remove/list
  - ondrej/php PPA auto-setup
  - FPM service status tracking
  - Per-site FPM pool creation/deletion
  - Zero-downtime version switching
  - Dynamic process manager (pm=dynamic)
  - Per-site user/group isolation
  - Security restrictions (open_basedir, extensions)
  - TUI PHP management screen

- **Phase 05** (Complete): Database Management
  - MariaDB user/database CRUD operations
  - Database import/export with streaming
  - Password generation and reset
  - System database protection
  - TUI database management screen
  - 20+ unit tests with mock executor

- **Phase 06** (Complete): SSL Management
  - Let's Encrypt certificate automation via certbot
  - Nginx SSL injection with atomic rollback
  - Certificate status monitoring and renewal
  - HTTP→HTTPS redirect with ACME challenge support
  - TUI SSL management screen
  - Full unit test coverage

- **Phase 07** (Complete): Service Control
  - Systemd service control (start/stop/restart/reload)
  - Service health monitoring
  - Dynamic PHP-FPM service detection
  - Whitelist-based security (no shell injection)
  - Service status with PID and memory tracking
  - TUI service management screen
  - 16+ unit tests

- **Phase 5-6** (Complete): TUI Input Forms & Feedback
  - **Phase 5**: Form component wiring to all operation screens
    - PHP screen: version picker form for install, confirmation for remove
    - Database screen: create/import forms with dual modes, confirmation for drop
    - SSL screen: domain+email form for certificate obtain, confirmation for revoke
    - Backup screen: domain+type form for create, confirmation for delete/restore
    - SiteDetail: confirmation for delete
    - All message types updated with parameter fields
    - All handlers converted from stubs to real implementations
  - **Phase 6**: Feedback components for UX polish
    - Toast notifications: success/error toasts for all operation results (app-level)
    - Spinners: loading indicators for long operations (PHP install, backup create/restore, SSL obtain)
    - Confirmations: modal dialogs for destructive actions (remove, drop, delete, revoke)
    - Complete operation flow: form → validation → spinner → result toast → refresh

- **Phase 10** (Complete): Firewall Management
  - UFW rule management (open/close ports with TCP/UDP protocol selection)
  - Fail2ban IP blocking integration for brute-force protection
  - Rule deletion by number
  - Jail status with blocked IP lists
  - Dual-tab TUI screen: UFW rules + blocked IPs table view
  - CLI subcommands for firewall operations
  - Port range validation (1-65535), IP format validation

- **Phase 11** (Complete): Cache Management
  - Redis status monitoring (connectivity, version, memory)
  - Redis enable/disable per site with DB validation
  - Cache flush operations (per-database and all databases)
  - PHP Opcache reset via PHP-FPM restart
  - TUI cache management screen with status display
  - CLI subcommands for cache operations
  - 15 unit tests with mock executor

## Dependencies

### Go Modules
- `github.com/charmbracelet/bubbletea`: TUI framework
- `github.com/charmbracelet/bubbles`: TUI components
- `github.com/charmbracelet/lipgloss`: Styling
- `github.com/charmbracelet/huh`: Form components
- `github.com/spf13/cobra`: CLI framework
- `github.com/BurntSushi/toml`: Config parsing

### System Requirements
- Nginx
- PHP-FPM (multiple versions)
- MariaDB
- Redis
- Supervisor
- certbot (SSL)
