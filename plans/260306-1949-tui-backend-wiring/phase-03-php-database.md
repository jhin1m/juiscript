---
phase: 3
title: "PHP & Database Handler Methods"
status: pending
depends_on: [1]
parallel: true
parallel_with: [2, 4, 5]
effort: 1h
---

# Phase 3: PHP & Database Handlers

## Context
PHP operations (2 TODOs: install, remove) and Database operations (4 TODOs: create, drop, import, export) need async handlers.

## Parallelization
Runs in parallel with Phases 2, 4, 5.

## File Ownership (exclusive)
- `internal/tui/app_handlers_php.go` -- NEW
- `internal/tui/app_handlers_db.go` -- NEW

## app_handlers_php.go

**handleInstallPHP(version)** -- Long-running, spawns goroutine
```go
func (a *App) handleInstallPHP(version string) tea.Cmd {
    if a.phpMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.phpMgr.InstallVersion(context.Background(), version); err != nil {
            return PHPVersionsErrMsg{Err: err}
        }
        // Re-fetch versions after install completes
        versions, err := a.phpMgr.ListVersions(context.Background())
        if err != nil {
            return PHPVersionsErrMsg{Err: err}
        }
        return PHPVersionsMsg{Versions: versions}
    }
}
```

**handleRemovePHP(version)** -- Remove PHP version
```go
func (a *App) handleRemovePHP(version string) tea.Cmd {
    if a.phpMgr == nil { return nil }
    return func() tea.Msg {
        // Pass empty sites slice -- TUI doesn't track site-version mapping yet
        if err := a.phpMgr.RemoveVersion(context.Background(), version, nil); err != nil {
            return PHPVersionsErrMsg{Err: err}
        }
        versions, err := a.phpMgr.ListVersions(context.Background())
        if err != nil {
            return PHPVersionsErrMsg{Err: err}
        }
        return PHPVersionsMsg{Versions: versions}
    }
}
```

Note: Existing `fetchPHPVersions()` already exists in app.go. The install/remove handlers return PHPVersionsMsg directly to refresh the screen after operation completes.

## app_handlers_db.go

**fetchDatabases()** -- Load DB list
```go
func (a *App) fetchDatabases() tea.Cmd {
    if a.dbMgr == nil { return nil }
    return func() tea.Msg {
        dbs, err := a.dbMgr.ListDBs(context.Background())
        if err != nil { return DBListErrMsg{Err: err} }
        return DBListMsg{Databases: dbs}
    }
}
```

**handleCreateDB()** -- Create database
Note: CreateDBMsg has no fields. Need a name input. For now, this is a placeholder until a form/dialog is added. Return error explaining input needed.
```go
func (a *App) handleCreateDB() tea.Cmd {
    if a.dbMgr == nil { return nil }
    return func() tea.Msg {
        // TODO: Need form input for DB name. For now, surface as error.
        return DBOpErrMsg{Err: fmt.Errorf("database creation requires a name input form (not yet implemented)")}
    }
}
```

**handleDropDB(name)** -- Drop database
```go
func (a *App) handleDropDB(name string) tea.Cmd {
    if a.dbMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.dbMgr.DropDB(context.Background(), name); err != nil {
            return DBOpErrMsg{Err: err}
        }
        return DBOpDoneMsg{}
    }
}
```

**handleImportDB(name)** -- Import SQL into database
Note: ImportDBMsg only has Name. Needs file path input. Placeholder for now.
```go
func (a *App) handleImportDB(name string) tea.Cmd {
    if a.dbMgr == nil { return nil }
    return func() tea.Msg {
        return DBOpErrMsg{Err: fmt.Errorf("import requires a file path input form (not yet implemented)")}
    }
}
```

**handleExportDB(name)** -- Export database
```go
func (a *App) handleExportDB(name string) tea.Cmd {
    if a.dbMgr == nil { return nil }
    return func() tea.Msg {
        // Export to backup dir with standard naming
        path := fmt.Sprintf("/var/backups/juiscript/%s.sql.gz", name)
        if err := a.dbMgr.Export(context.Background(), name, path); err != nil {
            return DBOpErrMsg{Err: err}
        }
        return DBOpDoneMsg{}
    }
}
```

## Implementation Steps

- [ ] Create `internal/tui/app_handlers_php.go` with handleInstallPHP, handleRemovePHP
- [ ] Create `internal/tui/app_handlers_db.go` with fetchDatabases, handleCreateDB, handleDropDB, handleImportDB, handleExportDB
- [ ] Verify imports compile: `go build ./internal/tui/...`

## Success Criteria
- Both files compile
- PHP handlers return PHPVersionsMsg for auto-refresh
- DB handlers use correct manager method signatures
- Nil-safe checks on all methods

## Conflict Prevention
- New files only, no overlap with phases 2/4/5
- Reuses existing PHPVersionsMsg/PHPVersionsErrMsg from app.go (read-only)

## Risk Assessment
- **Medium:** CreateDBMsg and ImportDBMsg lack user input fields. Handlers return errors for now. Future phases need input dialogs (forms).
- **Low:** PHP install is long-running (5min timeout). Bubble Tea handles this fine since handler runs in goroutine via tea.Cmd.
- **Note:** `RemoveVersion` requires sites list to check usage. Passing nil skips the check. Safe enough for now but could cause issues if PHP version is in use.
