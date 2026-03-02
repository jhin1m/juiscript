# Phase 05: App Integration

## Context

Wire the provisioner and setup screen into the existing TUI app router and dashboard. Add auto-detection on startup and warning banner for missing packages.

## Overview

Modify `app.go` to add ScreenSetup enum, setupScreen field, provisioner dependency, and message handlers. Modify `dashboard.go` to show warning banner and 's' key for setup navigation.

## Key Insights

- `NewApp()` already accepts `*service.Manager` — add `*provisioner.Provisioner` same way
- Auto-detect runs as tea.Cmd on Init (like fetchServiceStatus)
- Warning banner on dashboard: conditionally rendered if missing packages detected
- Channel-based progress streaming from app.go to setupScreen

## Requirements

### app.go Changes

1. Add `ScreenSetup` to Screen enum (after ScreenBackup)
2. Add to `screenNames`: `"Setup": ScreenSetup`
3. Add fields to App struct:
   - `provisioner *provisioner.Provisioner`
   - `setupScreen *screens.SetupScreen`
   - `setupProgressCh chan provisioner.ProgressEvent` (nil when not installing)
   - `missingPackages []provisioner.PackageInfo` (cached detection result)
4. Modify `NewApp()`: accept `*provisioner.Provisioner`, create setupScreen
5. Add `DetectPackagesMsg` and `DetectPackagesErrMsg` message types
6. In `Init()`: batch with `detectPackages()` cmd
7. Handle `DetectPackagesMsg`: cache results, call `setupScreen.SetPackages()`
8. Handle `screens.RunSetupMsg`:
   - Create channel, start goroutine calling `provisioner.InstallSelected`
   - Store channel in `setupProgressCh`
   - Return `waitForProgress(ch)` cmd
9. Handle `screens.SetupProgressMsg`:
   - Forward to setupScreen Update
   - Return `waitForProgress(ch)` to continue streaming
10. Handle `screens.SetupDoneMsg`:
    - Forward to setupScreen
    - Set `setupProgressCh = nil`
    - Re-run detection to refresh state
    - Re-fetch service status
11. Add setupScreen to `updateActiveScreen()` and `View()` switch
12. Add key bindings for setup screen in `currentBindings()`

### dashboard.go Changes

1. Add `missingCount int` field and `SetMissingCount(n int)` method
2. In `View()`: if missingCount > 0, render warning banner above menu
   - Style: WarnText, e.g. `"  ⚠ N package(s) not installed — press 's' for Setup"`
3. In `Update()`: handle 's' key → emit `NavigateMsg{Screen: "Setup"}`
4. Add "9" key shortcut for Setup (or 's' direct)

## Architecture

### Message Flow

```
App.Init()
    ├── dashboard.Init()
    ├── fetchServiceStatus()       (existing)
    └── detectPackages()           (NEW)
            ↓
    DetectPackagesMsg{pkgs}
            ↓
    app.missingPackages = filter(not installed)
    dashboard.SetMissingCount(len(missing))
    setupScreen.SetPackages(pkgs)
```

```
Dashboard ──['s']──> NavigateMsg{Screen: "Setup"}
    ↓
App routes to ScreenSetup
    ↓
SetupScreen: checklist → confirm → RunSetupMsg{names}
    ↓
App: start goroutine + channel
    ↓
SetupProgressMsg (repeated via waitForProgress)
    ↓
SetupDoneMsg
    ↓
App: re-detect + re-fetch services → GoBackMsg → Dashboard
```

### Channel Lifecycle

```go
// In App struct
setupProgressCh chan provisioner.ProgressEvent

// On RunSetupMsg:
ch := make(chan provisioner.ProgressEvent, 10)
a.setupProgressCh = ch
go func() {
    a.provisioner.InstallSelected(ctx, msg.Names, func(ev) { ch <- ev })
    close(ch)
}()
return a, waitForProgress(ch)

// On SetupProgressMsg:
return a, waitForProgress(a.setupProgressCh)

// On SetupDoneMsg:
a.setupProgressCh = nil
return a, tea.Batch(a.detectPackages(), a.fetchServiceStatus())
```

## Related Code Files

- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go` - main router (465 lines)
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/dashboard.go` - dashboard (119 lines)
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/setup.go` (Phase 04)

## Implementation Steps

### app.go

1. Add `ScreenSetup` enum value
2. Add "Setup" to `screenNames` map
3. Add provisioner + setupScreen + channel fields to App struct
4. Update `NewApp()` signature: `NewApp(svcMgr *service.Manager, prov *provisioner.Provisioner) *App`
5. Create `setupScreen` in constructor
6. Add `detectPackages()` method returning `tea.Cmd`
7. Add `DetectPackagesMsg` / `DetectPackagesErrMsg` types
8. Handle messages in `Update()`: DetectPackagesMsg, RunSetupMsg, SetupProgressMsg, SetupDoneMsg
9. Add ScreenSetup cases to `updateActiveScreen()`, `View()`, `screenTitle()`, `currentBindings()`
10. Update `goBack()`: ScreenSetup returns to Dashboard

### dashboard.go

1. Add `missingCount int` field
2. Add `SetMissingCount(n int)` method
3. Add warning banner rendering in `View()` (before menu items)
4. Add 's' key handler in `Update()` → NavigateMsg
5. Add menu item: `{Title: "Setup", Desc: "Install missing packages", Key: "s"}` — or keep as banner-only

### cmd/juiscript/main.go

1. Update to create provisioner and pass to NewApp
2. Create executor, php.Manager, provisioner in order
3. Pass to `NewApp(svcMgr, provisioner)`

## Todo

- [ ] Add ScreenSetup enum + screenNames entry
- [ ] Add provisioner/setupScreen/channel fields to App
- [ ] Update NewApp() signature and constructor
- [ ] Implement detectPackages() tea.Cmd
- [ ] Handle DetectPackagesMsg in Update
- [ ] Handle RunSetupMsg with channel creation
- [ ] Handle SetupProgressMsg with channel chaining
- [ ] Handle SetupDoneMsg with re-detection
- [ ] Add setup screen to router (updateActiveScreen, View, screenTitle, currentBindings)
- [ ] Add warning banner to dashboard.go
- [ ] Add 's' key handler to dashboard.go
- [ ] Update main.go to create provisioner

## Success Criteria

- Auto-detection runs on app startup without blocking TUI
- Dashboard shows warning when packages missing
- 's' key navigates to setup screen
- Install progress streams to TUI in real-time
- After install: detection refreshes, service bar updates, return to dashboard
- No regression in existing screen navigation

## Risk Assessment

- **Medium**: Channel-based progress streaming — ensure channel closed on ctx cancel/error to prevent goroutine leak
- **Medium**: `NewApp()` signature change — update all callers (only main.go)
- **Low**: Dashboard warning banner layout — test with narrow terminals

## Security

- Provisioner passed via constructor (no global state)
- Channel is private to App struct

## Next Steps

After Phase 05: update `docs/codebase-summary.md` and `docs/system-architecture.md` with provisioner docs.
