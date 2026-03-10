# Phase 3: TUI Integration

## Context
- [Plan](./plan.md) | Reference: `internal/tui/screens/firewall.go`, `internal/tui/app_handlers_firewall.go`
- **Depends on**: Phase 1 (cache.Manager), Phase 2 (main.go wiring)
- **Start after**: Phase 1 complete (needs cache.CacheStatus type)

## Overview
Add Cache screen to TUI with Redis status display, flush actions, and Opcache reset. Follow firewall screen pattern exactly.

## File Ownership
| File | Role |
|------|------|
| `internal/tui/screens/cache.go` | EXCLUSIVE |
| `internal/tui/app_handlers_cache.go` | EXCLUSIVE |
| `internal/tui/app.go` | EDIT |
| `internal/tui/app_messages.go` | EDIT |
| `internal/tui/screens/dashboard.go` | EDIT |

## Implementation Steps

### Step 1: app_messages.go - Add cache messages

Append after firewall messages block:

```go
// -- Cache result messages --

// CacheStatusMsg delivers Redis/Opcache status.
type CacheStatusMsg struct {
    Status *cache.CacheStatus
}

// CacheStatusErrMsg reports failure to fetch cache status.
type CacheStatusErrMsg struct{ Err error }

// CacheOpDoneMsg signals a cache operation succeeded.
type CacheOpDoneMsg struct{}

// CacheOpErrMsg reports a failed cache operation.
type CacheOpErrMsg struct{ Err error }
```

Add import: `"github.com/jhin1m/juiscript/internal/cache"`

### Step 2: app.go - Screen enum, deps, routing

Add to Screen enum (after ScreenFirewall):
```go
ScreenCache
```

Add to screenNames:
```go
"Cache": ScreenCache,
```

Add to AppDeps:
```go
CacheMgr *cache.Manager
```

Add to App struct:
```go
cacheMgr    *cache.Manager
cacheScreen *screens.CacheScreen
```

Add to NewApp constructor:
```go
cacheMgr:    deps.CacheMgr,
cacheScreen: screens.NewCacheScreen(t),
```

Add to NavigateMsg switch (fetch data):
```go
case ScreenCache:
    cmds = append(cmds, a.fetchCacheStatus())
```

Add fetchCacheStatus method:
```go
func (a *App) fetchCacheStatus() tea.Cmd {
    if a.cacheMgr == nil {
        return func() tea.Msg {
            return CacheStatusErrMsg{Err: fmt.Errorf("cache manager not available")}
        }
    }
    return func() tea.Msg {
        status, err := a.cacheMgr.Status(context.Background())
        if err != nil { return CacheStatusErrMsg{Err: err} }
        return CacheStatusMsg{Status: status}
    }
}
```

Add result message handlers in Update():
```go
// Cache results
case CacheStatusMsg:
    a.cacheScreen.SetStatus(msg.Status)
    return a, nil
case CacheStatusErrMsg:
    a.cacheScreen.SetError(msg.Err)
    return a, nil
case CacheOpDoneMsg:
    toastCmd := a.toast.Show(components.ToastSuccess, "Cache operation completed")
    return a, tea.Batch(toastCmd, a.fetchCacheStatus())
case CacheOpErrMsg:
    a.cacheScreen.SetError(msg.Err)
    toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
    return a, toastCmd
```

Add screen action message handlers:
```go
// Cache screen messages
case screens.FlushRedisDBMsg:
    return a, a.handleFlushRedisDB(msg.DB)
case screens.FlushRedisAllMsg:
    return a, a.handleFlushRedisAll()
case screens.ResetOpcacheMsg:
    return a, a.handleResetOpcache(msg.PHPVersion)
```

Add to updateActiveScreen:
```go
case ScreenCache:
    updated, cmd := a.cacheScreen.Update(msg)
    a.cacheScreen = updated.(*screens.CacheScreen)
    return cmd
```

Add to View switch:
```go
case ScreenCache:
    content = a.cacheScreen.View()
```

Add to currentBindings:
```go
case ScreenCache:
    return append([]components.KeyBinding{
        {Key: "f", Desc: "flush db"},
        {Key: "F", Desc: "flush all"},
        {Key: "o", Desc: "opcache reset"},
        {Key: "esc", Desc: "back"},
    }, base...)
```

### Step 3: app_handlers_cache.go

```go
package tui

import (
    "context"
    tea "github.com/charmbracelet/bubbletea"
)

func (a *App) handleFlushRedisDB(db int) tea.Cmd {
    if a.cacheMgr == nil { return nil }
    return func() tea.Msg {
        err := a.cacheMgr.FlushDB(context.Background(), db)
        if err != nil { return CacheOpErrMsg{Err: err} }
        return CacheOpDoneMsg{}
    }
}

func (a *App) handleFlushRedisAll() tea.Cmd {
    if a.cacheMgr == nil { return nil }
    return func() tea.Msg {
        err := a.cacheMgr.FlushAll(context.Background())
        if err != nil { return CacheOpErrMsg{Err: err} }
        return CacheOpDoneMsg{}
    }
}

func (a *App) handleResetOpcache(phpVersion string) tea.Cmd {
    if a.cacheMgr == nil { return nil }
    return func() tea.Msg {
        err := a.cacheMgr.ResetOpcache(context.Background(), phpVersion)
        if err != nil { return CacheOpErrMsg{Err: err} }
        return CacheOpDoneMsg{}
    }
}
```

### Step 4: screens/cache.go

Screen message types:
```go
type FlushRedisDBMsg struct{ DB int }
type FlushRedisAllMsg struct{}
type ResetOpcacheMsg struct{ PHPVersion string }
```

CacheScreen struct:
```go
type CacheScreen struct {
    theme      *theme.Theme
    status     *cache.CacheStatus
    inputMode  string // "", "flush-db", "opcache-version"
    inputBuffer string
    confirmMode   bool
    confirmPrompt string
    pendingAction string
    width, height int
    err           error
}
```

Key bindings:
- `f` - flush specific DB (prompts for DB number)
- `F` (shift+f) - flush all (confirm y/n)
- `o` - opcache reset (prompts for PHP version, defaults to config)
- `esc` - back

View: Show Redis status (running/stopped, version, memory), then action hints.

### Step 5: dashboard.go - Add Cache menu item

Insert before Firewall item (index 8):
```go
{Title: "Cache", Desc: "Redis and Opcache management", Key: "c"},
```

Update key handling - add `c` case:
```go
case "c":
    // Find Cache item index
    for i, item := range d.items {
        if item.Title == "Cache" { d.cursor = i; break }
    }
    return d, func() tea.Msg { return NavigateMsg{Screen: "Cache"} }
```

**Dashboard key layout after change**:
- 1-9, 0: existing items
- c: Cache (new letter-based key)

## Todo
- [x] Edit `internal/tui/app_messages.go` - add 4 cache message types
- [x] Edit `internal/tui/app.go` - screen enum, deps, routing, handlers, view, bindings
- [x] Create `internal/tui/app_handlers_cache.go`
- [x] Create `internal/tui/screens/cache.go`
- [x] Edit `internal/tui/screens/dashboard.go` - add Cache menu item with 'c' key

## Success Criteria
- Cache screen accessible via dashboard 'c' key
- Shows Redis status (running/version/memory or "not running")
- Flush DB, Flush All, Opcache Reset all work with confirmation
- Toast notifications on success/error
- Back navigation works (esc -> dashboard)

## Conflict Prevention
- **main.go**: NOT edited by Phase 3. Phase 2 handles all main.go wiring.
- **app.go**: Only Phase 3 edits this. No conflict.
- **dashboard.go**: Only Phase 3 edits this. Adding new item + key handler.

## Risk Assessment
- **Medium**: Dashboard menu now has 11 items. Letter key 'c' avoids number overflow. May need scroll if terminal height < 20 lines.
- **Low**: Cache screen is simpler than firewall (no tabs, fewer actions). Low complexity risk.
