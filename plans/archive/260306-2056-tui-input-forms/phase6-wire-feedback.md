# Phase 6: Wire Feedback Components

## Context
- [Plan overview](./plan.md)
- Depends on: Phases 2 (ConfirmModel), 3 (ToastModel), 4 (SpinnerModel), 5 (forms wired)

## Overview
Integrate confirmation dialogs, toast notifications, and spinners into the App and screens. This is the UX polish phase - all backend operations already work from Phase 5.

## Key Insights
- Toast lives in App (single instance, renders in App.View)
- Spinner lives in individual screens (each screen owns its spinner)
- Confirm dialog lives in individual screens (context-specific messages)
- All OpDoneMsg/OpErrMsg types should trigger toast
- Destructive actions: drop DB, delete site, revoke cert, delete backup, restore backup

## Requirements
1. Toast in App for all success/error operation results
2. Spinner in screens during long-running ops (PHP install, backup create/restore, SSL obtain)
3. Confirmation before: drop DB, delete site, revoke cert, delete backup, restore backup
4. Confirmation before PHP remove (sites may depend on version)

## Architecture

### Toast Integration (app.go)

Add to App struct:
```go
toast *components.ToastModel
```

In App.Update(), convert OpDone/OpErr to toast:
```go
case DBOpDoneMsg:
    cmd := a.toast.Show(components.ToastSuccess, "Database operation completed")
    return a, tea.Batch(cmd, a.fetchDatabases())

case DBOpErrMsg:
    cmd := a.toast.Show(components.ToastError, msg.Err.Error())
    return a, cmd
```

In App.View(), insert toast between content and status bar:
```go
toastLine := a.toast.View()
return fmt.Sprintf("%s\n%s\n\n%s\n%s\n%s", header, svcBar, content, toastLine, statusBar)
```

### Spinner Integration (per screen)

Screens with long ops embed SpinnerModel:
- PHPScreen: install (apt-get, can take minutes)
- BackupScreen: create, restore (file operations)
- SSLScreen: obtain (certbot, network call)

Pattern in screen Update:
```go
// On form submit -> start spinner + emit msg
case components.FormSubmitMsg:
    spinCmd := s.spinner.Start("Installing PHP " + values["version"] + "...")
    installCmd := func() tea.Msg { return InstallPHPMsg{Version: values["version"]} }
    return s, tea.Batch(spinCmd, installCmd)

// On result msg received by App -> screen's spinner stopped via method call
```

App stops spinner on result:
```go
case PHPVersionsMsg:
    a.phpScreen.StopSpinner()
    a.phpScreen.SetVersions(msg.Versions)
case PHPVersionsErrMsg:
    a.phpScreen.StopSpinner()
    a.phpScreen.SetError(msg.Err)
```

### Confirmation Integration (per screen)

Screens with destructive actions embed ConfirmModel:
- DatabaseScreen: drop DB
- SSLScreen: revoke cert
- BackupScreen: delete backup, restore backup
- PHPScreen: remove version
- SiteDetail: delete site (already has action list)

Pattern:
```go
// 'd' key for drop DB
case "d":
    if len(d.dbs) > 0 {
        d.confirm.Show(fmt.Sprintf("Drop database '%s'? This cannot be undone.", d.dbs[d.cursor].Name))
        d.pendingAction = "drop"
        d.pendingTarget = d.dbs[d.cursor].Name
    }

// Confirm result
case components.ConfirmYesMsg:
    switch d.pendingAction {
    case "drop":
        return d, func() tea.Msg { return DropDBMsg{Name: d.pendingTarget} }
    }
case components.ConfirmNoMsg:
    d.pendingAction = ""
```

### Screens Needing Changes

| Screen | Spinner | Toast | Confirm |
|--------|---------|-------|---------|
| App | - | Yes (single instance) | - |
| PHPScreen | Install | - | Remove |
| DatabaseScreen | - | - | Drop |
| SSLScreen | Obtain | - | Revoke |
| BackupScreen | Create, Restore | - | Delete, Restore |
| SiteDetail | - | - | Delete |

## Related Code Files
- `internal/tui/app.go` - add toast, wire toast in Update/View
- `internal/tui/components/toast.go` - Phase 3
- `internal/tui/components/spinner.go` - Phase 4
- `internal/tui/components/confirm.go` - Phase 2
- `internal/tui/screens/php.go` - add spinner + confirm
- `internal/tui/screens/database.go` - add confirm for drop
- `internal/tui/screens/ssl.go` - add spinner + confirm
- `internal/tui/screens/backup.go` - add spinner + confirm
- `internal/tui/screens/sitedetail.go` - add confirm for delete

## Implementation Steps

### TODO

**Toast (App level)**
- [x] Add `toast *components.ToastModel` to App struct
- [x] Initialize toast in `NewApp()`
- [x] Add `ToastDismissMsg` case in App.Update() forwarding to toast
- [x] Convert all `OpDoneMsg` handlers to include toast.Show success cmd
- [x] Convert all `OpErrMsg` handlers to include toast.Show error cmd
- [x] Update App.View() to render toast line between content and status bar
- [x] Specific toast messages per operation (e.g., "PHP 8.3 installed", "Database 'mydb' dropped")

**Spinner (per screen)**
- [x] Add `spinner *components.SpinnerModel` to PHPScreen, SSLScreen, BackupScreen
- [x] Initialize spinners in constructors
- [x] Add `StopSpinner()` method to each screen
- [x] Start spinner on form submit (before emitting action msg)
- [x] Forward `spinner.TickMsg` in screen Update() when spinner active
- [x] Stop spinner in App when result msg received (call screen.StopSpinner())
- [x] Render spinner.View() in screen View() when active (replaces list content)

**Confirmation (per screen)**
- [x] Add `confirm *components.ConfirmModel`, `pendingAction string`, `pendingTarget string` to screens
- [x] Initialize confirm in constructors
- [x] Gate destructive key handlers to show confirm instead of immediate action
- [x] Handle ConfirmYesMsg -> emit original action msg
- [x] Handle ConfirmNoMsg -> clear pending state
- [x] Render confirm.View() in screen View() when active (replaces list content)
- [x] Wire confirm for: drop DB, remove PHP, revoke cert, delete backup, restore backup, delete site

**Testing**
- [x] Run `make test` - all existing tests pass
- [x] Manual test: trigger each form, verify spinner shows, toast appears, confirm gates actions

## Success Criteria
- Every async operation shows a spinner while in progress
- Every operation result (success or error) shows a toast
- Every destructive action requires confirmation
- No regressions in existing functionality
- Clean UX flow: action key -> form -> spinner -> result toast

## Risk Assessment
- **Medium**: Many files modified at once. Mitigate: commit after each sub-section (toast, then spinner, then confirm)
- **Low**: Spinner tick messages from bubbles may conflict with other messages. Mitigate: only forward tick when spinner.Active()
- **Low**: Toast auto-dismiss timing feels wrong. Mitigate: make duration configurable, start with 3s

## Security Considerations
- Confirmation dialog is critical security gate - must not be bypassable
- Error toast messages may leak internal paths; sanitize before display
- Spinner should not prevent ctrl+c quit

## Next Steps
After this phase, TUI Phase 2 is complete. All screens have full input forms, feedback components, and safety gates. Suggested follow-up: keyboard shortcut help overlay, responsive layout for small terminals.
