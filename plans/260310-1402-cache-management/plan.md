---
title: "Cache Management (Redis/Opcache)"
description: "Add Redis enable/disable/flush and Opcache reset via CLI, TUI, and backend manager"
status: done
priority: P1
effort: 6h
branch: main
tags: [cache, redis, opcache, feature]
created: 2026-03-09
---

## Overview

Add cache management: Redis per-site (DB isolation), Opcache reset via PHP-FPM restart, CLI commands, and TUI screen. No Memcached for MVP.

## Dependency Graph

```
Phase 1: Backend Manager ──┐
                            ├──> Integration (main.go wiring)
Phase 2: CLI Commands ──────┘        │
                                     v
Phase 3: TUI Screen ────────────> Done
```

**Parallel**: Phase 1 + Phase 2 can start simultaneously (exclusive files).
**Sequential**: Phase 3 depends on Phase 1 completion (imports cache.Manager). Phase 2 + 3 share main.go edits (coordinate).

## File Ownership Matrix

| File | Phase 1 | Phase 2 | Phase 3 |
|------|---------|---------|---------|
| `internal/cache/manager.go` | OWNER | - | - |
| `internal/cache/manager_test.go` | OWNER | - | - |
| `cmd/juiscript/cmd-cache.go` | - | OWNER | - |
| `cmd/juiscript/main.go` | - | EDIT | EDIT |
| `internal/tui/screens/cache.go` | - | - | OWNER |
| `internal/tui/app_handlers_cache.go` | - | - | OWNER |
| `internal/tui/app.go` | - | - | EDIT |
| `internal/tui/app_messages.go` | - | - | EDIT |
| `internal/tui/screens/dashboard.go` | - | - | EDIT |

## Key Decisions

1. **Redis per-site**: Use database numbers (0-15) via `redis-cli -n {db}`. Track in site config.
2. **Opcache reset**: Restart PHP-FPM (`systemctl restart php{ver}-fpm`). Simplest, no external deps.
3. **No Memcached**: Skip for MVP. Redis covers primary use case.
4. **Dashboard key**: Use `c` key for Cache (letter-based, since 0-9 exhausted).
5. **Exec-only**: All Redis ops via `redis-cli` commands. No go-redis library.

## Phase Files

- [Phase 1: Backend Manager](./phase-01-backend-manager.md) ~2h
- [Phase 2: CLI Commands](./phase-02-cli-commands.md) ~1.5h
- [Phase 3: TUI Integration](./phase-03-tui-integration.md) ~2.5h

## Shared File Edit Coordination

**main.go** (Phase 2 adds Cache to Managers + CLI registration; Phase 3 adds CacheMgr to AppDeps):
- Phase 2 adds: `Cache *cache.Manager` field, init line, `rootCmd.AddCommand(cacheCmd(mgrs))`
- Phase 3 adds: `CacheMgr: mgrs.Cache` in runTUI AppDeps

**Conflict prevention**: Phase 2 handles all main.go edits (both CLI + TUI wiring) since it completes first.

## Validation Summary

**Validated:** 2026-03-10
**Questions asked:** 4

### Confirmed Decisions
- **Opcache reset**: Restart PHP-FPM confirmed. Accepted brief downtime for simplicity.
- **EnableRedis scope**: MVP chỉ kiểm tra Redis service running. KHÔNG tự sửa Laravel .env / WP wp-config.php. User tự cấu hình app.
- **DB-to-site mapping**: Lưu `redis_db` field trong site metadata JSON (`/etc/juiscript/sites/{domain}.json`). Extend site metadata struct.
- **FLUSHALL safety**: Confirm dialog trong TUI + `--force` flag cho CLI.

### Action Items
- [x] Phase 1: Add `redis_db` field to site metadata read/write (hoặc manager tự track DB assignment)
- [x] Phase 2: Add `--force` flag to `cache flush --all` CLI command
- [x] Phase 3: Add confirm dialog trước FLUSHALL và Opcache reset trong TUI screen

## Completion Summary

All three phases complete. Cache management feature implemented:
- Backend manager with Redis/Opcache operations via system executor pattern
- CLI commands for status, enable/disable, flush, and opcache reset
- TUI integration with cache screen, confirmation dialogs, and toast notifications
