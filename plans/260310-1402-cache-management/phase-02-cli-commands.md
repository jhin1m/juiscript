# Phase 2: CLI Commands

## Context
- [Plan](./plan.md) | Reference: `cmd/juiscript/cmd-firewall.go`
- **Parallel with**: Phase 1 (separate files, but imports Phase 1's package)
- **Note**: Can stub cache.Manager import initially; Phase 1 provides implementation

## Overview
Create `cmd/juiscript/cmd-cache.go` with Cobra subcommands and wire into main.go (including TUI deps for Phase 3).

## File Ownership
| File | Role |
|------|------|
| `cmd/juiscript/cmd-cache.go` | EXCLUSIVE |
| `cmd/juiscript/main.go` | EDIT (add Cache manager + CLI + TUI wiring) |

## Implementation Steps

### Step 1: cmd-cache.go - Root command

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    "github.com/spf13/cobra"
)

func cacheCmd(mgrs *Managers) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "cache",
        Short: "Manage cache services (Redis, Opcache)",
    }
    cmd.AddCommand(
        cacheStatusCmd(mgrs),
        cacheEnableRedisCmd(mgrs),
        cacheDisableRedisCmd(mgrs),
        cacheFlushCmd(mgrs),
        cacheOpcacheResetCmd(mgrs),
    )
    return cmd
}
```

### Step 2: Status subcommand

```go
func cacheStatusCmd(mgrs *Managers) *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show cache service status",
        RunE: func(cmd *cobra.Command, args []string) error {
            status, err := mgrs.Cache.Status(context.Background())
            if err != nil { return err }

            redisStr := "not running"
            if status.RedisRunning {
                redisStr = fmt.Sprintf("running (v%s, mem: %s)", status.RedisVersion, status.RedisMemory)
            }
            fmt.Fprintf(os.Stdout, "Redis: %s\n", redisStr)
            return nil
        },
    }
}
```

### Step 3: Enable/Disable Redis subcommands

```go
func cacheEnableRedisCmd(mgrs *Managers) *cobra.Command {
    var domain string
    var db int
    cmd := &cobra.Command{
        Use:   "enable-redis",
        Short: "Enable Redis for a site",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := mgrs.Cache.EnableRedis(context.Background(), domain, db); err != nil {
                return err
            }
            fmt.Printf("Redis enabled for %s (db: %d)\n", domain, db)
            return nil
        },
    }
    cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
    cmd.Flags().IntVar(&db, "db", 0, "Redis database number (0-15)")
    cmd.MarkFlagRequired("domain")
    return cmd
}

func cacheDisableRedisCmd(mgrs *Managers) *cobra.Command {
    var domain string
    cmd := &cobra.Command{
        Use:   "disable-redis",
        Short: "Disable Redis for a site",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := mgrs.Cache.DisableRedis(context.Background(), domain); err != nil {
                return err
            }
            fmt.Printf("Redis disabled for %s\n", domain)
            return nil
        },
    }
    cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
    cmd.MarkFlagRequired("domain")
    return cmd
}
```

### Step 4: Flush subcommand

```go
func cacheFlushCmd(mgrs *Managers) *cobra.Command {
    var domain string
    var db int
    var all bool
    cmd := &cobra.Command{
        Use:   "flush",
        Short: "Flush Redis cache",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := context.Background()
            if all {
                if err := mgrs.Cache.FlushAll(ctx); err != nil { return err }
                fmt.Println("All Redis databases flushed")
                return nil
            }
            if err := mgrs.Cache.FlushDB(ctx, db); err != nil { return err }
            fmt.Printf("Redis database %d flushed\n", db)
            return nil
        },
    }
    cmd.Flags().StringVar(&domain, "domain", "", "Site domain")
    cmd.Flags().IntVar(&db, "db", 0, "Redis database number")
    cmd.Flags().BoolVar(&all, "all", false, "Flush all Redis databases")
    return cmd
}
```

### Step 5: Opcache reset subcommand

```go
func cacheOpcacheResetCmd(mgrs *Managers) *cobra.Command {
    var phpVersion string
    cmd := &cobra.Command{
        Use:   "opcache-reset",
        Short: "Reset PHP Opcache by restarting PHP-FPM",
        RunE: func(cmd *cobra.Command, args []string) error {
            ver := phpVersion
            if ver == "" { ver = mgrs.Cfg.PHP.DefaultVersion }
            if err := mgrs.Cache.ResetOpcache(context.Background(), ver); err != nil {
                return err
            }
            fmt.Printf("Opcache reset (PHP %s FPM restarted)\n", ver)
            return nil
        },
    }
    cmd.Flags().StringVar(&phpVersion, "php-version", "", "PHP version (default: from config)")
    return cmd
}
```

### Step 6: main.go edits

Add to `Managers` struct:
```go
Cache *cache.Manager
```

Add to `initManagers()`:
```go
cacheMgr := cache.NewManager(exec, cfg)
```

Add to return:
```go
Cache: cacheMgr,
```

Add import:
```go
"github.com/jhin1m/juiscript/internal/cache"
```

Add CLI registration:
```go
rootCmd.AddCommand(cacheCmd(mgrs))
```

Add TUI wiring (for Phase 3):
```go
// In runTUI AppDeps:
CacheMgr: mgrs.Cache,
```

## Todo
- [x] Create `cmd/juiscript/cmd-cache.go`
- [x] Edit `cmd/juiscript/main.go`: add Cache to Managers, init, CLI + TUI wiring
- [x] Test CLI commands manually: `juiscript cache status`, `juiscript cache flush --all`

## Success Criteria
- All 5 subcommands registered under `cache`
- Flag validation matches existing patterns (--domain required where needed)
- main.go compiles with cache import
- `juiscript cache --help` shows all subcommands

## Conflict Prevention
- **main.go**: Phase 2 owns ALL main.go edits (Managers struct, initManagers, CLI reg, AND TUI AppDeps). Phase 3 does NOT touch main.go.

## Security Considerations
- `--domain` flag validated (no shell injection since executor handles quoting)
- DB number validated (0-15 range)
- Requires root (existing PersistentPreRunE check)
