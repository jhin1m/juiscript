---
phase: 2
title: "App Integration"
status: pending
effort: 1.5h
---

# Phase 02: App Integration

## Context

- Parent plan: [plan.md](plan.md)
- Depends on: [Phase 01](phase-01-service-status-bar-component.md) (ServiceStatusBar component)
- Key files: `internal/tui/app.go`, `cmd/juiscript/main.go`

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-02 |
| Description | Wire ServiceStatusBar into App model; inject service.Manager; handle async fetch |
| Priority | P2 |
| Impl Status | pending |
| Review Status | pending |

## Key Insights

- `NewApp()` currently takes no args — needs `*service.Manager` param (or nil for graceful degradation)
- `App.View()` renders: `header + "\n\n" + content + "\n\n" + statusBar` — insert serviceBar between header and content
- Navigation triggers via `screens.NavigateMsg` and `screens.GoBackMsg` — both should trigger re-fetch
- Service action msgs (Start/Stop/Restart/Reload) already handled in Update() — add re-fetch after each
- `service.Manager.ListAll()` is synchronous with systemctl calls — must wrap in `tea.Cmd` for async

## Requirements

1. Add `serviceBar *components.ServiceStatusBar` and `svcMgr *service.Manager` fields to `App`
2. Define `ServiceStatusMsg` and `ServiceStatusErrMsg` message types
3. Create `fetchServiceStatus()` that returns `tea.Cmd` calling `svcMgr.ListAll()`
4. `Init()` triggers initial status fetch
5. `Update()` handles `ServiceStatusMsg` → update bar, `ServiceStatusErrMsg` → show warning
6. `NavigateMsg` and `GoBackMsg` trigger re-fetch
7. Service action msgs (Start/Stop/Restart/Reload) trigger re-fetch
8. `View()` layout: `header + "\n" + serviceBar + "\n\n" + content + "\n\n" + statusBar`
9. `cmd/juiscript/main.go`: create `service.Manager`, pass to `NewApp()`
10. If `svcMgr` is nil, serviceBar shows error state gracefully

## Architecture

### New Message Types (in `app.go`)

```go
// ServiceStatusMsg delivers fresh service statuses to the App.
type ServiceStatusMsg struct {
    Services []service.Status
}

// ServiceStatusErrMsg reports a failure to read service status.
type ServiceStatusErrMsg struct {
    Err error
}
```

### fetchServiceStatus() Command

```go
func (a *App) fetchServiceStatus() tea.Cmd {
    if a.svcMgr == nil {
        return func() tea.Msg {
            return ServiceStatusErrMsg{Err: fmt.Errorf("service manager not available")}
        }
    }
    return func() tea.Msg {
        ctx := context.Background()
        statuses, err := a.svcMgr.ListAll(ctx)
        if err != nil {
            return ServiceStatusErrMsg{Err: err}
        }
        return ServiceStatusMsg{Services: statuses}
    }
}
```

### Modified NewApp Signature

```go
func NewApp(svcMgr *service.Manager) *App {
    t := theme.New()
    return &App{
        // ... existing fields ...
        serviceBar: components.NewServiceStatusBar(t),
        svcMgr:     svcMgr,
    }
}
```

### Modified View() Layout

```go
func (a *App) View() string {
    header := a.header.View(a.screenTitle())
    svcBar := a.serviceBar.View()
    // ... content switch ...
    bindings := a.currentBindings()
    statusBar := a.statusBar.View(bindings)
    return fmt.Sprintf("%s\n%s\n\n%s\n\n%s", header, svcBar, content, statusBar)
}
```

## Related Code Files

| File | Action | Details |
|------|--------|---------|
| `internal/tui/app.go` | MODIFY | Add fields, messages, fetch cmd, Update cases, View layout |
| `cmd/juiscript/main.go` | MODIFY | Create service.Manager with system.NewExecutor, pass to NewApp |
| `internal/service/manager.go` | READ | NewManager(exec), ListAll(ctx) API |
| `internal/system/executor.go` | READ | NewExecutor(logger) for main.go |

## Implementation Steps

### Step 1: Add imports to app.go
- `"context"`, `"github.com/jhin1m/juiscript/internal/service"`

### Step 2: Define message types
- `ServiceStatusMsg{Services []service.Status}`
- `ServiceStatusErrMsg{Err error}`

### Step 3: Add fields to App struct
```go
serviceBar *components.ServiceStatusBar
svcMgr     *service.Manager
```

### Step 4: Modify NewApp signature
- Accept `svcMgr *service.Manager` parameter
- Initialize `serviceBar: components.NewServiceStatusBar(t)`
- Store `svcMgr`

### Step 5: Implement fetchServiceStatus()
- Returns `tea.Cmd` that calls `svcMgr.ListAll(ctx)`
- Returns `ServiceStatusMsg` on success, `ServiceStatusErrMsg` on error
- Handles nil `svcMgr` gracefully

### Step 6: Modify Init()
```go
func (a *App) Init() tea.Cmd {
    return tea.Batch(a.dashboard.Init(), a.fetchServiceStatus())
}
```

### Step 7: Add Update() cases
- `ServiceStatusMsg` → `a.serviceBar.SetServices(msg.Services)`, clear error
- `ServiceStatusErrMsg` → `a.serviceBar.SetError(msg.Err.Error())`
- After `NavigateMsg` → return `a.fetchServiceStatus()`
- After `GoBackMsg` → return `a.fetchServiceStatus()`
- After service action msgs (Start/Stop/Restart/Reload) → batch existing logic + `a.fetchServiceStatus()`

### Step 8: Modify WindowSizeMsg handler
- Add `a.serviceBar.SetWidth(msg.Width)`

### Step 9: Modify View() layout
- Insert `a.serviceBar.View()` between header and content
- Adjust spacing: `header + "\n" + svcBar + "\n\n" + content + "\n\n" + statusBar`

### Step 10: Modify cmd/juiscript/main.go
```go
import (
    "log/slog"
    "github.com/jhin1m/juiscript/internal/service"
    "github.com/jhin1m/juiscript/internal/system"
)

func runTUI(cmd *cobra.Command, args []string) error {
    logger := slog.Default()
    exec := system.NewExecutor(logger)
    svcMgr := service.NewManager(exec)
    app := tui.NewApp(svcMgr)
    // ... rest unchanged ...
}
```

## Todo

- [ ] Add message types to app.go
- [ ] Add serviceBar + svcMgr fields to App
- [ ] Change NewApp signature to accept *service.Manager
- [ ] Implement fetchServiceStatus() tea.Cmd
- [ ] Modify Init() to batch dashboard init + fetch
- [ ] Handle ServiceStatusMsg in Update()
- [ ] Handle ServiceStatusErrMsg in Update()
- [ ] Re-fetch on NavigateMsg
- [ ] Re-fetch on GoBackMsg
- [ ] Re-fetch after service action msgs
- [ ] Add serviceBar.SetWidth in WindowSizeMsg
- [ ] Insert serviceBar.View() in View() layout
- [ ] Modify main.go to create service.Manager and pass to NewApp
- [ ] Verify compile

## Success Criteria

- ServiceStatusBar renders on every screen below header
- Status fetched async on Init() — TUI renders immediately
- Navigation triggers status refresh
- Service actions (start/stop/restart/reload) trigger refresh
- Nil service manager shows warning, doesn't crash
- Existing tests still pass (NewApp signature change requires updating callers)

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| NewApp signature change breaks tests | Medium | Update any test files calling NewApp() to pass nil |
| systemctl calls slow on first render | Low | Async cmd — bar shows empty then populates |
| Race between navigation and fetch | Low | Each fetch replaces state atomically via SetServices |

## Security Considerations

- service.Manager uses whitelist — no arbitrary systemctl calls
- No user input flows into service queries
- Executor logs all commands for audit

## Next Steps

After this phase, proceed to [Phase 03](phase-03-testing.md) for unit tests.
