# Code Review: Phase 6 — Wire Handlers into app.go Update()

## Scope
- Files reviewed: `internal/tui/app.go`, `app_handlers_site.go`, `app_handlers_nginx.go`, `app_handlers_db.go`, `app_handlers_ssl.go`, `app_handlers_service.go`, `app_handlers_queue.go`, `app_handlers_backup.go`, `app_handlers_php.go`, `app_messages.go`
- Lines analyzed: ~900 across all files
- Review focus: Update() handler wiring, message routing, error propagation

## Overall Assessment

Wiring is complete and correct. Build passes (`go build ./...` clean). All 22 TODOs replaced. Message field access matches struct definitions throughout. No remaining TODO comments in any app.go or handler file.

## Critical Issues

None.

## High Priority Findings

### H1 — `fetchSiteDetail` error goes to wrong screen

`app_handlers_site.go:30` returns `SiteOpErrMsg` on Get failure. In `app.go:481`, `SiteOpErrMsg` routes to `a.siteList.SetError(msg.Err)`. But when `fetchSiteDetail` fails, the user is already on `ScreenSiteDetail`, and `SiteDetail` has **no** `SetError` method — the error silently routes to the hidden siteList screen.

```go
// app_handlers_site.go:29-31
s, err := a.siteMgr.Get(domain)
if err != nil {
    return SiteOpErrMsg{Err: err}  // routes to siteList.SetError — wrong screen
}
```

Fix: add a `SiteDetailErrMsg` type and route it to `siteDetail.SetError` (after adding that method), or add `SetError` to `SiteDetail` and handle `SiteOpErrMsg` conditionally by current screen.

### H2 — `ScreenServices` gets no data fetch on navigation

`NavigateMsg` switch (app.go:304–319) triggers data fetches for Sites, Nginx, PHP, DB, SSL, Queues, Backup — but **not** `ScreenServices`. The services screen shows stale/empty data until the background `fetchServiceStatus` (which runs on every navigation via `cmds = []tea.Cmd{a.fetchServiceStatus()}`) completes. This works only if `ServicesScreen` consumes `ServiceStatusMsg` — it does not; `ServiceStatusMsg` goes to `serviceBar.SetServices`, not `servicesScreen`.

```go
case ServiceStatusMsg:
    a.serviceBar.SetServices(msg.Services)  // ServicesScreen never sees this
    return a, nil
```

The `ServicesScreen` has no dedicated data-load path. If it has its own service list display, it will always be empty on navigation.

Fix: add a `case ScreenServices: cmds = append(cmds, a.fetchServiceStatus())` that produces a message type `ServicesScreen` can consume, or wire `ServiceStatusMsg` to also call `a.servicesScreen.SetServices(msg.Services)`.

## Medium Priority Improvements

### M1 — `handleInstallPHP` exists but is unreachable dead code

`app_handlers_php.go` defines `handleInstallPHP(version string)` but `app.go` handles `InstallPHPMsg` inline with a stub error — never calls `handleInstallPHP`. Dead code that could mislead future contributors.

```go
// app.go:388 — inline stub, ignores handleInstallPHP
case screens.InstallPHPMsg:
    return a, func() tea.Msg {
        return PHPVersionsErrMsg{Err: fmt.Errorf("PHP install requires a version selector...")}
    }
```

Fix: either delete `handleInstallPHP` until the version picker is implemented, or add a comment linking the stub to the dead handler.

### M2 — Hardcoded backup export path

`app_handlers_db.go:63` hardcodes `/var/backups/juiscript/%s.sql.gz`. No config, no user input. Will fail silently on systems where this path doesn't exist or isn't writable.

```go
path := fmt.Sprintf("/var/backups/juiscript/%s.sql.gz", name)
```

Fix: derive from `a.cfg` or ensure directory is created before export.

### M3 — `SetupProgressMsg` / `SetupDoneMsg` bypass `updateActiveScreen`

app.go:343–352 calls `a.setupScreen.Update(msg)` directly and discards the returned `tea.Cmd`. If the setup screen produces commands from those messages, they are lost.

```go
a.setupScreen.Update(msg)  // returned cmd discarded
return a, waitForProgress(a.setupProgressCh)
```

Fix: capture and batch the returned cmd:
```go
_, cmd := a.setupScreen.Update(msg)
return a, tea.Batch(cmd, waitForProgress(a.setupProgressCh))
```

## Low Priority Suggestions

- `goBack()` (app.go:561) uses `ScreenSites` as the fallback for `ScreenSiteDetail`, but ignores `a.previous`. If detail is reached from a screen other than Sites in the future, back-navigation will be wrong. Already mitigated by `a.previous` being set — consider using it.
- `NginxTestOkMsg` (app.go:497) triggers `fetchVhosts()` but `Output` field is never surfaced to the screen. Consider routing the success message to `nginxScreen`.

## Positive Observations

- DRY: `handleServiceAction` and `handleWorkerAction` consolidate 4 action variants each into a single dispatcher — well done.
- Nil-guard pattern is consistent across all handlers — graceful degradation when managers are absent.
- All placeholder handlers (ObtainCert, CreateDB, ImportDB, CreateBackup, RestoreBackup) return proper error messages instead of silent no-ops.
- Message field access is correct throughout: `msg.Domain`, `msg.Name`, `msg.Path`, `msg.Version`, `msg.CurrentlyEnabled` all match their struct definitions.
- Build compiles cleanly with zero errors or warnings.

## Recommended Actions

1. **[High]** Add `SiteDetailErrMsg` type + `SiteDetail.SetError` method; update `fetchSiteDetail` to use it.
2. **[High]** Wire `ServiceStatusMsg` to also call `a.servicesScreen.SetServices(...)`, or add a dedicated `fetchServicesForScreen()` cmd in the navigation switch.
3. **[Medium]** Delete dead `handleInstallPHP` function or document it as future use.
4. **[Medium]** Fix discarded cmd in `SetupProgressMsg`/`SetupDoneMsg` handlers.
5. **[Medium]** Make DB export path configurable via `a.cfg`.

## Metrics

- Remaining TODOs in app.go: **0**
- Remaining TODOs in handler files: **0**
- Build errors: **0**
- Critical issues: **0**
- High issues: **2**
- Medium issues: **3**

---

_Unresolved questions_:
- Does `ServicesScreen` render its own service list (separate from `serviceBar`)? If not, H2 is moot.
- Is `a.previous` intentionally unused in `goBack()` for sub-screens beyond site detail?
