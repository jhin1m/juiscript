---
phase: 2
title: "Site & Nginx Handler Methods"
status: pending
depends_on: [1]
parallel: true
parallel_with: [3, 4, 5]
effort: 1h
---

# Phase 2: Site & Nginx Handlers

## Context
Site operations (5 TODOs) and Nginx operations (3 TODOs) need async handler methods. These methods live in new files, called from app.go Update() in Phase 6.

## Parallelization
Runs in parallel with Phases 3, 4, 5. No shared file writes.

## File Ownership (exclusive)
- `internal/tui/app_handlers_site.go` -- NEW
- `internal/tui/app_handlers_nginx.go` -- NEW

## Overview
Create handler methods on `*App` that call backend managers and return tea.Cmd. Each follows the established pattern:

```go
func (a *App) handleX(params) tea.Cmd {
    if a.mgr == nil { return nil }
    return func() tea.Msg {
        result, err := a.mgr.DoThing(context.Background(), params)
        if err != nil { return OpErrMsg{Err: err} }
        return OpDoneMsg{}
    }
}
```

## app_handlers_site.go

### Methods to implement:

**fetchSites()** -- Load site list for SiteList screen
```go
func (a *App) fetchSites() tea.Cmd {
    if a.siteMgr == nil { return nil }
    return func() tea.Msg {
        sites, err := a.siteMgr.List()
        if err != nil { return SiteListErrMsg{Err: err} }
        return SiteListMsg{Sites: sites}
    }
}
```

**fetchSiteDetail(domain)** -- Load single site for SiteDetail screen
```go
func (a *App) fetchSiteDetail(domain string) tea.Cmd {
    if a.siteMgr == nil { return nil }
    return func() tea.Msg {
        s, err := a.siteMgr.Get(domain)
        if err != nil { return SiteOpErrMsg{Err: err} }
        return SiteDetailMsg{Site: s}  // add to app_messages.go if needed
    }
}
```

**handleCreateSite(opts site.CreateOptions)** -- Create site async
```go
func (a *App) handleCreateSite(opts site.CreateOptions) tea.Cmd {
    if a.siteMgr == nil { return nil }
    return func() tea.Msg {
        s, err := a.siteMgr.Create(opts)
        if err != nil { return SiteOpErrMsg{Err: err} }
        return SiteCreatedMsg{Site: s}
    }
}
```

**handleToggleSite(domain)** -- Enable/disable site
```go
func (a *App) handleToggleSite(domain string) tea.Cmd {
    if a.siteMgr == nil { return nil }
    return func() tea.Msg {
        s, err := a.siteMgr.Get(domain)
        if err != nil { return SiteOpErrMsg{Err: err} }
        if s.Enabled {
            err = a.siteMgr.Disable(domain)
        } else {
            err = a.siteMgr.Enable(domain)
        }
        if err != nil { return SiteOpErrMsg{Err: err} }
        return SiteOpDoneMsg{}
    }
}
```

**handleDeleteSite(domain)** -- Delete site
```go
func (a *App) handleDeleteSite(domain string) tea.Cmd {
    if a.siteMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.siteMgr.Delete(domain, false); err != nil {
            return SiteOpErrMsg{Err: err}
        }
        return SiteOpDoneMsg{}
    }
}
```

## app_handlers_nginx.go

### Methods to implement:

**fetchVhosts()** -- Load vhost list for Nginx screen
```go
func (a *App) fetchVhosts() tea.Cmd {
    if a.nginxMgr == nil { return nil }
    return func() tea.Msg {
        vhosts, err := a.nginxMgr.List()
        if err != nil { return VhostListErrMsg{Err: err} }
        return VhostListMsg{Vhosts: vhosts}
    }
}
```

**handleToggleVhost(domain, currentlyEnabled)** -- Enable/disable vhost
```go
func (a *App) handleToggleVhost(domain string, enabled bool) tea.Cmd {
    if a.nginxMgr == nil { return nil }
    return func() tea.Msg {
        var err error
        if enabled {
            err = a.nginxMgr.Disable(domain)
        } else {
            err = a.nginxMgr.Enable(domain)
        }
        if err != nil { return NginxOpErrMsg{Err: err} }
        return NginxOpDoneMsg{}
    }
}
```

**handleDeleteVhost(domain)** -- Delete vhost
```go
func (a *App) handleDeleteVhost(domain string) tea.Cmd {
    if a.nginxMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.nginxMgr.Delete(domain); err != nil {
            return NginxOpErrMsg{Err: err}
        }
        return NginxOpDoneMsg{}
    }
}
```

**handleTestNginx()** -- Test nginx config
```go
func (a *App) handleTestNginx() tea.Cmd {
    if a.nginxMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.nginxMgr.Test(); err != nil {
            return NginxOpErrMsg{Err: err}
        }
        return NginxTestOkMsg{}
    }
}
```

## Implementation Steps

- [ ] Create `internal/tui/app_handlers_site.go` with package `tui`
- [ ] Implement fetchSites, fetchSiteDetail, handleCreateSite, handleToggleSite, handleDeleteSite
- [ ] Create `internal/tui/app_handlers_nginx.go` with package `tui`
- [ ] Implement fetchVhosts, handleToggleVhost, handleDeleteVhost, handleTestNginx
- [ ] Verify imports compile: `go build ./internal/tui/...`

## Success Criteria
- Both files compile as part of `tui` package
- Methods match message types from Phase 1's app_messages.go
- Nil-safe: all methods return nil cmd when manager is nil

## Conflict Prevention
- These files are NEW -- no merge conflicts
- Only access App struct via `a.siteMgr`, `a.nginxMgr` (read-only)
- Do NOT modify app.go Update() -- that's Phase 6

## Risk Assessment
- **Low:** site.Manager.Nginx() exposes nginx.Manager, but we have a separate nginxMgr on App. Prefer the standalone nginxMgr for direct vhost ops (List, Enable, Disable). siteMgr creates its own internal nginx.Manager -- this is fine, they share the same executor.
- **Note:** `site.Manager.Delete(domain, removeDB)` takes a bool. For now pass `false`. Future: could ask user via confirmation dialog.
