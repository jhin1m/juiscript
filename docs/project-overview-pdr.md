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
- Auto-configure PHP-FPM pools per site
- Pool per site for isolation and resource control

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
- [ ] Site creation and deletion
- [ ] Nginx vhost generation
- [ ] PHP-FPM pool configuration
- [ ] MariaDB user/database management
- [ ] SSL certificate automation
- [ ] Backup scheduling and execution
- [ ] Supervisor queue worker management

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

- **v0.1.0-dev** (2026-03-02): Initial project structure, MVP scaffolding
- Framework: Go CLI with Bubble Tea TUI, config system, template engine
- Status: Core infrastructure complete, feature implementation in progress

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
