# Cache Services Management: Redis, Memcached, PHP Opcache
**Research Report** | Ubuntu 22.04/24.04 Hosting Management Tool

---

## 1. Redis Per-Site Configuration

### Architecture Option: Shared Instance with Separate DBs
Single Redis instance supports 16 databases (0-15) by default. Multi-site isolation via database numbers.

**Configuration (per site in Laravel config/database.php):**
```php
'redis' => [
    'default' => ['host' => '127.0.0.1', 'port' => 6379, 'database' => 0],
    'cache' => ['host' => '127.0.0.1', 'port' => 6379, 'database' => 1],
    'session' => ['host' => '127.0.0.1', 'port' => 6379, 'database' => 2],
]
```

### Connection Methods
- **TCP**: Default. Bind to 127.0.0.1 for localhost or 0.0.0.0 for network access.
- **Unix Socket** (`/var/run/redis/redis.sock`): Faster, lower overhead. Configure in `/etc/redis/redis.conf`:
  ```
  unixsocket /var/run/redis/redis.sock
  unixsocketperm 700
  port 0
  ```

### Authentication
Enable ACL in `/etc/redis/redis.conf`:
```
requirepass your_password
# or modern (Redis 6+):
user default on >password ~* &* +@all
```

Connect with password:
```bash
redis-cli -h 127.0.0.1 -p 6379 -a password
# or via socket:
redis-cli -s /var/run/redis/redis.sock -a password
```

### Limitation
Redis Cluster doesn't support multiple databases (only DB 0). Use key prefixes for isolation instead.

---

## 2. Memcached Per-Site Configuration

### Instance-Per-Site Approach (Recommended)
Run separate Memcached instances on different ports per site:
```bash
# Site 1 - Port 11211
memcached -p 11211 -u memcache -d -m 256

# Site 2 - Port 11212
memcached -p 11212 -u memcache -d -m 256
```

Or use systemd socket activation for per-site instances:
```ini
[Unit]
Description=Memcached for Site1
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/memcached -p 11211 -u memcache -m 256
```

### SASL Authentication (Single Instance)
Requires: `sasl2-bin` and `libmemcached` with SASL support.

**Setup:**
```bash
sudo saslpasswd2 -c -u memcached siteuser1
# Add to /etc/memcached.conf:
-S
```

**PHP Connection with SASL:**
```php
$mem = new Memcached();
$mem->setOption(Memcached::OPT_BINARY_PROTOCOL, true);
$mem->setSaslAuthData('siteuser1', 'password');
$mem->addServer('localhost', 11211);
```

### Cache Flushing
```bash
# Flush all items
memcached-tool 127.0.0.1:11211 flush

# Or via CLI:
echo "flush_all" | nc localhost 11211
```

---

## 3. PHP Opcache Management

### Problem
CLI `opcache_reset()` doesn't affect FPM cache (separate process). CLI and FPM have isolated opcaches.

### Solution 1: CacheTool (Recommended)
```bash
# Install
curl -sLO https://github.com/gordalina/cachetool/releases/latest/download/cachetool.phar
chmod +x cachetool.phar

# Reset opcache via PHP-FPM socket
php cachetool.phar opcache:reset --fcgi=/var/run/php/php8.2-fpm.sock

# Or TCP
php cachetool.phar opcache:reset --fcgi=127.0.0.1:9000

# View status
php cachetool.phar opcache:status --fcgi=/var/run/php/php8.2-fpm.sock
```

### Solution 2: PHP-FPM Reload
```bash
# Full restart (clears opcache)
sudo systemctl restart php8.2-fpm

# Graceful reload (clears opcache on next request)
sudo systemctl reload php8.2-fpm

# Per pool:
php-fpm8.2 -t  # test config
sudo kill -USR2 $(cat /var/run/php-fpm8.2.pid)
```

### Solution 3: Alternative Tool (Chop)
Lightweight alternative to CacheTool for CLI-based opcache reset.

### Opcache Reset Limitations
- Cannot reset from CLI directly (CLI has separate cache)
- Must target FPM process or reload service
- Per-site isolation: Use separate PHP-FPM pools if per-site resets needed

---

## 4. Cache Flushing Commands Summary

| Service | Command | Effect |
|---------|---------|--------|
| Redis (all) | `redis-cli FLUSHALL` | Clear all DBs |
| Redis (single DB) | `redis-cli -n 0 FLUSHDB` | Clear DB 0 only |
| Redis (pattern) | `redis-cli --eval script.lua 0` | Key pattern flush |
| Memcached | `memcached-tool 127.0.0.1:11211 flush` | Flush all items |
| Opcache | `php cachetool.phar opcache:reset --fcgi=/run/php/php8.2-fpm.sock` | Clear FPM cache |
| Nginx FastCGI | `sudo rm -rf /var/cache/nginx/*` | Clear disk cache |

---

## 5. Nginx FastCGI vs Redis Caching Layers

### Nginx FastCGI Cache
- **Storage**: Disk-based (doesn't consume RAM)
- **Scope**: Full HTML page caching
- **Use Case**: Static/semi-static content, high concurrency
- **Config**:
  ```nginx
  fastcgi_cache_path /var/cache/nginx levels=1:2 keys_zone=CACHE:10m;
  fastcgi_cache CACHE;
  fastcgi_cache_valid 200 10m;
  ```

### Redis Object Cache (WordPress/Laravel)
- **Storage**: In-memory (RAM-bound)
- **Scope**: Database query results, object caching
- **Use Case**: Dynamic content, lower concurrency, persistent sessions
- **WordPress Plugin**: Redis Object Cache Pro

### Combined Approach (Recommended)
1. **Nginx FastCGI**: Page-level HTML caching (first layer)
2. **Redis Object Cache**: Query/object caching (second layer)

Performance: Redis page caching slightly faster but costs RAM. FastCGI better for high traffic with disk space.

---

## Unresolved Questions
- How to implement per-pool Opcache reset for multi-site setups?
- Best practices for Redis memory limits with multi-database setups?
- Memcached vs Redis trade-offs for WordPress multi-site?

## Sources
- [Redis Per-Site Configuration](https://aregsar.com/blog/2020/create-laravel-project-with-multiple-redis-stores/)
- [Laravel Redis Documentation](https://laravel.com/docs/12.x/redis)
- [CacheTool - Opcache Management](https://gordalina.github.io/cachetool/)
- [PHP Opcache Reset Methods](https://www.10gbhosting.com/tutorial-on-how-to-flush-php-opcache/)
- [Memcached SASL Authentication](https://www.linode.com/docs/guides/install-and-secure-memcached-on-debian-11-and-ubuntu-2204/)
- [Redis Security & AUTH](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)
- [Nginx FastCGI vs Redis Caching](https://wp-rocket.me/wordpress-cache/redis-full-page-cache-vs-nginx-fastcgi/)
- [WP Rocket Comparison](https://runcloud.io/blog/redis-full-page-vs-nginx-fastcgi-caching)
