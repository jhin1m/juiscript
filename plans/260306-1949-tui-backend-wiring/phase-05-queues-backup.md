---
phase: 5
title: "Queue & Backup Handler Methods"
status: pending
depends_on: [1]
parallel: true
parallel_with: [2, 3, 4]
effort: 0.5h
---

# Phase 5: Queue & Backup Handlers

## Context
Queue worker operations (4 TODOs) and Backup operations (3 TODOs). Queue handlers use supervisor.Manager, backup handlers use backup.Manager.

## Parallelization
Runs in parallel with Phases 2, 3, 4.

## File Ownership (exclusive)
- `internal/tui/app_handlers_queue.go` -- NEW
- `internal/tui/app_handlers_backup.go` -- NEW

## app_handlers_queue.go

**fetchWorkers()** -- Load worker list
```go
func (a *App) fetchWorkers() tea.Cmd {
    if a.supervisorMgr == nil { return nil }
    return func() tea.Msg {
        workers, err := a.supervisorMgr.ListAll(context.Background())
        if err != nil { return WorkerListErrMsg{Err: err} }
        return WorkerListMsg{Workers: workers}
    }
}
```

**handleWorkerAction(name, action)** -- Consolidated start/stop/restart
```go
func (a *App) handleWorkerAction(name string, action string) tea.Cmd {
    if a.supervisorMgr == nil { return nil }
    return func() tea.Msg {
        ctx := context.Background()
        var err error
        switch action {
        case "start":
            err = a.supervisorMgr.Start(ctx, name)
        case "stop":
            err = a.supervisorMgr.Stop(ctx, name)
        case "restart":
            err = a.supervisorMgr.Restart(ctx, name)
        }
        if err != nil { return QueueOpErrMsg{Err: err} }
        return QueueOpDoneMsg{}
    }
}
```

**handleDeleteWorker(name)** -- Delete worker config
```go
func (a *App) handleDeleteWorker(name string) tea.Cmd {
    if a.supervisorMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.supervisorMgr.Delete(context.Background(), name); err != nil {
            return QueueOpErrMsg{Err: err}
        }
        return QueueOpDoneMsg{}
    }
}
```

## app_handlers_backup.go

**fetchBackups()** -- Load backup list
Note: backup.Manager.List() takes a domain. BackupScreen needs a domain context. For now, list all by passing empty or iterating sites. Check Manager.List signature.
```go
func (a *App) fetchBackups(domain string) tea.Cmd {
    if a.backupMgr == nil { return nil }
    return func() tea.Msg {
        backups, err := a.backupMgr.List(domain)
        if err != nil { return BackupListErrMsg{Err: err} }
        return BackupListMsg{Backups: backups}
    }
}
```

**handleCreateBackup()** -- Create backup
Note: CreateBackupMsg has no fields. Needs domain + type selection. Placeholder.
```go
func (a *App) handleCreateBackup() tea.Cmd {
    if a.backupMgr == nil { return nil }
    return func() tea.Msg {
        return BackupOpErrMsg{Err: fmt.Errorf("backup creation requires domain and type selection (not yet implemented)")}
    }
}
```

**handleRestoreBackup(path)** -- Restore from archive
Note: Needs domain for restore. Placeholder until domain context is available.
```go
func (a *App) handleRestoreBackup(path string) tea.Cmd {
    if a.backupMgr == nil { return nil }
    return func() tea.Msg {
        return BackupOpErrMsg{Err: fmt.Errorf("restore requires domain context (not yet implemented)")}
    }
}
```

**handleDeleteBackup(path)** -- Delete backup file
```go
func (a *App) handleDeleteBackup(path string) tea.Cmd {
    if a.backupMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.backupMgr.Delete(path); err != nil {
            return BackupOpErrMsg{Err: err}
        }
        return BackupOpDoneMsg{}
    }
}
```

## Implementation Steps

- [ ] Create `internal/tui/app_handlers_queue.go` with fetchWorkers, handleWorkerAction, handleDeleteWorker
- [ ] Create `internal/tui/app_handlers_backup.go` with fetchBackups, handleCreateBackup, handleRestoreBackup, handleDeleteBackup
- [ ] Verify supervisor.Manager methods: Start/Stop/Restart take `(ctx, domain)` not `(ctx, name)`
- [ ] Verify backup.Manager.List takes `domain string`
- [ ] Verify imports compile

## Success Criteria
- Both files compile
- Worker actions consolidated into 1 method (DRY, mirrors Phase 4 service pattern)
- Backup handlers handle domain-scoped listing

## Conflict Prevention
- New files only

## Risk Assessment
- **Medium:** Supervisor methods use "domain" as identifier (e.g., `example.com`), but screen sends `Name` field. Verify WorkerStatus.Name matches what Start/Stop expect. The Name field is the program name (e.g., "example.com-worker"), supervisor methods may expect just the domain.
- **Medium:** CreateBackupMsg and RestoreBackupMsg lack domain context. Need form input. Returning informative errors for now.
- **Low:** fetchBackups needs a domain. Screen may need to track selected domain. For initial wiring, can pass "" and handle gracefully.
