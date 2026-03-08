---
phase: 4
title: "SSL & Service Handler Methods"
status: pending
depends_on: [1]
parallel: true
parallel_with: [2, 3, 5]
effort: 0.5h
---

# Phase 4: SSL & Service Handlers

## Context
SSL operations (3 TODOs) and Service operations (4 TODOs). Service handlers are straightforward since the pattern (execute + re-fetch) is already established in app.go comments.

## Parallelization
Runs in parallel with Phases 2, 3, 5.

## File Ownership (exclusive)
- `internal/tui/app_handlers_ssl.go` -- NEW
- `internal/tui/app_handlers_service.go` -- NEW

## app_handlers_ssl.go

**fetchCerts()** -- Load certificate list
```go
func (a *App) fetchCerts() tea.Cmd {
    if a.sslMgr == nil { return nil }
    return func() tea.Msg {
        certs, err := a.sslMgr.ListCerts(context.Background())
        if err != nil { return CertListErrMsg{Err: err} }
        return CertListMsg{Certs: certs}
    }
}
```

**handleObtainCert()** -- Obtain SSL cert
Note: ObtainCertMsg has no fields. Needs domain + email input. Placeholder.
```go
func (a *App) handleObtainCert() tea.Cmd {
    if a.sslMgr == nil { return nil }
    return func() tea.Msg {
        return SSLOpErrMsg{Err: fmt.Errorf("SSL obtain requires domain and email input (not yet implemented)")}
    }
}
```

**handleRevokeCert(domain)** -- Revoke cert
```go
func (a *App) handleRevokeCert(domain string) tea.Cmd {
    if a.sslMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.sslMgr.Revoke(context.Background(), domain); err != nil {
            return SSLOpErrMsg{Err: err}
        }
        return SSLOpDoneMsg{}
    }
}
```

**handleRenewCert(domain)** -- Renew cert
```go
func (a *App) handleRenewCert(domain string) tea.Cmd {
    if a.sslMgr == nil { return nil }
    return func() tea.Msg {
        if err := a.sslMgr.Renew(context.Background(), domain); err != nil {
            return SSLOpErrMsg{Err: err}
        }
        return SSLOpDoneMsg{}
    }
}
```

## app_handlers_service.go

All 4 service handlers follow identical pattern: call svcMgr action, then re-fetch status.

**handleServiceAction(name, action)** -- Generic service action
```go
func (a *App) handleServiceAction(name service.ServiceName, action string) tea.Cmd {
    if a.svcMgr == nil { return nil }
    return func() tea.Msg {
        ctx := context.Background()
        var err error
        switch action {
        case "start":
            err = a.svcMgr.Start(ctx, name)
        case "stop":
            err = a.svcMgr.Stop(ctx, name)
        case "restart":
            err = a.svcMgr.Restart(ctx, name)
        case "reload":
            err = a.svcMgr.Reload(ctx, name)
        }
        if err != nil {
            return ServiceOpErrMsg{Err: err}
        }
        // Re-fetch status after action
        statuses, err := a.svcMgr.ListAll(ctx)
        if err != nil {
            return ServiceStatusErrMsg{Err: err}
        }
        return ServiceStatusMsg{Services: statuses}
    }
}
```

This consolidates 4 nearly identical handlers into 1 method. DRY.

## Implementation Steps

- [ ] Create `internal/tui/app_handlers_ssl.go` with fetchCerts, handleObtainCert, handleRevokeCert, handleRenewCert
- [ ] Create `internal/tui/app_handlers_service.go` with handleServiceAction
- [ ] Verify imports compile

## Success Criteria
- Both files compile
- Service handler consolidates 4 actions into 1 method (DRY)
- SSL handlers match ssl.Manager method signatures (Revoke, Renew take domain + ctx)

## Conflict Prevention
- New files only

## Risk Assessment
- **Medium:** ObtainCertMsg needs domain + email. Placeholder returns error. Future: needs form screen.
- **Low:** Service action uses string switch. Could use typed constant but KISS -- only 4 values, all internal.
- **Verify:** ssl.Manager.Revoke/Renew signatures -- confirm they accept `(ctx, domain)`.
