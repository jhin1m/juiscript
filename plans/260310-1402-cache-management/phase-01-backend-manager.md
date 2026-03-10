# Phase 1: Backend Manager

## Context
- [Plan](./plan.md) | [Research: Cache Services](./research/researcher-01-cache-services-ubuntu.md) | [Research: Go Patterns](./research/researcher-02-go-cache-patterns.md)
- **Parallel with**: Phase 2 (no shared files)
- **Blocks**: Phase 3 (imports cache.Manager)

## Overview
Create `internal/cache/manager.go` with Redis and Opcache operations using exec-based pattern (matching firewall/manager.go).

## Key Insights
- All Redis ops via `redis-cli` (no go-redis library)
- Opcache reset = restart PHP-FPM service (simplest, no cachetool dep)
- DB number validation: 0 to `config.Redis.MaxDatabases - 1`
- Redis status check: `redis-cli PING` returns "PONG"

## File Ownership
| File | Role |
|------|------|
| `internal/cache/manager.go` | EXCLUSIVE |
| `internal/cache/manager_test.go` | EXCLUSIVE |

## Architecture

```go
package cache

type CacheStatus struct {
    RedisRunning  bool
    RedisVersion  string
    RedisMemory   string   // human-readable used memory
    OpcacheActive bool
}

type Manager struct {
    executor system.Executor
    cfg      *config.Config
}

func NewManager(exec system.Executor, cfg *config.Config) *Manager
```

## Implementation Steps

### Step 1: manager.go - Core struct and constructor

```go
package cache

import (
    "context"
    "fmt"
    "strconv"
    "strings"

    "github.com/jhin1m/juiscript/internal/config"
    "github.com/jhin1m/juiscript/internal/system"
)

type CacheStatus struct {
    RedisRunning  bool
    RedisVersion  string
    RedisMemory   string
}

type Manager struct {
    executor system.Executor
    cfg      *config.Config
}

func NewManager(exec system.Executor, cfg *config.Config) *Manager {
    return &Manager{executor: exec, cfg: cfg}
}
```

### Step 2: Redis status/info

```go
func (m *Manager) Status(ctx context.Context) (*CacheStatus, error) {
    status := &CacheStatus{}
    out, err := m.executor.Run(ctx, "redis-cli", "PING")
    if err != nil {
        return status, nil // Redis not running, not an error
    }
    status.RedisRunning = strings.TrimSpace(out) == "PONG"

    if status.RedisRunning {
        info, _ := m.executor.Run(ctx, "redis-cli", "INFO", "server", "memory")
        status.RedisVersion = parseRedisField(info, "redis_version")
        status.RedisMemory = parseRedisField(info, "used_memory_human")
    }
    return status, nil
}
```

### Step 3: Redis enable/disable per site

```go
func (m *Manager) validateDB(db int) error {
    max := m.cfg.Redis.MaxDatabases
    if max <= 0 { max = 16 }
    if db < 0 || db >= max {
        return fmt.Errorf("redis database must be 0-%d, got %d", max-1, db)
    }
    return nil
}

func (m *Manager) EnableRedis(ctx context.Context, domain string, db int) error {
    if err := m.validateDB(db); err != nil { return err }
    // Ensure Redis service is running
    _, err := m.executor.Run(ctx, "systemctl", "is-active", "redis-server")
    if err != nil {
        if _, startErr := m.executor.Run(ctx, "systemctl", "start", "redis-server"); startErr != nil {
            return fmt.Errorf("start redis: %w", startErr)
        }
    }
    // Verify connectivity
    out, err := m.executor.Run(ctx, "redis-cli", "PING")
    if err != nil || strings.TrimSpace(out) != "PONG" {
        return fmt.Errorf("redis not responding after start")
    }
    return nil
}

func (m *Manager) DisableRedis(ctx context.Context, domain string) error {
    // Flush site DB before disabling (caller should provide db number)
    return nil // Config-level disable handled by caller
}
```

### Step 4: Flush operations

```go
func (m *Manager) FlushDB(ctx context.Context, db int) error {
    if err := m.validateDB(db); err != nil { return err }
    _, err := m.executor.Run(ctx, "redis-cli", "-n", strconv.Itoa(db), "FLUSHDB")
    if err != nil { return fmt.Errorf("flush redis db %d: %w", db, err) }
    return nil
}

func (m *Manager) FlushAll(ctx context.Context) error {
    _, err := m.executor.Run(ctx, "redis-cli", "FLUSHALL")
    if err != nil { return fmt.Errorf("flush all redis: %w", err) }
    return nil
}
```

### Step 5: Opcache reset

```go
func (m *Manager) ResetOpcache(ctx context.Context, phpVersion string) error {
    if phpVersion == "" { return fmt.Errorf("php version required") }
    svc := fmt.Sprintf("php%s-fpm", phpVersion)
    _, err := m.executor.Run(ctx, "systemctl", "restart", svc)
    if err != nil { return fmt.Errorf("restart %s: %w", svc, err) }
    return nil
}
```

### Step 6: Helper - parse redis INFO output

```go
func parseRedisField(info, field string) string {
    for _, line := range strings.Split(info, "\n") {
        if strings.HasPrefix(line, field+":") {
            return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
        }
    }
    return ""
}
```

### Step 7: manager_test.go

Test with mock executor:
- `TestStatus_RedisRunning` - PING returns PONG
- `TestStatus_RedisDown` - PING returns error
- `TestFlushDB_Valid` - db=0 succeeds
- `TestFlushDB_InvalidDB` - db=99 returns validation error
- `TestResetOpcache` - calls systemctl restart
- `TestValidateDB_Bounds` - boundary checks

## Todo
- [x] Create `internal/cache/` directory
- [x] Implement `manager.go` with all methods
- [x] Implement `manager_test.go` with mock executor tests
- [x] Verify import path matches module: `github.com/jhin1m/juiscript/internal/cache`

## Success Criteria
- All methods follow executor pattern (no direct exec)
- Input validation on DB numbers
- Graceful handling when Redis not installed
- Tests pass with mock executor

## Risk Assessment
- **Low**: Opcache reset via FPM restart affects all sites on same PHP version. Acceptable for MVP since most setups use single PHP version.
- **Low**: Redis INFO parsing may differ across Redis versions. Parse defensively.
