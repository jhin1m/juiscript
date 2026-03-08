---
title: "Wire TUI Screens to Backend Managers"
description: "Connect 22 TODO handlers in app.go to backend managers via async command pattern"
status: in-progress
priority: P1
effort: 6h
branch: main
tags: [tui, wiring, backend, managers]
created: 2026-03-06
phase_1_completed: 2026-03-06
---

# TUI Backend Wiring Plan

## Problem
22 TODO handlers in `internal/tui/app.go` are stubs. Screens render but can't trigger backend operations.

## Strategy: Extract Handler Files

**Key insight:** app.go is the bottleneck. We split handler logic into separate files in the same `tui` package, keeping app.go changes minimal (struct + init + import routing).

### Dependency Graph
```
Phase 1 (Sequential) ─── Foundation: managers in App struct + init + result messages
    |
    +── Phase 2 (Parallel) ─── Site & Nginx handlers  [app_handlers_site.go, app_handlers_nginx.go]
    +── Phase 3 (Parallel) ─── PHP & Database handlers [app_handlers_php.go, app_handlers_db.go]
    +── Phase 4 (Parallel) ─── SSL & Services handlers [app_handlers_ssl.go, app_handlers_service.go]
    +── Phase 5 (Parallel) ─── Queues & Backup handlers [app_handlers_queue.go, app_handlers_backup.go]
    |
Phase 6 (Sequential) ─── Wire handlers in app.go Update() + data-fetch on navigation
```

### File Ownership Matrix

| Phase | Owns (exclusive write) | Reads only |
|-------|----------------------|------------|
| 1 | `app.go`, `app_messages.go`, `main.go` | manager packages |
| 2 | `app_handlers_site.go`, `app_handlers_nginx.go` | app.go (struct) |
| 3 | `app_handlers_php.go`, `app_handlers_db.go` | app.go (struct) |
| 4 | `app_handlers_ssl.go`, `app_handlers_service.go` | app.go (struct) |
| 5 | `app_handlers_queue.go`, `app_handlers_backup.go` | app.go (struct) |
| 6 | `app.go` (Update switch-case wiring) | all handler files |

### Manager Initialization Order (Phase 1)
```
exec, fileMgr, tplEngine, userMgr (shared deps from main.go)
    |
    +── nginxMgr  = nginx.NewManager(exec, fileMgr, tplEngine, cfg paths)
    +── dbMgr     = database.NewManager(exec)
    +── siteMgr   = site.NewManager(cfg, exec, fileMgr, userMgr, tplEngine)
    +── sslMgr    = ssl.NewManager(exec, nginxMgr, fileMgr)
    +── supervisorMgr = supervisor.NewManager(exec, fileMgr, tplEngine)
    +── backupMgr = backup.NewManager(cfg, exec, fileMgr, dbMgr)
```

## Phases Summary

| Phase | Scope | Parallel | Files | Effort |
|-------|-------|----------|-------|--------|
| 1 | Foundation: managers + messages + init | No | 3 | 1.5h |
| 2 | Site + Nginx handlers | Yes (with 3-5) | 2 | 1h |
| 3 | PHP + Database handlers | Yes (with 2,4,5) | 2 | 1h |
| 4 | SSL + Service handlers | Yes (with 2,3,5) | 2 | 0.5h |
| 5 | Queue + Backup handlers | Yes (with 2-4) | 2 | 0.5h |
| 6 | Wire handlers into app.go Update() | No | 1 | 1.5h |

## Success Criteria
- All 22 TODOs replaced with async manager calls
- Each screen loads data on navigation (fetchX pattern)
- Errors surface via screen.SetError()
- Existing tests still pass (`make test`)
- No new files outside `internal/tui/` and `cmd/juiscript/`

## Validation Summary

**Validated:** 2026-03-06
**Questions asked:** 4

### Confirmed Decisions
- **Placeholder errors OK:** 4 operations (CreateDB, ImportDB, ObtainCert, CreateBackup) return informative errors until form screens are built in a separate plan
- **No confirmation dialogs for v1:** Destructive operations proceed without confirmation. YAGNI — add later
- **PHP RemoveVersion safety:** Fetch sites list via siteMgr.List() before calling RemoveVersion, pass actual sites instead of nil
- **8 handler files approved:** One file per domain (app_handlers_*.go), parallel-friendly structure

### Action Items
- [ ] Update Phase 3: `handleRemovePHP` must call `a.siteMgr.List()` first and pass sites to `RemoveVersion` instead of nil
