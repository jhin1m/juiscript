# LEMP Management Tools Research Report

## Executive Summary
Analyzed 7+ open-source LEMP tools (ServerPilot, RunCloud, CyberPanel, WordOps, EasyEngine) + architectural patterns for multi-site WordPress/Laravel hosting. Key takeaway: successful tools leverage modular config management, PHP-FPM pooling, and container-like isolation.

---

## 1. Similar Tools & Feature Patterns

### Tier-1 Commercial Tools
- **ServerPilot**: Cloud-agnostic (AWS, DigitalOcean, Azure). Nginx native. Focus on simplicity.
- **RunCloud**: Nginx + optional Apache. Native Redis/Memcached. Full git/backup integration.
- **CyberPanel**: OpenLiteSpeed-based (not Nginx). Python/Flask backend, React frontend. Git deployment built-in.

### Open-Source Alternatives
- **WordOps** (EasyEngine v3 fork): CLI-first, Python-based. TLS 1.3, HTTP/3 support. Nginx + FastCGI/Redis cache.
- **EasyEngine v4**: Moved to Docker (controversial). v3 remains preferred by ops teams.
- **aaPanel**: Free, Python/Flask UI. One-click LAMP/LEMP. Modular plugin system (backup, DNS, Fail2ban).

**Pattern**: All successful tools provide:
- Abstraction over native Linux tools (vhost creation = wrapper around Nginx/config generation)
- CLI + Web UI duality (CLI for scripting, UI for management)
- Database/user auto-provisioning
- SSL/Let's Encrypt automation

---

## 2. Nginx Vhost Management Structure

### Recommended Layout
```
/etc/nginx/
├── nginx.conf          # Main config (include directives)
├── conf.d/             # Global settings (rate limits, log formats)
├── sites-available/    # All vhost configs (domain.conf)
└── sites-enabled/      # Symlinks to active sites (symlink sites-available/*)
```

### Key Patterns
- **Symlink strategy**: Enable/disable by managing symlinks, not deleting files
- **Per-domain files**: One file per domain (e.g., `example.com.conf`) vs monolithic config
- **Reload safety**: Test config before reload (`nginx -t`)

### sites-available vs conf.d
- `sites-available/sites-enabled`: Debian convention, selective enable/disable
- `conf.d`: Direct includes, processes before sites-enabled (precedence matters)
- **Hybrid approach**: Use `conf.d` for global settings, `sites-available` for vhosts

**Decision**: Use `sites-available/sites-enabled` + symlinks (better auditability for multi-site)

---

## 3. Multi-PHP Version Management

### Implementation via ondrej/php PPA
```bash
add-apt-repository ppa:ondrej/php
apt install php7.4-fpm php8.0-fpm php8.1-fpm
# Each installs as: php7.4-fpm, php8.0-fpm, etc.
```

### FPM Pool Configuration
- **Socket location**: `/run/php/php{VERSION}-fpm.sock`
- **Pool config**: `/etc/php/{VERSION}/fpm/pool.d/www.conf`
- **Per-site pooling**: Create custom pools per version + site
  - Example: `site-example-7.4.conf`, `site-example-8.0.conf`

### Nginx Routing Pattern
```nginx
# Proxy to specific PHP-FPM socket based on site requirements
location ~ \.php$ {
    fastcgi_pass unix:/run/php/php7.4-fpm.sock;  # or php8.0-fpm.sock
}
```

**Key tuning**: Adjust `pm.max_children` per pool based on traffic + memory

---

## 4. WordPress + Laravel Deployment Patterns

### WordPress-specific
- **wp-cli integration**: Automate install, plugin management, migrations
- **Directory structure**: `/home/{user}/public_html/{domain}/`
- **Permissions**: `{user}:{user}` for content, `www-data:www-data` for caches
- **Config**: `wp-config.php` with environment-based DB constants

### Laravel-specific
- **Artisan automation**: Seed DB, run migrations via `php artisan` commands
- **Directory structure**: `/home/{user}/public_html/{domain}/public` (web root) + `/home/{user}/public_html/{domain}` (app root)
- **Environment**: `.env` file per site with unique DB/Redis configs
- **Queue workers**: Supervisor manages async job processing (next section)

### Database per Site
- Isolated DB credential per site (user + password unique to domain)
- Naming: `db_domain_com` or `domain_production`

---

## 5. Supervisor for Laravel Queue Workers

### Configuration Pattern
```ini
# /etc/supervisor/conf.d/site-example-queue.conf
[program:queue-example-com]
command=php /home/user/public_html/example.com/artisan queue:work --tries=3
process_name=%(program_name)s_%(process_num)02d
autostart=true
autorestart=true
user=user
numprocs=5
startsecs=0
stdout_logfile=/var/log/supervisor/%(program_name)s.log
redirect_stderr=true
```

### Multi-site Strategy
- One supervisor config per site/domain
- Naming convention: `{domain}-queue.conf`
- Each worker connects to site-specific queue (Redis/database)
- Commands: `supervisorctl reread`, `supervisorctl update`, `supervisorctl start site-example-queue:*`

**Key**: Per-site worker isolation prevents cross-site job pollution

---

## 6. MariaDB User & Database Management

### Pattern: Role-Based Access Control
```sql
CREATE ROLE 'wp_example_role';
GRANT SELECT, INSERT, UPDATE, DELETE ON `db_example_com`.* TO 'wp_example_role';
CREATE USER 'wp_example'@'localhost' IDENTIFIED BY 'password';
GRANT 'wp_example_role' TO 'wp_example'@'localhost';
```

### Per-Site Isolation
- **User**: `{prefix}_{domain_slug}` (e.g., `wp_example_com`)
- **Database**: `db_{domain_slug}` (e.g., `db_example_com`)
- **Privileges**: SELECT, INSERT, UPDATE, DELETE on own DB only (no SUPER, GRANT, etc.)
- **Host whitelist**: Use `localhost` or specific IP instead of `%`

### Best Practices
- Rotate passwords quarterly
- Use strong random passwords (30+ chars)
- No wildcard `%` host access

---

## 7. Redis Isolation Per Site

### Database Number Strategy
```
Site 1 (example.com):   SELECT 0  (cache)
Site 2 (example.org):   SELECT 1  (cache)
Site N:                 SELECT N
```

### Configuration
- Default: 16 databases (0-15), configurable to 256 in `redis.conf`
- **Lookup pattern**: Use `SELECT {db_number}` in app config
- Each DB has isolated keyspace

### Limitations & Alternatives
- **Limitation**: All databases share eviction policy, memory limits (configuration-wide)
- **Alternative for strong isolation**: Separate Redis instances per site (port 6379 + N)
- **Hybrid**: Use DB numbers for cache (relaxed isolation), separate instances for queues (strict isolation)

**Recommendation**: Use DB numbers for cache/sessions, separate instances for mission-critical queues

---

## 8. Integration Architecture (Synthesized)

### Control Panel Data Flow
```
API Request
  ↓
Validation Layer
  ↓
Business Logic (create vhost, DB, SSL, etc.)
  ↓
System Config Writers (templates → actual files)
  ↓
Service Reloaders (nginx -s reload, supervisorctl update, etc.)
```

### File Generation Patterns
- **Templates** (Jinja2/Python): Store as templates in code
- **Variables**: Site name, domain, PHP version, DB credentials injected at runtime
- **Output paths**: Write to actual system paths (`/etc/nginx/sites-available/`, etc.)
- **Validation**: Syntax check before reload (critical)

### Atomic Operations
- All site creation = atomic (DB + vhost + SSL or rollback)
- Idempotent: Re-running provisioning doesn't corrupt existing configs

---

## Key Insights & Best Practices

1. **Modular config over monolithic**: Per-site configs enable granular enable/disable
2. **Symlinks for activation**: Don't delete configs, manage symlinks
3. **PHP-FPM pooling**: Per-version support requires separate sockets + careful routing
4. **Supervisor per-site**: Queue isolation prevents cross-tenant issues
5. **MariaDB roles**: Better than direct user grants for privilege management
6. **Redis DB numbers**: Acceptable for cache, separate instances for persistence
7. **Atomic provisioning**: DB + config + permissions created together or not at all
8. **Template-based generation**: Reduces config drift, enables version control

---

## Unresolved Questions

1. How to handle SSL wildcard renewal across multiple domains (ACME challenge routing)?
2. Optimal supervisor worker count per site (CPU cores vs memory tradeoff)?
3. Log aggregation strategy for multi-site (centralized vs per-site files)?
4. Backup strategy for isolated databases + Redis data per site?

---

## Sources

- [ServerPilot, RunCloud, CyberPanel Comparison](https://cyberpanel.net/blog/runcloud-alternative)
- [WordOps Documentation](https://wordops.net/)
- [Nginx Virtual Host Configuration](https://www.getpagespeed.com/server-setup/nginx/nginx-virtual-host-multiple-domains)
- [Ondrej PHP PPA](https://launchpad.net/~ondrej/+archive/ubuntu/php)
- [Laravel Queue & Supervisor](https://laravel.com/docs/12.x/queues)
- [MariaDB User Management](https://mariadb.com/docs/server/reference/sql-statements/account-management-sql-statements/grant)
- [Redis Multiple Databases](https://oneuptime.com/blog/post/2026-01-25-redis-multiple-databases/view)
- [CyberPanel GitHub](https://github.com/usmannasir/cyberpanel)
- [Nginx conf.d vs sites-available](https://www.baeldung.com/linux/sites-available-sites-enabled-conf-d)
