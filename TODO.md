# TODO - juiscript

## Phase 1: TUI-Backend Wiring -- DONE (v1.3.0)

All 22 TODO handlers wired. 9 backend managers injected via AppDeps pattern.

### Site Operations
- [x] Load site detail from manager
- [x] Call site manager to create site
- [x] Call site manager to toggle site
- [x] Call site manager to delete site

### Nginx Operations
- [x] Call nginx manager to toggle vhost
- [x] Call nginx manager to delete vhost
- [x] Call nginx manager to test config

### PHP Operations
- [x] Call PHP manager to install version
- [x] Call PHP manager to remove version (with active sites safety check)

### Database Operations
- [x] Call database manager to create DB
- [x] Call database manager to drop DB
- [x] Call database manager to import DB
- [x] Call database manager to export DB

### SSL Operations
- [x] Call SSL manager to obtain cert
- [x] Call SSL manager to revoke cert
- [x] Call SSL manager to renew cert

### Service Operations
- [x] Call service manager to start/stop/restart/reload (consolidated via handleServiceAction)

### Queue/Supervisor Operations
- [x] Call supervisor manager to start/stop/restart worker (consolidated via handleWorkerAction)
- [x] Call supervisor manager to delete worker

### Backup Operations
- [x] Call backup manager to create backup
- [x] Call backup manager to restore backup
- [x] Call backup manager to delete backup

## Phase 2: TUI Input Forms & Feedback -- DONE (v1.4.0)

Reusable FormModel, ConfirmModel, ToastModel, SpinnerModel components. All 6 placeholder handlers now accept real user input.

### Form Screens
- [x] PHP version picker for InstallPHPMsg
- [x] Database name input for CreateDBMsg
- [x] File path input for ImportDBMsg
- [x] Domain + email input for ObtainCertMsg
- [x] Domain + type selector for CreateBackupMsg
- [x] Confirmation dialog for destructive actions (delete site, drop DB, revoke cert, delete backup, restore backup)

### Progress & Feedback
- [x] Spinner cho long-running operations (backup, PHP install, SSL obtain)
- [x] Error toast/notification khi operation fail
- [x] Success toast sau mỗi action

## Phase 3: Feature Parity with HostVN (Priority: HIGH)

Tính năng thiết yếu cho hosting management, chưa có trong juiscript.

### Firewall Management (UFW + Fail2ban)
- [x] `firewall-manager.go` - wrap UFW + Fail2ban commands
- [x] CLI: `juiscript firewall status` - show firewall status + rules
- [x] CLI: `juiscript firewall open-port --port <port>` - allow port
- [x] CLI: `juiscript firewall close-port --port <port>` - deny port
- [x] CLI: `juiscript firewall block-ip --ip <ip>` - block IP via Fail2ban
- [x] CLI: `juiscript firewall unblock-ip --ip <ip>` - unblock IP
- [x] CLI: `juiscript firewall list-blocked` - list blocked IPs
- [x] TUI: Firewall screen with port/IP management

### Cache Management (Redis/Opcache) -- DONE
- [x] `cache-manager.go` - manage cache services per site
- [x] CLI: `juiscript cache enable-redis --domain <domain>` - enable Redis for site
- [x] CLI: `juiscript cache disable-redis --domain <domain>` - disable Redis
- [x] CLI: `juiscript cache flush --domain <domain>` - flush all caches
- [x] CLI: `juiscript cache opcache-reset` - reset PHP Opcache
- [x] TUI: Cache screen with enable/disable/flush actions

### Nginx Config Tuning
- [ ] Extend `nginx-manager.go` - FastCGI cache, Brotli, security headers
- [ ] CLI: `juiscript nginx enable-cache --domain <domain>` - FastCGI cache
- [ ] CLI: `juiscript nginx disable-cache --domain <domain>`
- [ ] CLI: `juiscript nginx enable-brotli` - Brotli compression
- [ ] CLI: `juiscript nginx security-headers --domain <domain>` - add security headers
- [ ] Template-based nginx snippets for cache/compression configs

### WordPress Auto-Install & Management
- [ ] `wordpress-manager.go` - WP-CLI wrapper
- [ ] CLI: `juiscript wp install --domain <domain> --title <title> --admin-user <user> --admin-email <email>`
- [ ] CLI: `juiscript wp update --domain <domain>` - update WP core
- [ ] CLI: `juiscript wp secure --domain <domain>` - apply security hardening (disable XML-RPC, protect wp-admin, disable file editing)
- [ ] CLI: `juiscript wp plugins --domain <domain>` - list/update plugins
- [ ] TUI: WordPress screen with install/update/secure actions

### Cloud Backup (Google Drive / OneDrive via Rclone)
- [ ] Extend `backup-manager.go` - Rclone integration
- [ ] CLI: `juiscript backup create --domain <domain> --dest gdrive|onedrive|local`
- [ ] CLI: `juiscript backup restore --source gdrive|onedrive|local --path <path>`
- [ ] CLI: `juiscript backup connect-gdrive` - configure Rclone remote
- [ ] CLI: `juiscript backup connect-onedrive` - configure Rclone remote
- [ ] TUI: Backup destination selector in create form

### VPS Info & Monitoring
- [ ] `vps-manager.go` - system info from /proc/ and systemctl
- [ ] CLI: `juiscript vps info` - CPU, RAM, disk, uptime, OS version
- [ ] CLI: `juiscript vps disk` - disk usage per partition
- [ ] CLI: `juiscript vps processes` - top processes by CPU/memory
- [ ] TUI: VPS info dashboard screen

## Phase 4: Extended Features (Priority: MEDIUM)

### Domain Redirect & Alias
- [ ] CLI: `juiscript site add-alias --domain <domain> --alias <alias>` - parked/alias domain
- [ ] CLI: `juiscript site add-redirect --from <domain> --to <url>` - 301/302 redirect
- [ ] Nginx config generation for aliases and redirects

### Clone Website
- [ ] CLI: `juiscript site clone --source <domain> --dest <domain>` - duplicate site (files + DB)
- [ ] Copy webroot, DB dump/import, generate new vhost config

### SSH Hardening
- [ ] CLI: `juiscript vps ssh-port --port <port>` - change SSH port
- [ ] CLI: `juiscript vps create-swap --size <GB>` - create swap file
- [ ] Update UFW rules after SSH port change

### Cronjob Management
- [ ] `cron-manager.go` - programmatic crontab read/write
- [ ] CLI: `juiscript cron list` - list current crontab entries
- [ ] CLI: `juiscript cron add --schedule "<cron>" --command "<cmd>"` - add cron entry
- [ ] CLI: `juiscript cron remove --id <id>` - remove cron entry
- [ ] TUI: Cron screen with list/add/remove actions

### PHP Per-Site Config
- [ ] CLI: `juiscript php config --domain <domain> --open-basedir on|off`
- [ ] CLI: `juiscript php extensions --domain <domain> --install <ext>` - install PHP extension
- [ ] FPM pool config per-site tuning (memory_limit, upload_max_filesize, etc.)

### Directory Protection
- [ ] CLI: `juiscript site protect --domain <domain> --path <path>` - add HTTP basic auth
- [ ] CLI: `juiscript site unprotect --domain <domain> --path <path>` - remove protection
- [ ] Generate .htpasswd files and Nginx auth config

## Phase 5: Nice-to-Have Features (Priority: LOW)

### WordPress Security Hardening
- [ ] Disable XML-RPC per site
- [ ] Block plugin/theme file editing
- [ ] Disable user enumeration API
- [ ] Auto-configure cache plugins (WP-Rocket, W3TC, etc.)

### Nginx Pagespeed Module
- [ ] Minify JS/CSS/HTML
- [ ] Image compression & WebP conversion
- [ ] Combine JS/CSS files

### Telegram Notifications
- [ ] `telegram-manager.go` - Telegram Bot API wrapper
- [ ] SSH login alerts
- [ ] Service down alerts
- [ ] Disk space warnings
- [ ] CLI: `juiscript notify setup --bot-token <token> --chat-id <id>`

### Admin Panels
- [ ] phpMyAdmin install/update
- [ ] Redis admin panel
- [ ] Opcache dashboard
- [ ] Random port for admin panel security

### Security Tools
- [ ] ClamAV anti-virus install/scan
- [ ] ImunifyAV integration
- [ ] Malware scan scheduling

### Deployment Tools
- [ ] Git deploy from GitHub/GitLab repo
- [ ] Image compression tools (OptiPNG)
- [ ] Node.js/NPM installation

### Permissions Management
- [ ] CLI: `juiscript site fix-permissions --domain <domain>` - chown/chmod single site
- [ ] CLI: `juiscript site fix-permissions --all` - fix all sites

## Phase 6: Missing Documentation (Priority: LOW)

- [ ] `docs/design-guidelines.md`
- [ ] `docs/deployment-guide.md`
- [ ] `docs/project-roadmap.md`

## Notes

- Backend modules: 11/11 complete voi test coverage
- TUI screens: 8/8 co UI rendering, 8/8 wired to backend (v1.3.0)
- TUI components: FormModel, ConfirmModel, ToastModel, SpinnerModel (v1.4.0)
- 9 managers injected via AppDeps: service, provisioner, php, site, nginx, database, ssl, supervisor, backup
- 28 handler methods across 8 domain files (app_handlers_*.go)
- 28 result/error message types in app_messages.go
- Data auto-fetched on screen navigation
- 28 component unit tests (13 form + 9 confirm + 6 toast)
