# HOSTVN File Inventory & Key Paths

## Main Entry Points & Core Files

### Bootstrap & Installation
- `/Users/jhin1m/Desktop/hostvn/install` - Main installation bootstrap script
- `/Users/jhin1m/Desktop/hostvn/ubuntu/ubuntu` - Ubuntu installer (1000+ lines)

### CLI Interface
- `/Users/jhin1m/Desktop/hostvn/menu/hostvn` - Primary CLI menu entry point (75 lines)

---

## Menu System Structure

### Route Files (27 files - Navigation logic)
Path: `/Users/jhin1m/Desktop/hostvn/menu/route/`

**Primary Routes:**
- `parent` (559 lines) - Master route dispatcher
- `domain` (96 lines) - Domain management menu
- `ssl` - SSL/TLS management menu
- `cache` (61 lines) - Cache management menu
- `lemp` (61 lines) - LEMP stack menu
- `lemp_nginx` (260 lines) - Nginx-specific management
- `lemp_php` (103 lines) - PHP management
- `lemp_database` (75 lines) - Database management
- `lemp_log` (137 lines) - Log viewing
- `lemp_ngx_pagespeed` (76 lines) - PageSpeed module
- `ssl_letencrypt` (62 lines) - Let's Encrypt configuration
- `backup` (71 lines) - Backup menu
- `firewall` (130 lines) - Firewall management
- `cronjob` (56 lines) - Scheduled tasks
- `opcache` (60 lines) - OpCache management
- `account_manage` (61 lines) - Account management
- `admin_tool` (58 lines) - Admin tools
- `permission` - Permission management
- `vps_manage` (109 lines) - VPS management
- `vps_ssh` (28 lines) - SSH management
- `wordpress` (50 lines) - WordPress menu
- `wordpress_advanced` (54 lines) - Advanced WordPress
- `wordpress_plugins_manage` - Plugin management
- `menu_tools` (50 lines) - Utility tools
- `notify` - Notifications
- `log_level` - Log level management

---

## Controller Files (122 files total)

### Domain Management (12 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/domain/`

- `add_domain` - Create new website
- `delete_domain` - Remove website
- `list_domain` - List all domains
- `change_domain` - Modify domain
- `alias_domain` - Parked domains
- `redirect_domain` - Domain redirects
- `clone_website` - Duplicate site
- `change_php_version` - PHP version per site
- `change_database_info` - Update DB credentials
- `change_pass_sftp` - Reset SFTP password
- `protect_dir` - HTTP basic auth
- `rewrite_config` - Nginx config templates

### SSL Management (5 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/ssl/`

- `create_le_ssl` - Let's Encrypt setup
- `le_alias_domain` - SSL for alias domains
- `remove_le` - Remove SSL certificate
- `cf_api` - CloudFlare DNS API
- `paid_ssl` - Paid certificate installation

### PHP Management (9 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/php/`

- `change_php1` - Switch PHP version
- `change_php2` - Alternative PHP switching
- `install_php2` - Install additional PHP
- `php_setting` - PHP configuration
- `allow_url_fopen` - Enable/disable remote files
- `open_basedir` - Security restrictions
- `proc_close` - Process control
- `process_manager` - php-fpm workers
- `install_ioncube` - ionCube loader

### Database Management (5 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/database/`

- `create_db` - Create database
- `delete_db` - Remove database
- `change_password` - Reset password
- `import_db` - Import SQL
- `remote_mysql` - Remote access

### Cache Management (10 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/cache/`

- `install_memcached` - Install Memcached
- `install_redis` - Install Redis
- `manage_memcached` - Memcached management
- `manage_redis` - Redis management
- `nginx_cache` - Nginx cache config
- `opcache/enable_disable` - OpCache toggle
- `opcache/add_blacklist` - BlackList files
- `opcache/remove_blacklist` - Remove blacklist
- `clear_cache` - Flush all caches
- `script/install_php_redis.sh` - PHP Redis extension
- `script/install_php_memcached.sh` - PHP Memcached extension

### Backup & Restore (7 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/backup/`

- `backup` - Backup creation
- `restore` - Restore from backup
- `delete` - Delete backups
- `connect_google_drive` - Google Drive setup
- `connect_ondrive` - OneDrive setup
- `delete_connect_google_drive` - Remove Drive connection
- `delete_connect_onedrive` - Remove OneDrive connection

### WordPress Management (28 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/wordpress/`

- `auto_install` - Automatic WP install
- `auto/` - Sub-directory for auto-install
- `update_wordpress` - Update core
- `update_plugins` - Update plugins
- `optimize_database` - DB optimization
- `debug_mode` - Enable/disable debug
- `maintenance_mode` - Maintenance page
- `disable_xmlrpc` - Disable XML-RPC
- `disable_user_api` - Hide user API
- `disable_edit_theme_plugins` - Lock theme/plugins
- `change_pass_wp_admin` - Reset password
- `htpasswd_wp_admin` - HTTP auth for admin
- `lock_down` - Security hardening
- `post_revision` - Limit revisions
- `cache_plugins` - Configure cache plugins
- `cache_key` - Multi-site cache keys
- `yoast_seo` - Yoast SEO config
- `rank_math_seo` - Rank Math config
- `webp_express` - WebP support
- `cron_job` - WP cron setup
- `deactivate_all_plugins` - Disable all plugins
- `move_wp_config` - Move wp-config.php
- `change_domain` - Domain migration (implied)

### VPS Management (11 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/vps/`

- `vps_info` - VPS information
- `webserver_info` - Server details
- `change_ip_vps` - Change IP address
- `change_port_ssh` - SSH port change
- `change_vps_parameter` - Optimize VPS
- `create_swap` - Create swap memory
- `update_scripts` - Update HOSTVN scripts
- `install_clamav` - ClamAV antivirus
- `install_av` - ImunifyAV (implied)
- `move_vps` - VPS migration
- `vpssim` - VPS simulator
- `change_language` - Language selection

### Cron Jobs (7 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/cronjob/`

- `backup_local` - Local backup cron
- `backup_onedrive_all` - OneDrive backup all
- `backup_onedrive_one` - OneDrive backup single
- `gg_drive_all` - Google Drive backup all
- `gg_drive_one` - Google Drive backup single
- `wpcron` - WordPress cron
- `updateCloudflareRangeIP` - CF IP sync

### Cronjob Controller (7 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/cronjob/`

- `backup_local` - Setup local backup
- `backup_google` - Setup Google Drive
- `backup_onedrive` - Setup OneDrive
- `destroy_cron` - Remove cron jobs
- `connect_google_drive` - Drive connection
- `connect_ondrive` - OneDrive connection
- `delete_connect_google_drive` - Disconnect Drive

### Firewall Management (implied)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/firewall/`

- Firewall configuration (referenced in routes)
- IP blocking/unblocking
- Fail2ban configuration

### Admin Tools (5 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/admin/`

- `php_memcached_admin` - Memcached admin
- `redis_panel` - Redis admin
- `opcache_panel` - OpCache dashboard
- `change_port` - Admin port change
- `update_phpmyadmin` - phpMyAdmin update

### Logging (1 file)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/log/`

- `domain_log` - Domain log viewer

### Permission Management (2 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/permission/`

- `all` - Fix all permissions
- `one` - Fix single domain

### Tools & Utilities (9 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/tools/`

- `auto_install_source` - Framework auto-install
- `compress_image` - Image optimization
- `decompress_file` - Extract archives
- `deploy_website` - Git deployment
- `download_file_gg_drive` - Download from Drive
- `find_large_file_folder` - Disk analysis
- `install_av` - Antivirus install
- `install_nodejs` - Node.js setup
- `website_disk_usage` - Storage analysis

### PageSpeed Optimization (8 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/pagespeed/`

- `on_off` - Enable/disable
- `minify_html` - HTML minification
- `minify_css` - CSS minification
- `minify_js` - JS minification
- `combine_css` - CSS combining
- `combine_js` - JS combining
- `compress_image` - Image compression
- `convert_img_to_webp` - WebP conversion

### Account Management (3 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/account/`

- `website_info` - Website details
- `phpmyadmin` - phpMyAdmin access
- `admin_tool` - Admin tool access

### Telegram Notifications (6 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/controller/telegram/`

- `connect_telegram` - Setup Telegram
- `disk_notify` - Disk alerts
- `service_notify` - Service status
- `ssh_notify` - SSH login alerts
- `highload` - High load alerts
- `delete_notify` - Remove notifications

---

## Helper & Support Files

### Helpers (4 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/helpers/`

- `function` - Common utility functions
- `variable_common` - Shared variables
- `variable_php` - PHP-specific variables
- `mysql_variable` - MySQL variables

### Validation (2 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/validate/`

- `rule` - Input validation rules
- `check_value` - Value validation

---

## Nginx Configuration Templates (31 files)

Path: `/Users/jhin1m/Desktop/hostvn/menu/template/`

### Base Application Templates
- `default.conf` - Basic PHP
- `wordpress.conf` - WordPress
- `laravel.conf` - Laravel
- `magento2.conf` - Magento 2 main
- `magento2_alias.conf` - Magento 2 alias
- `cakephp.conf` - CakePHP
- `cscart.conf` - CS-Cart main
- `cscart_alias.conf` - CS-Cart alias
- `moodle.conf` - Moodle
- `nextcloud.conf` - NextCloud
- `sendy.conf` - Sendy
- `mautic.conf` - Mautic

### PHP Cache Variants
- `php_cache.conf` - Default cache
- `php_no_cache.conf` - No cache
- `php_dynamic.conf` - Dynamic PHP
- `php_ondemand.conf` - On-demand mode
- `php_cache_woo.conf` - WooCommerce cache
- `nginx_alias.conf` - Alias domains

### NodeJS
- `nodejs.conf` - NodeJS proxy
- `nodejs_alias.conf` - NodeJS alias

### PageSpeed Modules (9 files)
Path: `/Users/jhin1m/Desktop/hostvn/menu/template/ngx_pagespeed/`

- `main.conf` - Main config
- `minify_html.conf` - HTML minify
- `minify_css.conf` - CSS minify
- `minify_js.conf` - JS minify
- `combine_css.conf` - CSS combine
- `combine_js.conf` - JS combine
- `compress_img.conf` - Image compress
- `img_to_webp.conf` - WebP conversion
- `google_fonts.conf` - Google Fonts

### Fail2ban
Path: `/Users/jhin1m/Desktop/hostvn/menu/template/fail2ban/`

- `jail.local` - Fail2ban configuration

### Error Pages
Path: `/Users/jhin1m/Desktop/hostvn/menu/template/error_page/`

- `hvn_404.html` - Custom 404 page

---

## Localization (2+ languages)

Path: `/Users/jhin1m/Desktop/hostvn/menu/lang/`

- `vi/` - Vietnamese translations
- `en/` - English translations

---

## Installation Modules

Path: `/Users/jhin1m/Desktop/hostvn/ubuntu/`

- `modules/` - Installation sub-modules
- `update/` - Update scripts

---

## Configuration & Build Files

Path: `/Users/jhin1m/Desktop/hostvn/`

- `.editorconfig` - Editor configuration
- `.gitignore` - Git ignore rules
- `changelog.txt` - Version history
- `README.md` - Documentation
- `docker` - Docker support file (binary)
- `phpmyadmin.sql` - phpMyAdmin database
- `phpMyAdmin-5.2.2-english.tar.gz` - phpMyAdmin archive
- `optipng-0.7.8.tar.gz` - Image optimizer

---

## Summary Statistics

| Component | Count | Type |
|-----------|-------|------|
| Routes | 27 | Navigation |
| Controllers | 122 | Feature logic |
| Templates | 31 | Nginx configs |
| Helper files | 4 | Utilities |
| Validation | 2 | Rules |
| Languages | 2+ | Localization |
| **Total** | **189+** | **Core files** |

---

## Key Paths Quick Reference

```
CLI Entry: /menu/hostvn
Main Routes: /menu/route/parent, /menu/route/domain, /menu/route/lemp_nginx
Domain Control: /menu/controller/domain/
SSL/TLS: /menu/controller/ssl/ + /menu/route/ssl_letencrypt
PHP: /menu/controller/php/ + /menu/route/lemp_php
Database: /menu/controller/database/ + /menu/route/lemp_database
Cache: /menu/controller/cache/ + /menu/route/cache
WordPress: /menu/controller/wordpress/ + /menu/route/wordpress
Backup: /menu/controller/backup/ + /menu/route/backup
Nginx Templates: /menu/template/*.conf
PageSpeed: /menu/template/ngx_pagespeed/
Firewall: /menu/route/firewall + Fail2ban
VPS Mgmt: /menu/controller/vps/ + /menu/route/vps_manage
Tools: /menu/controller/tools/ + /menu/route/menu_tools
```

