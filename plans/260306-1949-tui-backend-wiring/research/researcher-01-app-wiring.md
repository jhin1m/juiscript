# TUI-Backend Wiring Analysis: app.go Structure & Gaps

**Date:** 2026-03-06 | **Analysis:** internal/tui/app.go (657 lines)

## 1. Manager Storage in App Struct (Lines 80–107)

Managers injected at construction:
- `svcMgr *service.Manager` (line 86)
- `prov *provisioner.Provisioner` (line 87)
- `phpMgr *php.Manager` (line 88)

**Pattern:** Nil-safe construction allows graceful degradation (line 111).

## 2. TODO Comment Locations & Current Handler Structure

### Wired (Working Pattern):
- **Lines 148–162:** `fetchServiceStatus()` - fetches async, returns `ServiceStatusMsg/Err`
- **Lines 165–176:** `fetchPHPVersions()` - loads versions async, returns `PHPVersionsMsg/Err`
- **Lines 282–307:** Setup provisioning flow with progress channel handling
- **Lines 360–365:** `SetDefaultPHPMsg` - updates config & persists

### TODOs Requiring Wiring (22 TODOs):
- **Sites:** ShowSiteDetail (319), CreateSite (323), ToggleSite (329), DeleteSite (333)
- **Nginx:** ToggleVhost (338), DeleteVhost (342), TestNginx (346)
- **PHP:** InstallPHP (351), RemovePHP (355)
- **Database:** CreateDB (369), DropDB (373), ImportDB (377), ExportDB (381)
- **SSL:** ObtainCert (386), RevokeCert (390), RenewCert (394)
- **Services:** StartService (399), StopService (403), RestartService (407), ReloadService (411)
- **Queues:** StartWorker (416), StopWorker (420), RestartWorker (424), DeleteWorker (429)
- **Backup:** CreateBackup (433), RestoreBackup (437), DeleteBackup (441)

## 3. Message Types by Operation Group

| Group | Messages | Line Refs |
|-------|----------|-----------|
| **Site** | ShowCreateForm, ShowSiteDetail, CreateSite, ToggleSite, DeleteSite | sites.go:145–147, sitecreate.go:243, sitedetail.go:137 |
| **Nginx** | ToggleVhost, DeleteVhost, TestNginx | nginx.go:133–142 |
| **PHP** | InstallPHP, RemovePHP, SetDefaultPHP | php.go:157–164 |
| **Database** | CreateDB, DropDB, ImportDB, ExportDB | database.go:131–141 |
| **SSL** | ObtainCert, RevokeCert, RenewCert | ssl.go:156–162 |
| **Service** | Start/Stop/Restart/ReloadService | services.go:168–180 |
| **Queues** | Start/Stop/Restart/DeleteWorker | queues.go:170–173 |
| **Backup** | CreateBackup, RestoreBackup, DeleteBackup | backup.go:127–133 |

## 4. Existing Wired Pattern (Lines 148–176)

**Template to replicate:**
```go
func (a *App) fetchX() tea.Cmd {
  if a.mgr == nil { return nil }
  return func() tea.Msg {
    result, err := a.mgr.DoThing(context.Background())
    if err != nil { return XErrMsg{Err: err} }
    return XMsg{Data: result}
  }
}
```

**In Update():**
- Async fetch + return cmd (line 271–274)
- Handle success/error messages (lines 214–252)
- Refresh dependent screens (line 307: re-fetch PHP versions)

## 5. Data Flow & Return Expectations

| Operation | Input | Handler Pattern | Return Flow |
|-----------|-------|-----------------|-------------|
| SetDefault PHP | version string | Direct (lines 358–365) | Save config, update screen |
| Service actions | service name | Missing: Execute, then fetchServiceStatus (line 400) | Refresh via ServiceStatusMsg |
| Site operations | domain/config | Missing: Execute, then what? | Likely: refresh site list |
| PHP install/remove | version | Missing: Execute async, return? | Likely: fetchPHPVersions() follow-up |
| Setup install | package names | Spawns goroutine (line 287), channels progress | SetupDoneMsg with summary |

**Key Insight:** Service operations pattern established (lines 398–413): execute then re-fetch status. Apply similarly for PHP (re-fetch versions), sites (re-fetch list).

## Critical Gaps

1. **No site/nginx/database managers** in App struct—must be added if they exist
2. **No error feedback UI**—TODOs don't show how errors surface to user
3. **Missing result messages**—Some ops need *Result/*Ok messages to confirm success
4. **Async patterns inconsistent**—Some ops missing goroutine spawning vs. sync approach clarity
5. **No confirmation dialogs**—Destructive ops (delete) may need user verification flow

## Unresolved Questions

- What managers exist in codebase? (site, nginx, database, ssl, backup, supervisor)
- Should failed operations show modal error, or just status bar?
- Do create/edit operations navigate back automatically, or wait for confirmation?
- Is progress/loading state needed for non-trivial operations (cert renewal, DB import)?
