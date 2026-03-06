# Phase 3: Toast Notification Component

## Context
- [Plan overview](./plan.md)
- Independent of Phases 1-2

## Overview
Auto-dismissing notification for operation results. Lives in `App` model, renders as overlay in `App.View()`. Uses `tea.Tick` for auto-dismiss.

## Key Insights
- Currently, success results (DBOpDoneMsg, SSLOpDoneMsg, etc.) refresh data silently - no user feedback
- Errors show via screen's `SetError()` which persists until next action - not great UX
- Toast should be lightweight: type + message + auto-dismiss timer
- Single toast at a time (new toast replaces old)

## Requirements
1. Three variants: Success (green), Error (red), Warning (amber)
2. Auto-dismiss after configurable duration (default 3s)
3. Renders at bottom of App.View() above status bar
4. Manual dismiss with any keypress
5. Single active toast (last one wins)

## Architecture

### File: `internal/tui/components/toast.go`

```go
type ToastType int
const (
    ToastSuccess ToastType = iota
    ToastError
    ToastWarning
)

// ToastModel manages a single notification.
type ToastModel struct {
    theme    *theme.Theme
    message  string
    toastType ToastType
    visible  bool
    id       int  // incremented per show, used to match dismiss tick
}

// ToastDismissMsg is sent by tea.Tick to auto-dismiss.
type ToastDismissMsg struct {
    ID int
}

// ShowToastMsg can be sent from anywhere to trigger a toast.
type ShowToastMsg struct {
    Type    ToastType
    Message string
}
```

### Key Methods
- `NewToast(t *theme.Theme) *ToastModel`
- `Show(typ ToastType, message string) tea.Cmd` - sets visible, returns tick cmd for auto-dismiss
- `Update(msg tea.Msg) (*ToastModel, tea.Cmd)` - handles ToastDismissMsg
- `View() string` - returns styled notification line or empty string
- `Visible() bool`

### Auto-Dismiss Flow
```
App receives DBOpDoneMsg
  -> toast.Show(ToastSuccess, "Database created successfully")
  -> returns tea.Tick(3s, func() ToastDismissMsg{ID: n})

After 3s, App receives ToastDismissMsg{ID: n}
  -> toast.Update(msg) -> if msg.ID == toast.id, hide toast
```

### View Layout
```
  [normal screen content]

  SUCCESS: Database created successfully     <- green styled, one line

  ctrl+c:quit  ...                           <- status bar
```

## Related Code Files
- `internal/tui/app.go` - App.View() will render toast, App.Update() handles ShowToastMsg
- `internal/tui/app_messages.go` - OpDone/OpErr messages trigger toast

## Implementation Steps

### TODO
- [x] Define `ToastType`, `ToastModel`, `ToastDismissMsg`, `ShowToastMsg` types
- [x] Implement `NewToast()` constructor
- [x] Implement `Show()` returning tea.Tick cmd with matching ID
- [x] Implement `Update()` checking ID match for dismiss
- [x] Implement `View()` with type-based coloring (Success=OkText, Error=ErrorText, Warning=WarnText)
- [x] Write tests: show sets visible, dismiss with matching ID hides, mismatched ID ignores, view output

### Test File: `internal/tui/components/toast_test.go`
- TestToast_Show - message and type set correctly, visible true
- TestToast_DismissMatchingID - matching ID hides toast
- TestToast_DismissMismatchID - old ID ignored, toast stays visible
- TestToast_ViewVariants - success/error/warning render with correct style prefix

## Success Criteria
- Operations produce visible feedback
- Toast auto-dismisses (no stale messages)
- Does not interfere with screen navigation
- 80%+ test coverage

## Risk Assessment
- **Low**: Tick message arrives after screen change. Mitigate: ID-based matching ensures stale ticks are ignored
- **Low**: Toast overlaps important content. Mitigate: single line, rendered between content and status bar

## Security Considerations
- Toast messages should not include sensitive data (passwords, tokens)
- Error messages from backend should be sanitized before display

## Next Steps
Phase 6 wires toast into App's Update() for all OpDone/OpErr result messages.
