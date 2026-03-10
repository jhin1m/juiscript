# Go Cache Management Implementation Patterns
**Research Report** | 2026-03-09

## 1. Interface Design for Cache Manager

**Pattern used by similar tools** (Ploi, Forge, Cyberpanel):
```go
type Manager struct {
    executor system.Executor  // For exec-based operations
    config   *config.Config   // Redis connection details
}

type CacheOp interface {
    FlushRedis(ctx context.Context, siteID string) error
    EnableRedis(ctx context.Context, siteID string, db int) error
    DisableRedis(ctx context.Context, siteID string) error
    ResetOpcache(ctx context.Context, phpVersion string) error
    GetStatus(ctx context.Context, siteID string) (CacheStatus, error)
}

type CacheStatus struct {
    RedisEnabled bool
    OpcacheSize  int64
    RedisDB      int
}
```

**Key design choice**: Follow `juiscript` pattern - use dependency-injected `system.Executor` for all CLI calls. Avoid direct `redis` Go library since this is CLI-based system management, not an application server.

---

## 2. Redis Go Client Patterns (CLI vs Library)

### Why Exec Approach is Preferred for juiscript:
- **Tool type**: System management CLI (like Ploi, Forge)
- **Responsibility**: Execute commands, don't maintain app-level connections
- **Risk profile**: Lower - exec failures are atomic and isolated

### Implementation Pattern:
```go
// Use redis-cli via exec (preferred)
func (m *Manager) FlushRedis(ctx context.Context, db int) error {
    return m.executor.Run(ctx, "redis-cli", "-n", fmt.Sprintf("%d", db), "FLUSHDB")
}

// For connection validation only:
func (m *Manager) CheckRedisConnection(ctx context.Context) error {
    _, err := m.executor.Run(ctx, "redis-cli", "PING")
    return err
}

// Do NOT use go-redis library here:
// - Adds runtime dependency complexity
// - Requires connection pooling management
// - Not aligned with CLI tool architecture
// - Risk of leaving connections open if tool crashes
```

### Async Operations Pattern:
If status checking needed during long operations, use exec with timeout:
```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
output, err := m.executor.Run(ctx, "redis-cli", "--latency")
```

---

## 3. PHP-FPM Opcache Reset Approaches

### Three Strategies (Ranked by Safety for Management Tool):

**1. PHP Script Injection (SAFEST for server management)**
```go
// Write temp script, execute via PHP CLI, clean up
func (m *Manager) ResetOpcache(ctx context.Context, phpVersion string) error {
    script := `<?php opcache_reset(); echo 'OK'; ?>`
    tempFile := fmt.Sprintf("/tmp/opcache_reset_%d.php", time.Now().UnixNano())

    if err := ioutil.WriteFile(tempFile, []byte(script), 0644); err != nil {
        return err
    }
    defer os.Remove(tempFile)

    return m.executor.Run(ctx, fmt.Sprintf("php%s", phpVersion), tempFile)
}
```
- **Why safest**: Immediate, no service restart needed, works per-site
- **Risk**: Minimal - isolated script execution

**2. Cachetool (Next best)**
```go
// Install: composer global require gordalina/cachetool
func (m *Manager) ResetOpcacheViaCachetool(ctx context.Context, fpmSocket string) error {
    return m.executor.Run(ctx, "cachetool",
        "opcache:reset",
        fmt.Sprintf("--fcgi=%s", fpmSocket))
}
```
- **Why good**: Designed for this, works with FPM pools
- **Risk**: Requires cachetool installation, external dependency

**3. FPM Restart (AVOID for management tool)**
```go
// LAST RESORT - service restart
func (m *Manager) ResetOpcacheViaRestart(ctx context.Context, phpVersion string) error {
    return m.executor.Run(ctx, "systemctl", "restart",
        fmt.Sprintf("php%s-fpm", phpVersion))
}
```
- **Why avoid**: Kills all connections, causes downtime, overkill for reset
- **Risk**: High - affects all sites using that PHP version

**Recommendation**: Use PHP script injection as default (option 1). No external deps, instant, safe.

---

## 4. Per-Site Redis Configuration

### Laravel Configuration Pattern:
```go
// In .env file (per site)
func (m *Manager) ConfigureLaravelRedis(ctx context.Context, siteDir string, db int) error {
    envFile := filepath.Join(siteDir, ".env")
    envContent, _ := ioutil.ReadFile(envFile)

    // Update REDIS_DB only
    newEnv := regexp.MustCompile(`REDIS_DB=\d+`).
        ReplaceAllString(string(envContent), fmt.Sprintf("REDIS_DB=%d", db))

    return ioutil.WriteFile(envFile, []byte(newEnv), 0644)
}
```

### WordPress Configuration Pattern:
```go
// In wp-config.php (per site)
func (m *Manager) ConfigureWordPressRedis(ctx context.Context, siteDir string, db int) error {
    wpConfig := filepath.Join(siteDir, "wp-config.php")
    content, _ := ioutil.ReadFile(wpConfig)

    def := fmt.Sprintf("define('WP_REDIS_DATABASE', %d);", db)
    newContent := regexp.MustCompile(`define\('WP_REDIS_DATABASE'.*?\);`).
        ReplaceAllString(string(content), def)

    return ioutil.WriteFile(wpConfig, []byte(newContent), 0644)
}
```

### Key Points:
- Each site gets unique Redis DB number (0-15 are typical)
- Modify .env or wp-config.php, not Redis side
- Validate DB number before applying (0-15 for standard Redis)

---

## 5. Cobra CLI Subcommand Structure

### Recommended Pattern (Matching juiscript architecture):
```go
// cmd/cache.go
var cacheCmd = &cobra.Command{
    Use:   "cache",
    Short: "Manage site caches",
}

func init() {
    rootCmd.AddCommand(cacheCmd)

    cacheCmd.AddCommand(&cobra.Command{
        Use:   "enable-redis <site-id> [--db=0]",
        Short: "Enable Redis for a site",
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr := cache.NewManager(executor, config)
            return mgr.EnableRedis(ctx, args[0], dbNum)
        },
    })

    cacheCmd.AddCommand(&cobra.Command{
        Use:   "disable-redis <site-id>",
        Short: "Disable Redis for a site",
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr := cache.NewManager(executor, config)
            return mgr.DisableRedis(ctx, args[0])
        },
    })

    cacheCmd.AddCommand(&cobra.Command{
        Use:   "flush <site-id>",
        Short: "Flush all caches for a site",
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr := cache.NewManager(executor, config)
            return mgr.FlushAll(ctx, args[0])
        },
    })

    cacheCmd.AddCommand(&cobra.Command{
        Use:   "opcache-reset <site-id>",
        Short: "Reset PHP Opcache for a site",
        RunE: func(cmd *cobra.Command, args []string) error {
            mgr := cache.NewManager(executor, config)
            phpVer := getSitePhpVersion(args[0])
            return mgr.ResetOpcache(ctx, phpVer)
        },
    })
}
```

### Multi-Step Command Pattern:
```go
cacheCmd.AddCommand(&cobra.Command{
    Use:   "flush-all",
    Short: "Flush Redis + Opcache for all sites",
    RunE: func(cmd *cobra.Command, args []string) error {
        sites := loadAllSites()
        for _, site := range sites {
            mgr.FlushRedis(ctx, site.RedisDB)
            mgr.ResetOpcache(ctx, site.PhpVersion)
        }
        return nil
    },
})
```

---

## Architecture Alignment

**Follow `juiscript` patterns**:
- ✓ Manager pattern with injected `system.Executor`
- ✓ No external library dependencies (exec only)
- ✓ Cobra subcommand structure
- ✓ Context-based cancellation
- ✓ Per-site configuration in site directory

**Example file layout**:
```
internal/cache/
  manager.go           (Core Manager + interface)
  redis.go             (Redis CLI operations)
  opcache.go           (PHP Opcache operations)
  config.go            (Per-site config files)
  manager_test.go      (Unit tests)
cmd/
  cache.go             (Cobra subcommands)
```

---

## Unresolved Questions

1. Should `FlushAll` include database query cache (MySQL), or Redis only?
2. Redis password support needed, or assume local socket auth?
3. Should Opcache reset validate script execution (read temp file output)?
