# HOSTVN Scout Summary - Quick Reference

## What is HOSTVN?

Production-grade VPS management system for automated LEMP stack (Linux, Nginx, MariaDB, PHP-FPM) installation & optimization on Ubuntu. Interactive bash CLI with 145+ feature modules organized across ~189 files.

**Maturity**: Production (2000+ real deployments)  
**Type**: Server management / VPS control panel alternative  
**Language**: Bash shell scripts  
**Scope**: Complete VPS lifecycle management

---

## Quick Stats

- **Entry Point**: `/menu/hostvn` (75-line interactive menu)
- **Core Modules**: 122 controller files + 27 route files
- **Configuration Templates**: 31 Nginx configs (WordPress, Laravel, Magento2, etc.)
- **Supported OS**: Ubuntu 20.04, 22.04, 24.04
- **Minimum Requirements**: 512MB RAM, root access
- **Installation Time**: ~15-30 minutes for initial setup

---

## Core Feature Categories (19 Total)

### 1. Site Management (Domain/Website)
- Create/delete/modify domains
- SFTP user per domain
- Domain aliasing & redirects
- Clone entire websites
- Per-domain PHP version switching

### 2. Web Server (Nginx)
- Install/update/rebuild from source
- Test & restart safely
- Custom modules (PageSpeed, cache purge, etc.)
- 31 pre-configured templates
- Per-app configuration (WordPress, Laravel, Magento2, etc.)

### 3. PHP Management
- Support PHP 5.6 through 8.4
- Dual PHP version support
- Per-domain version assignment
- PHP-FPM tuning & optimization
- ionCube, Redis, Memcached extensions

### 4. SSL/TLS
- Let's Encrypt (free, automated)
- Zero SSL support
- Paid SSL installation
- CloudFlare DNS API validation
- Auto-renewal configuration

### 5. Database (MariaDB 11.8)
- Create/delete databases & users
- Password management
- SQL import
- Remote access configuration
- Optimization

### 6. Caching Layer
- **Redis**: Installation, admin panel (phpRedisAdmin)
- **Memcached**: Installation, admin panel (PHPMemcachedAdmin)
- **Opcache**: Enable/disable, blacklisting, dashboard
- **Nginx FastCGI**: Nginx-level caching
- Cache flushing & management

### 7. Backup & Restore
- Local backup (VPS storage)
- Google Drive backup (Rclone)
- OneDrive backup (Rclone)
- Selective domain backups
- Backup scheduling (cron)
- Full/code-only/DB-only options

### 8. WordPress Management (28 files)
- Auto-install with WP-CLI
- Core & plugin updates
- Database optimization
- Security hardening
- Cache plugin integration (WP-Rocket, W3TC, WP Supercache, etc.)
- SEO plugin config (Yoast, Rank Math)
- Anti-brute-force protection
- User API hiding
- XML-RPC disable

### 9. Firewall & Security
- Fail2ban integration
- IP blocking/unblocking
- SSH brute-force protection
- Admin tool DDoS protection
- Custom SSH port
- PHP function restrictions
- open_basedir security
- User isolation per domain

### 10. Logging & Monitoring
- Nginx access/error logs per domain
- MySQL error logs
- PHP error logs
- Real-time filtering

### 11. File Permissions
- Fix all permissions (chown/chmod)
- Per-domain permission repair
- User isolation enforcement

### 12. Page Speed Optimization
- CSS/JS combining
- CSS/JS/HTML minification
- Image compression
- WebP conversion
- Brotli compression
- Nginx PageSpeed module

### 13. VPS Management
- System info (IP, RAM, CPU, disk)
- Webserver version info
- Swap memory creation
- Parameter optimization
- Antivirus (ClamAV, ImunifyAV)
- SSH port customization
- IP address changes

### 14. Scheduled Tasks
- Local backup cronjobs
- Google Drive backup cronjobs
- OneDrive backup cronjobs
- WordPress cron jobs
- CloudFlare IP auto-sync
- Alert notifications

### 15. Notifications
- Telegram bot integration
- Disk usage alerts
- Service health checks
- SSH login notifications
- High load alerts

### 16. Admin Tools
- phpMyAdmin (custom port)
- Opcache dashboard
- Cache admin panels
- WP-CLI integration

### 17. Account Management
- Website info listing
- SFTP account control
- Admin access management

### 18. Development Tools
- Framework auto-install (WordPress, Laravel, etc.)
- Image compression/optimization
- Git deployment
- Google Drive file download
- Node.js installation
- Disk usage analysis

### 19. System Tools
- Language selection
- Script updates
- VPS migration
- Decompress archives

---

## Architecture Patterns

### Navigation Model
```
hostvn (main menu)
  ├── Domain Management → domain route → 12 controllers
  ├── SSL Management → ssl route → 5 controllers
  ├── Cache → cache route → 10 controllers
  ├── LEMP Stack
  │   ├── Nginx → lemp_nginx → 260-line controller
  │   ├── PHP → lemp_php → 103-line controller
  │   └── Database → lemp_database → 75-line controller
  ├── WordPress → wordpress + wordpress_advanced → 28 controllers
  ├── Backup → backup route → 7 controllers
  ├── Firewall → firewall route → 130-line controller
  └── [12 more main menus]
```

### Code Organization
- **Routes**: Navigate between menus (27 files)
- **Controllers**: Execute actual operations (122 files)
- **Helpers**: Shared functions & variables (4 files)
- **Templates**: Nginx configuration stubs (31 files)
- **Validation**: Input checking (2 files)
- **Localization**: Multi-language support (vi, en)

---

## Technology Stack

### Installed Components
- **OS**: Ubuntu 20.04/22.04/24.04
- **Web**: Nginx 1.26.3 (stable)
- **Database**: MariaDB 11.8
- **PHP**: 5.6-8.4 (selectable)
- **Cache**: Redis, Memcached, Opcache
- **Certs**: Let's Encrypt, CloudFlare DNS API
- **Firewall**: Fail2ban
- **Admin**: phpMyAdmin, custom dashboards
- **Deployment**: WP-CLI, Composer, Supervisor, Rclone
- **Security**: ClamAV, ImunifyAV
- **Backup**: Rclone (Google Drive, OneDrive)

---

## Key Files to Reference

**Essential**:
- `/menu/hostvn` - Entry point
- `/menu/route/parent` - All function definitions
- `/menu/helpers/function` - Utility functions
- `/ubuntu/ubuntu` - Installation logic

**Domain Operations**:
- `/menu/controller/domain/add_domain` - Create site
- `/menu/route/domain` - Domain menu

**Nginx**:
- `/menu/route/lemp_nginx` - Nginx management
- `/menu/template/wordpress.conf` - WordPress template example

**WordPress**:
- `/menu/controller/wordpress/` - 28 WordPress-specific functions

**Backup**:
- `/menu/controller/backup/` - Backup implementation
- `/menu/controller/cronjob/` - Scheduled backups

---

## Comparison to JuiScript

**What HOSTVN Does Better**:
- Proven production deployment record
- Extensive WordPress ecosystem support
- Multiple cloud backup providers
- Deep server optimization
- Dual PHP version support
- Malware scanning integration

**What JuiScript Should Improve**:
- Modular architecture (vs 122 flat controllers)
- Better error handling/recovery
- Modern TUI components vs interactive prompts
- Plugin/extension system
- Structured CLI subcommands (vs menu-driven)
- API/programmatic access
- Configuration file support
- Better code reusability

---

## Potential JuiScript Learnings

1. **Domain Isolation**: Separate SFTP user + PHP-FPM pool per domain = security
2. **Template System**: Pre-configured Nginx templates for common platforms saves setup time
3. **Multi-backup Strategy**: Support multiple cloud providers (GDrive, OneDrive, local)
4. **WordPress Focus**: Deep WP integration is valuable for shared hosting use case
5. **Caching Layers**: Support multiple cache backends (Redis, Memcached, Opcache, Nginx FastCGI)
6. **CloudFlare Integration**: DNS API validation speeds up SSL issuance
7. **Notification System**: Telegram alerts for critical events
8. **Auto-optimization**: Detect VPS specs and auto-tune configuration

---

## Detailed Reports Generated

1. **scout-260310-1452-hostvn-features.md** (13K)
   - 19 feature categories explained
   - Architecture overview
   - Integration points

2. **scout-260310-1452-hostvn-files.md** (12K)
   - Complete file inventory (189+ files)
   - Directory structure
   - Key paths reference

3. **scout-260310-1452-hostvn-summary.md** (this file)
   - Quick reference guide
   - Tech stack summary
   - Learning opportunities

---

## Contact & References

- **Homepage**: https://hostvn.vn, https://hostvn.net
- **Documentation**: https://help.hostvn.vn/
- **Community**: https://www.facebook.com/groups/hostvn.vn
- **GitHub**: https://github.com/dtt247/hostvn
- **Author**: Sanvv (HOSTVN Technical)

