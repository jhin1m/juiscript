---
phase: 6
title: "Wire Handlers into app.go Update()"
status: pending
depends_on: [1, 2, 3, 4, 5]
parallel: false
effort: 1.5h
---

# Phase 6: Wire Handlers into app.go Update()

## Context
Phases 2-5 created handler methods on *App in separate files. This phase replaces all 22 TODOs in app.go Update() with calls to those handlers, and adds result message handling + data-fetch on navigation.

## Parallelization
**Sequential.** Must run after all handler files exist.

## File Ownership (exclusive)
- `internal/tui/app.go` -- Update() method modifications only

## Overview

Three changes to app.go Update():

1. **Replace TODOs** -- Each `case screens.XMsg` calls `a.handleX()` and returns the tea.Cmd
2. **Handle result messages** -- New cases for success/error messages route data to screens
3. **Data-fetch on navigation** -- When navigating to a screen, fetch its data

## 1. Replace TODOs with Handler Calls

```go
// Sites
case screens.ShowSiteDetailMsg:
    a.previous = a.current
    a.current = ScreenSiteDetail
    return a, a.fetchSiteDetail(msg.Domain)

case screens.CreateSiteMsg:
    return a, a.handleCreateSite(msg.Options)

case screens.ToggleSiteMsg:
    return a, a.handleToggleSite(msg.Domain)

case screens.DeleteSiteMsg:
    return a, a.handleDeleteSite(msg.Domain)

// Nginx
case screens.ToggleVhostMsg:
    return a, a.handleToggleVhost(msg.Domain, msg.CurrentlyEnabled)

case screens.DeleteVhostMsg:
    return a, a.handleDeleteVhost(msg.Domain)

case screens.TestNginxMsg:
    return a, a.handleTestNginx()

// PHP
case screens.InstallPHPMsg:
    return a, a.handleInstallPHP(msg.Version)

case screens.RemovePHPMsg:
    return a, a.handleRemovePHP(msg.Version)

// Database
case screens.CreateDBMsg:
    return a, a.handleCreateDB()

case screens.DropDBMsg:
    return a, a.handleDropDB(msg.Name)

case screens.ImportDBMsg:
    return a, a.handleImportDB(msg.Name)

case screens.ExportDBMsg:
    return a, a.handleExportDB(msg.Name)

// SSL
case screens.ObtainCertMsg:
    return a, a.handleObtainCert()

case screens.RevokeCertMsg:
    return a, a.handleRevokeCert(msg.Domain)

case screens.RenewCertMsg:
    return a, a.handleRenewCert(msg.Domain)

// Services
case screens.StartServiceMsg:
    return a, a.handleServiceAction(msg.Name, "start")

case screens.StopServiceMsg:
    return a, a.handleServiceAction(msg.Name, "stop")

case screens.RestartServiceMsg:
    return a, a.handleServiceAction(msg.Name, "restart")

case screens.ReloadServiceMsg:
    return a, a.handleServiceAction(msg.Name, "reload")

// Queues
case screens.StartWorkerMsg:
    return a, a.handleWorkerAction(msg.Name, "start")

case screens.StopWorkerMsg:
    return a, a.handleWorkerAction(msg.Name, "stop")

case screens.RestartWorkerMsg:
    return a, a.handleWorkerAction(msg.Name, "restart")

case screens.DeleteWorkerMsg:
    return a, a.handleDeleteWorker(msg.Name)

// Backup
case screens.CreateBackupMsg:
    return a, a.handleCreateBackup()

case screens.RestoreBackupMsg:
    return a, a.handleRestoreBackup(msg.Path)

case screens.DeleteBackupMsg:
    return a, a.handleDeleteBackup(msg.Path)
```

## 2. Handle Result Messages

Add new cases in Update() for result messages from async operations:

```go
// Site results
case SiteListMsg:
    a.siteList.SetSites(msg.Sites)
    return a, nil
case SiteListErrMsg:
    a.siteList.SetError(msg.Err)
    return a, nil
case SiteCreatedMsg:
    a.current = ScreenSites
    return a, a.fetchSites()
case SiteOpDoneMsg:
    return a, a.fetchSites() // refresh after toggle/delete
case SiteOpErrMsg:
    a.siteList.SetError(msg.Err)
    return a, nil

// Nginx results
case VhostListMsg:
    a.nginxScreen.SetVhosts(msg.Vhosts)
    return a, nil
case VhostListErrMsg:
    a.nginxScreen.SetError(msg.Err)
    return a, nil
case NginxOpDoneMsg:
    return a, a.fetchVhosts()
case NginxOpErrMsg:
    a.nginxScreen.SetError(msg.Err)
    return a, nil
case NginxTestOkMsg:
    // Could show success status; for now just refresh
    return a, a.fetchVhosts()

// Database results
case DBListMsg:
    a.databaseScreen.SetDatabases(msg.Databases)
    return a, nil
case DBListErrMsg:
    a.databaseScreen.SetError(msg.Err)
    return a, nil
case DBOpDoneMsg:
    return a, a.fetchDatabases()
case DBOpErrMsg:
    a.databaseScreen.SetError(msg.Err)
    return a, nil

// SSL results
case CertListMsg:
    a.sslScreen.SetCerts(msg.Certs)
    return a, nil
case CertListErrMsg:
    a.sslScreen.SetError(msg.Err)
    return a, nil
case SSLOpDoneMsg:
    return a, a.fetchCerts()
case SSLOpErrMsg:
    a.sslScreen.SetError(msg.Err)
    return a, nil

// Service results (ServiceStatusMsg already handled)
case ServiceOpErrMsg:
    a.servicesScreen.SetError(msg.Err)
    return a, nil

// Queue results
case WorkerListMsg:
    a.queuesScreen.SetWorkers(msg.Workers)
    return a, nil
case WorkerListErrMsg:
    a.queuesScreen.SetError(msg.Err)
    return a, nil
case QueueOpDoneMsg:
    return a, a.fetchWorkers()
case QueueOpErrMsg:
    a.queuesScreen.SetError(msg.Err)
    return a, nil

// Backup results
case BackupListMsg:
    a.backupScreen.SetBackups(msg.Backups)
    return a, nil
case BackupListErrMsg:
    a.backupScreen.SetError(msg.Err)
    return a, nil
case BackupOpDoneMsg:
    return a, a.fetchBackups("") // refresh
case BackupOpErrMsg:
    a.backupScreen.SetError(msg.Err)
    return a, nil
```

## 3. Data-Fetch on Navigation

Update the `NavigateMsg` handler to fetch data per screen:

```go
case screens.NavigateMsg:
    if screen, ok := screenNames[msg.Screen]; ok {
        a.previous = a.current
        a.current = screen
    }
    cmds := []tea.Cmd{a.fetchServiceStatus()}
    switch a.current {
    case ScreenPHP:
        cmds = append(cmds, a.fetchPHPVersions())
    case ScreenSites:
        cmds = append(cmds, a.fetchSites())
    case ScreenNginx:
        cmds = append(cmds, a.fetchVhosts())
    case ScreenDatabase:
        cmds = append(cmds, a.fetchDatabases())
    case ScreenSSL:
        cmds = append(cmds, a.fetchCerts())
    case ScreenQueues:
        cmds = append(cmds, a.fetchWorkers())
    case ScreenBackup:
        cmds = append(cmds, a.fetchBackups(""))
    case ScreenServices:
        // Already fetching service status above
    }
    return a, tea.Batch(cmds...)
```

## Implementation Steps

- [ ] Replace all 22 TODO cases with handler calls
- [ ] Add result message handling cases (~30 new cases)
- [ ] Update NavigateMsg to fetch screen-specific data
- [ ] Move existing `ServiceStatusMsg` handling to coexist with new `ServiceOpErrMsg`
- [ ] Update imports in app.go if needed (new message types from app_messages.go are same package)
- [ ] Run `make build` -- verify compilation
- [ ] Run `make test` -- verify no regressions
- [ ] Manual smoke test: navigate to each screen, verify no panics

## Success Criteria
- Zero TODO comments remain in app.go Update()
- Every screen fetches data on navigation
- Every async operation surfaces errors via SetError
- Every successful mutation triggers a refresh
- `make build` and `make test` pass

## Conflict Prevention
- Only this phase modifies app.go Update() after Phase 1
- Handler methods (phases 2-5) are called but not modified here
- app_messages.go types are in same package, no import needed

## Risk Assessment
- **Medium:** Large Update() switch statement (~100 cases). Manageable since Go compiles fast and cases are independent.
- **Low:** Screen SetError/SetData methods might not all exist with exact signatures. Verify each screen's method names before wiring.
- **Note:** `ServiceOpDoneMsg` not needed separately since handleServiceAction returns ServiceStatusMsg directly (auto-refreshes). Only ServiceOpErrMsg needed.
- **Verify:** ServicesScreen has SetError method. Check services.go screen API.

## Unresolved Questions

1. **Backup domain context:** fetchBackups("") may return nothing if List requires non-empty domain. Need to check backup.Manager.List behavior with empty string.
2. **Screen loading indicators:** No spinner/loading state while async ops run. Users see stale data until response arrives. Acceptable for v1; future enhancement.
3. **Confirmation dialogs:** Destructive operations (delete site, drop DB, revoke cert) have no confirmation. YAGNI for now -- can add modal component later.
4. **Form screens:** CreateDB, ObtainCert, CreateBackup need input forms. These are separate features beyond wiring scope. Current plan returns informative errors.
