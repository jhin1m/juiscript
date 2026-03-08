# Phase 4: Integration Wiring

## Context

- Parent: [plan.md](plan.md)
- Dependencies: Phase 1, 2, 3
- Pattern: `cmd/juiscript/main.go`, `internal/tui/app.go`, `internal/tui/app_messages.go`, `internal/tui/app_handlers_service.go`

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-08 |
| Priority | P1 |
| Effort | 1h |
| Status | done |

Wire firewall manager into main.go (Managers struct, initManagers), app.go (AppDeps, screen enum, routing, View), app_messages.go (result messages), app_handlers_firewall.go (handler methods), and dashboard menu.

## Key Insights

- Every new feature touches exactly 6 files: main.go, app.go, app_messages.go, app_handlers_*.go, dashboard.go, and the new screen
- Pattern is mechanical: add field to structs, add cases to switches, add message types
- Screen enum order matters for navigation; add after ScreenBackup
- Dashboard menu item needs a new key binding

## Requirements

1. Add `Firewall *firewall.Manager` to Managers struct in main.go
2. Create firewall.Manager in initManagers()
3. Register `firewallCmd(mgrs)` with rootCmd
4. Add `FirewallMgr *firewall.Manager` to AppDeps
5. Add ScreenFirewall to screen enum
6. Add "Firewall" to screenNames map
7. Wire firewall screen in NewApp, updateActiveScreen, View, screenTitle, currentBindings
8. Create firewall message types in app_messages.go
9. Create app_handlers_firewall.go
10. Add fetch commands and navigation data loading
11. Add "Firewall" to dashboard menu items

## Architecture

Files to modify:
```
cmd/juiscript/main.go              # +Firewall field, +initManagers, +rootCmd.AddCommand, +runTUI
internal/tui/app.go                # +AppDeps field, +Screen enum, +screenNames, +NewApp, +Update, +View, etc.
internal/tui/app_messages.go       # +FirewallStatusMsg, +FirewallStatusErrMsg, +FirewallOpDoneMsg, +FirewallOpErrMsg
internal/tui/screens/dashboard.go  # +MenuItem
```

New file:
```
internal/tui/app_handlers_firewall.go  # Handler methods
```

## Related Code Files

- `cmd/juiscript/main.go` - Managers struct, initManagers, rootCmd, runTUI
- `internal/tui/app.go` - AppDeps, App struct, Screen enum, Update, View, all switches
- `internal/tui/app_messages.go` - Result message types
- `internal/tui/app_handlers_service.go` - Handler pattern
- `internal/tui/screens/dashboard.go` - Menu items

## Implementation Steps

### Step 1: main.go changes

```go
// Add to Managers struct:
Firewall *firewall.Manager

// Add to initManagers():
firewallMgr := firewall.NewManager(exec)

// Add to return:
Firewall: firewallMgr,

// Add to rootCmd commands:
rootCmd.AddCommand(firewallCmd(mgrs))

// Add to runTUI AppDeps:
FirewallMgr: mgrs.Firewall,
```

### Step 2: app.go - AppDeps and App struct

```go
// Add to AppDeps:
FirewallMgr *firewall.Manager

// Add to Screen enum (after ScreenBackup):
ScreenFirewall

// Add to screenNames map:
"Firewall": ScreenFirewall,

// Add to App struct:
firewallMgr    *firewall.Manager
firewallScreen *screens.FirewallScreen

// Add to NewApp constructor:
firewallMgr:    deps.FirewallMgr,
firewallScreen: screens.NewFirewallScreen(t),
```

### Step 3: app.go - Update message handling

Add to the `Update` switch in app.go:

```go
// Navigation: fetch firewall data when entering screen
case ScreenFirewall:
    cmds = append(cmds, a.fetchFirewallStatus())

// Firewall screen action messages
case screens.OpenPortMsg:
    return a, a.handleOpenPort(msg.Port, msg.Protocol)

case screens.ClosePortMsg:
    return a, a.handleClosePort(msg.Port, msg.Protocol)

case screens.DeleteUFWRuleMsg:
    return a, a.handleDeleteUFWRule(msg.RuleNum)

case screens.BanIPMsg:
    return a, a.handleBanIP(msg.IP, msg.Jail)

case screens.UnbanIPMsg:
    return a, a.handleUnbanIP(msg.IP, msg.Jail)

// Firewall result messages
case FirewallStatusMsg:
    a.firewallScreen.SetUFWStatus(msg.UFW)
    a.firewallScreen.SetJails(msg.Jails)
    return a, nil

case FirewallStatusErrMsg:
    a.firewallScreen.SetError(msg.Err)
    return a, nil

case FirewallOpDoneMsg:
    toastCmd := a.toast.Show(components.ToastSuccess, "Firewall operation completed")
    return a, tea.Batch(toastCmd, a.fetchFirewallStatus())

case FirewallOpErrMsg:
    a.firewallScreen.SetError(msg.Err)
    toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
    return a, toastCmd
```

### Step 4: app.go - updateActiveScreen, View, screenTitle, currentBindings

```go
// updateActiveScreen:
case ScreenFirewall:
    updated, cmd := a.firewallScreen.Update(msg)
    a.firewallScreen = updated.(*screens.FirewallScreen)
    return cmd

// View:
case ScreenFirewall:
    content = a.firewallScreen.View()

// currentBindings:
case ScreenFirewall:
    return append([]components.KeyBinding{
        {Key: "j/k", Desc: "navigate"},
        {Key: "tab", Desc: "switch tab"},
        {Key: "o/c/d", Desc: "open/close/delete"},
        {Key: "b/u", Desc: "block/unblock"},
        {Key: "esc", Desc: "back"},
    }, base...)
```

### Step 5: app.go - fetch function

```go
// fetchFirewallStatus returns a tea.Cmd that fetches UFW + Fail2ban status.
func (a *App) fetchFirewallStatus() tea.Cmd {
    if a.firewallMgr == nil {
        return func() tea.Msg {
            return FirewallStatusErrMsg{Err: fmt.Errorf("firewall manager not available")}
        }
    }
    return func() tea.Msg {
        ctx := context.Background()
        ufwStatus, ufwErr := a.firewallMgr.Status(ctx)
        jails, _ := a.firewallMgr.F2bStatus(ctx) // non-fatal if fail2ban missing
        if ufwErr != nil {
            return FirewallStatusErrMsg{Err: ufwErr}
        }
        return FirewallStatusMsg{UFW: ufwStatus, Jails: jails}
    }
}
```

### Step 6: app_messages.go

```go
// -- Firewall result messages --

// FirewallStatusMsg delivers UFW + Fail2ban status.
type FirewallStatusMsg struct {
    UFW   *firewall.UFWStatus
    Jails []firewall.F2bJailStatus
}

// FirewallStatusErrMsg reports failure to fetch firewall status.
type FirewallStatusErrMsg struct{ Err error }

// FirewallOpDoneMsg signals a firewall operation succeeded.
type FirewallOpDoneMsg struct{}

// FirewallOpErrMsg reports a failed firewall operation.
type FirewallOpErrMsg struct{ Err error }
```

### Step 7: app_handlers_firewall.go

```go
package tui

import (
    "context"

    tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleOpenPort(port int, proto string) tea.Cmd {
    if a.firewallMgr == nil {
        return nil
    }
    return func() tea.Msg {
        err := a.firewallMgr.AllowPort(context.Background(), port, proto)
        if err != nil {
            return FirewallOpErrMsg{Err: err}
        }
        return FirewallOpDoneMsg{}
    }
}

func (a *App) handleClosePort(port int, proto string) tea.Cmd {
    if a.firewallMgr == nil {
        return nil
    }
    return func() tea.Msg {
        err := a.firewallMgr.DenyPort(context.Background(), port, proto)
        if err != nil {
            return FirewallOpErrMsg{Err: err}
        }
        return FirewallOpDoneMsg{}
    }
}

func (a *App) handleDeleteUFWRule(ruleNum int) tea.Cmd {
    if a.firewallMgr == nil {
        return nil
    }
    return func() tea.Msg {
        err := a.firewallMgr.DeleteRule(context.Background(), ruleNum)
        if err != nil {
            return FirewallOpErrMsg{Err: err}
        }
        return FirewallOpDoneMsg{}
    }
}

func (a *App) handleBanIP(ip, jail string) tea.Cmd {
    if a.firewallMgr == nil {
        return nil
    }
    return func() tea.Msg {
        err := a.firewallMgr.BanIP(context.Background(), ip, jail)
        if err != nil {
            return FirewallOpErrMsg{Err: err}
        }
        return FirewallOpDoneMsg{}
    }
}

func (a *App) handleUnbanIP(ip, jail string) tea.Cmd {
    if a.firewallMgr == nil {
        return nil
    }
    return func() tea.Msg {
        err := a.firewallMgr.UnbanIP(context.Background(), ip, jail)
        if err != nil {
            return FirewallOpErrMsg{Err: err}
        }
        return FirewallOpDoneMsg{}
    }
}
```

### Step 8: Dashboard menu item

In `internal/tui/screens/dashboard.go`, add to `items` slice:

```go
{Title: "Firewall", Desc: "UFW rules and IP blocking", Key: "0"},
```

Note: Keys 1-9 are taken. Options: use "0", or shift Setup to "0" and use "9" for Firewall. Recommended: insert before Setup and shift Setup key.

Updated menu:
```go
items: []MenuItem{
    {Title: "Sites", Desc: "Manage websites and domains", Key: "1"},
    {Title: "Nginx", Desc: "Virtual host configuration", Key: "2"},
    {Title: "PHP", Desc: "PHP versions and FPM pools", Key: "3"},
    {Title: "Database", Desc: "MariaDB databases and users", Key: "4"},
    {Title: "SSL", Desc: "Let's Encrypt certificates", Key: "5"},
    {Title: "Services", Desc: "Start/stop/restart services", Key: "6"},
    {Title: "Queues", Desc: "Supervisor queue workers", Key: "7"},
    {Title: "Backup", Desc: "Backup and restore sites", Key: "8"},
    {Title: "Firewall", Desc: "UFW rules and IP blocking", Key: "9"},
    {Title: "Setup", Desc: "Install missing packages", Key: "0"},
},
```

## Todo

- [ ] Add `Firewall` field to Managers struct in main.go
- [ ] Create firewall.Manager in initManagers()
- [ ] Register firewallCmd with rootCmd
- [ ] Add FirewallMgr to AppDeps and runTUI
- [ ] Add ScreenFirewall to Screen enum
- [ ] Add "Firewall" to screenNames map
- [ ] Add firewallMgr and firewallScreen fields to App struct
- [ ] Initialize in NewApp constructor
- [ ] Add fetchFirewallStatus() method
- [ ] Add all firewall message cases to Update()
- [ ] Add ScreenFirewall to updateActiveScreen, View, screenTitle, currentBindings
- [ ] Create FirewallStatusMsg, FirewallStatusErrMsg, FirewallOpDoneMsg, FirewallOpErrMsg in app_messages.go
- [ ] Create app_handlers_firewall.go with 5 handler methods
- [ ] Add "Firewall" menu item to dashboard (shift Setup to key "0")
- [ ] Verify import of `firewall` package in all files

## Success Criteria

- `juiscript` TUI shows Firewall in dashboard menu
- Navigating to Firewall screen loads UFW + Fail2ban data
- All screen actions (open/close/delete/ban/unban) trigger handlers
- Toast notifications for success/error
- Data refreshes after each operation
- Graceful nil-safety for firewallMgr

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dashboard key collision | Navigation broken | Shift Setup to "0" |
| Missing import | Build error | Verify all imports after wiring |
| Screen enum order | Wrong screen displays | Add after ScreenBackup, before ScreenSetup |

## Security Considerations

- Nil checks on firewallMgr in all handlers (graceful degradation)
- All validation done in backend manager, not in wiring layer

## Next Steps

After completing all 4 phases: update docs (codebase-summary.md, system-architecture.md, project-overview-pdr.md)
