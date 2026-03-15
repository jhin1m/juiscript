# HOSTVN Codebase Scout Report

**Date**: 2026-03-10  
**Source**: /Users/jhin1m/Desktop/hostvn  
**Type**: Server Management & LEMP Stack Installation System

## Overview

HOSTVN is a comprehensive bash-based VPS management system for automated LEMP stack (Linux, Nginx, MariaDB, PHP) installation and optimization on Ubuntu. It provides a menu-driven CLI interface for server administration with ~145 feature modules.

**Stack**: Bash shell scripts + Nginx configuration templates  
**Supported OS**: Ubuntu 20.04, 22.04, 24.04  
**Min Requirements**: 512MB RAM, root access

---

## Architecture

### Entry Points

1. **Primary**: `/menu/hostvn` - Main menu interface
   - Interactive shell-based CLI
   - 15 main menu options
   - Language selection (Vietnamese/English)
   - Dispatches to feature routes

2. **Installation**: `/install` - Initial setup script
   - OS detection & validation
   - Prevents conflicting control panels
   - Downloads OS-specific scripts from GitHub

3. **Ubuntu Setup**: `/ubuntu/ubuntu` - Main installation module
   - Removes old services
   - Installs base packages
   - Installs/configures LEMP stack
   - Runs 1000+ lines of installation logic

### Directory Structure

```
hostvn/
├── menu/                          # Main application
│   ├── hostvn                      # CLI entry point
│   ├── route/                      # Menu navigation (27 route files)
│   ├── controller/                 # Feature implementation (122 files, 19 categories)
│   ├── helpers/                    # Shared utilities & variables
│   ├── template/                   # Nginx config templates (31 files)
│   ├── validate/                   # Input validation rules
│   ├── lang/                       # Localization (vi, en)
│   └── cronjob/                    # Scheduled tasks (7 cron jobs)
├── ubuntu/                         # Installation scripts
│   ├── ubuntu                      # Main installer
│   ├── modules/                    # Installation modules
│   └── update/                     # Update modules
├── install                         # Bootstrap script
└── [config files]                  # phpMyAdmin, images, etc.
```

---

## Feature Modules (122 Controllers)

### 1. Site/Domain Management
**Path**: `/menu/controller/domain/` (12 files)  
**Route**: `/menu/route/domain`

**Capabilities**:
- `add_domain` - Create new website with SFTP user
- `delete_domain` - Remove website & SFTP user
- `list_domain` - List all sites
- `change_domain` - Modify domain settings
- `alias_domain` - Add parked/alias domains
- `redirect_domain` - Domain redirect setup
- `clone_website` - Duplicate existing site
- `change_php_version` - Switch PHP version per domain
- `change_database_info` - Update DB credentials
- `change_pass_sftp` - Reset SFTP password
- `protect_dir` - HTTP Basic Auth for directories
- `rewrite_config` - Template-based nginx config (WordPress, Laravel, Magento2, etc.)

### 2. Nginx Configuration
**Path**: `/menu/controller/[domain]/` + `/menu/template/` (31 templates)  
**Route**: `/menu/route/lemp_nginx`

**Features**:
- Restart/test nginx configuration
- Update nginx from source
- Rebuild nginx with custom modules
- Templates for: WordPress, Laravel, Magento2, CakePHP, CS-Cart, Moodle, NextCloud, Sendy, Mautic
- NodeJS proxy configuration
- Nginx pagespeed module support

### 3. PHP Management
**Path**: `/menu/controller/php/` (9 files)  
**Route**: `/menu/route/lemp_php`

**Capabilities**:
- Dual PHP version support (5.6-8.4)
- Install additional PHP versions
- Change PHP version per domain
- PHP configuration: `php.ini` settings
- `allow_url_fopen` - Enable/disable remote file loading
- `open_basedir` - Security restriction
- `proc_close` - Control process functions
- `php_setting` - Optimize PHP memory, timeouts, upload limits
- `process_manager` - Manage php-fpm worker processes
- ionCube loader installation
- Extensions: Redis, Memcached

### 4. SSL/TLS Management
**Path**: `/menu/controller/ssl/` (5 files)  
**Route**: `/menu/route/ssl_letencrypt`

**Features**:
- Let's Encrypt SSL (automated, free)
- Zero SSL support
- Paid SSL installation
- CloudFlare DNS API validation (for DNS-01 challenge)
- Alias domain SSL
- Auto-renewal configuration
- Certificate management

### 5. Database Management
**Path**: `/menu/controller/database/` (5 files)  
**Route**: `/menu/route/lemp_database`

**Operations**:
- `create_db` - Create MariaDB database & user
- `delete_db` - Remove database
- `change_password` - Reset database user password
- `import_db` - Import SQL dumps
- `remote_mysql` - Configure remote database access

**MariaDB Version**: 11.8 (latest)

### 6. Cache Management
**Path**: `/menu/controller/cache/` (10 files)  
**Route**: `/menu/route/cache`

**Components**:
- **Redis**: Installation, management, admin panel (phpRedisAdmin)
- **Memcached**: Installation, management, admin panel (PHPMemcachedAdmin)
- **Opcache**: Enable/disable, blacklist management, dashboard
- **Nginx Fastcgi Cache**: Enable/disable, cache configuration
- **Clear Cache**: Flush Redis/Memcached/Opcache
- Cache key management for WordPress multi-site

### 7. Backup & Restore
**Path**: `/menu/controller/backup/` (7 files)  
**Route**: `/menu/route/backup`

**Features**:
- Multiple backup locations:
  - Local (VPS storage)
  - Google Drive (Rclone integration)
  - OneDrive (Rclone integration)
- Backup types:
  - Full (source code + database)
  - Code only
  - Database only
- Selective domain backup
- Automatic restore from backups
- Backup scheduling (cronjob setup)
- Multi-account Google Drive support

### 8. WordPress Management
**Path**: `/menu/controller/wordpress/` (28 files)  
**Route**: `/menu/route/wordpress` + `/menu/route/wordpress_advanced`

**Capabilities**:
- `auto_install` - Install WordPress with WP-CLI
- `update_wordpress` - Update core
- `update_plugins` - Bulk plugin updates
- `optimize_database` - Clean up DB, optimize tables
- `post_revision` - Limit post revisions
- Database repair
- Password reset (wp-admin)
- Domain migration
- Disable/enable edit theme & plugins
- Debug mode toggle
- Maintenance mode
- XML-RPC disable (security)
- User API disable (`/wp-json/wp/v2/users`)
- Plugin cache integration: WP-Rocket, W3 Total Cache, WP Supercache, Cache Enabler, Swift Performance, Fast Cache
- SEO config: Yoast SEO, Rank Math
- WebP Express support
- htpasswd protection for wp-admin
- WP-CLI installation

### 9. Firewall Management
**Path**: `/menu/controller/firewall/` (implied in routes)  
**Route**: `/menu/route/firewall`

**Integrated Tools**:
- Fail2ban configuration
- IP blocking/unblocking
- SSH brute-force protection
- SFTP protection
- Admin tool protection
- Port management

### 10. Logging
**Path**: `/menu/controller/log/domain_log` + nginx config  
**Route**: `/menu/route/lemp_log`

**Monitoring**:
- View Nginx access/error logs
- MySQL error logs
- PHP error logs
- Per-domain log filtering
- Real-time log monitoring

### 11. Permission Management
**Path**: `/menu/controller/permission/` (2 files)  
**Route**: `/menu/route/permission`

**Operations**:
- Fix permissions (chown/chmod) for all domains
- Fix permissions for single domain
- User isolation (different SFTP user per site)

### 12. Pagespeed Optimization
**Path**: `/menu/controller/pagespeed/` (8 files)  
**Route**: `/menu/route/lemp_ngx_pagespeed`

**Features**:
- CSS combining
- JS combining
- Image compression
- WebP image conversion
- CSS minification
- JS minification
- HTML minification
- Enable/disable pagespeed module

### 13. VPS Management
**Path**: `/menu/controller/vps/` (11 files)  
**Route**: `/menu/route/vps_manage`

**Capabilities**:
- VPS info (IP, RAM, CPU, disk usage)
- Webserver info (Nginx, MariaDB, PHP versions)
- Change VPS IP address
- SSH port change
- Create swap memory
- VPS parameter optimization
- Kernel updates
- Install ClamAV antivirus
- Install ImunifyAV malware scanner
- Move VPS (site migration)
- VPSSIM simulator
- Language selection

### 14. Scheduled Tasks (Cronjobs)
**Path**: `/menu/controller/cronjob/` + `/menu/cronjob/` (11 files)  
**Route**: `/menu/route/cronjob`

**Automated Tasks**:
- `backup_local` - Local backup scheduling
- `backup_google` - Google Drive backup
- `backup_onedrive` - OneDrive backup
- `gg_drive_all` / `gg_drive_one` - Drive-specific
- `wpcron` - WordPress cron setup
- `updateCloudflareRangeIP` - Auto-sync CloudFlare IPs
- Disk usage alerts (Telegram)
- Service status alerts
- SSH login alerts

### 15. Notifications
**Path**: `/menu/controller/telegram/` (6 files)  
**Route**: `/menu/route/notify`

**Channels**:
- Telegram bot notifications
- Disk usage alerts
- Service health checks
- SSH login notifications
- High load alerts
- Multiple notification profiles

### 16. Admin Tools
**Path**: `/menu/controller/admin/` (5 files)  
**Route**: `/menu/route/admin_tool`

**Tools**:
- phpMyAdmin (accessible on custom port)
- Opcache Dashboard (monitoring)
- phpMemcachedAdmin (cache inspection)
- phpRedisAdmin (cache inspection)
- Change admin port (security)
- WP-CLI setup

### 17. Account Management
**Path**: `/menu/controller/account/` (3 files)  
**Route**: `/menu/route/account_manage`

**Features**:
- Website info listing
- SFTP account management
- Admin tool access

### 18. Utility Tools
**Path**: `/menu/controller/tools/` (9 files)  
**Route**: `/menu/route/menu_tools`

**Tools**:
- `auto_install_source` - Auto-install frameworks (WordPress, Laravel, etc.)
- `compress_image` - Image optimization
- `decompress_file` - Extract archives
- `deploy_website` - Git deployment
- `download_file_gg_drive` - Download from Google Drive
- `find_large_file_folder` - Disk space analysis
- `install_av` - Install antivirus
- `install_nodejs` - Node.js setup
- `website_disk_usage` - Per-site storage analysis

### 19. Security Features (Built-in)
- **SSH Security**: Custom port, key-based auth
- **Firewall**: Fail2ban integration
- **PHP Hardening**: disable dangerous functions, open_basedir
- **Application isolation**: Separate SFTP users per site
- **DDoS protection**: Nginx rate limiting
- **SSL/TLS**: Let's Encrypt + CloudFlare DNS
- **Malware scanning**: ClamAV, ImunifyAV integration
- **WP Security**: Brute-force protection, xmlrpc disable, user API hide

---

## Key Files Summary

### Helper Functions
- `/menu/helpers/function` - Common utilities (_gen_pass, _select_domain, _bytes_for_humans, etc.)
- `/menu/helpers/variable_common` - Shared variables (colors, messages, paths)
- `/menu/helpers/variable_php` - PHP-specific variables

### Validation
- `/menu/validate/rule` - Input validation rules
- `/menu/validate/check_value` - Value validation

### Nginx Templates (31 files)
Base templates for applications:
- wordpress.conf
- default.conf (basic PHP)
- laravel.conf
- magento2.conf
- cakephp.conf
- nodejs.conf
- nextcloud.conf
- php_cache.conf
- pagespeed modules (8 sub-configs)

### Languages
- `/menu/lang/vi/` - Vietnamese localization
- `/menu/lang/en/` - English localization

---

## Feature Comparison with JuiScript

**HOSTVN Strengths**:
- Production-ready (2000+ deployments)
- Complex multi-server scenario handling
- Deep WordPress integration
- Extensive backup solutions (3 cloud providers)
- Built-in antivirus/malware scanning
- Telegram notifications
- User isolation per domain (security)
- Dual PHP version support
- Extensive template library

**JuiScript Opportunities**:
- More modular/structured CLI (vs flat menu system)
- Cleaner codebase architecture
- Better error handling patterns
- Modern TUI components
- Extensibility patterns
- Plugin system potential
- Better code organization (vs 150+ controller files)

---

## Configuration & Runtime

### Installation Process
1. Download bootstrap script → verify OS → download ubuntu installer
2. Remove conflicting control panels
3. Install dependencies (curl, git, build-tools)
4. Install LEMP (Nginx, MariaDB, PHP-FPM)
5. Configure Nginx vhosts
6. Install tools (WP-CLI, Composer, Rclone, Supervisor)
7. Setup security (Fail2ban, firewall rules)
8. Configure optimization (Nginx cache, PHP opcache, caching layers)

### Configuration Storage
- `/var/hostvn/.hostvn.conf` - Main config
- `/var/hostvn/ipaddress` - IP detection
- `/etc/nginx/` - Nginx configs (domains, templates, rewrites)
- `/home/[user]/[domain]/` - Per-site directories

### Script Integration Points
- Cronjobs in `/var/spool/cron/`
- Systemd services for Nginx, MariaDB, PHP-FPM
- Fail2ban jail configurations

---

## Unresolved Questions

1. Complete list of all 122 controller scripts with function counts
2. Total lines of code across all modules
3. Dependency tree between controllers
4. Security audit status
5. Performance benchmarks for large deployments
6. Upgrade/downgrade procedures
7. Disaster recovery procedures beyond backups

