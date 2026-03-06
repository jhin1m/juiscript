# Phase 2: Confirmation Dialog Component

## Context
- [Plan overview](./plan.md)
- Depends on: theme system only (independent of Phase 1)

## Overview
Lightweight confirm dialog for destructive actions (drop DB, delete site, revoke cert). Renders inline within the parent screen's View.

## Key Insights
- Destructive actions currently fire immediately on keypress (no safety net)
- sitedetail.go already has action list pattern but no confirm gate
- Dialog must be simple: message + yes/no, styled with Error/Warning colors
- Parent screen manages `confirmActive bool` state

## Requirements
1. Display warning message with destructive action description
2. Yes/No selection with keyboard navigation (tab/y/n/enter)
3. Red/warning styling for destructive context
4. Returns `ConfirmYesMsg{}` or `ConfirmNoMsg{}` on selection
5. Esc always cancels (same as No)

## Architecture

### File: `internal/tui/components/confirm.go`

```go
// ConfirmModel is an inline confirmation dialog.
type ConfirmModel struct {
    theme    *theme.Theme
    message  string      // e.g., "Drop database 'mydb'? This cannot be undone."
    selected bool        // false=No (default safe), true=Yes
    active   bool
}

// ConfirmYesMsg signals user confirmed the action.
type ConfirmYesMsg struct{}

// ConfirmNoMsg signals user cancelled.
type ConfirmNoMsg struct{}
```

### Key Methods
- `NewConfirm(t *theme.Theme) *ConfirmModel`
- `Show(message string)` - activate with message, reset to No
- `Update(msg tea.Msg) (*ConfirmModel, tea.Cmd)`
- `View() string` - renders warning box
- `Active() bool`

### Key Bindings
```
y      -> select Yes
n/esc  -> select No, emit ConfirmNoMsg
tab    -> toggle Yes/No
enter  -> emit based on current selection
```

### View Layout
```
  +--------------------------------------+
  |  WARNING                             |
  |                                      |
  |  Drop database 'mydb'?              |
  |  This cannot be undone.              |
  |                                      |
  |  [ No ]   [ Yes ]                   |
  +--------------------------------------+
```
- "Yes" rendered in Error color when selected
- "No" rendered in OkText when selected
- Border uses theme.Panel with ErrorText border color

## Related Code Files
- `internal/tui/theme/theme.go` - Panel, ErrorText, OkText styles
- `internal/tui/screens/sitedetail.go` - will use this for delete action

## Implementation Steps

### TODO
- [x] Define `ConfirmModel` struct with message, selected, active fields
- [x] Implement `NewConfirm()` constructor
- [x] Implement `Show(message)` to activate and reset selection to No
- [x] Implement `Update()` with y/n/tab/enter/esc key handling
- [x] Implement `View()` with warning-styled panel
- [x] Write tests: show/hide, toggle, confirm yes, confirm no, esc cancels

### Test File: `internal/tui/components/confirm_test.go`
- TestConfirm_DefaultNo - starts with No selected
- TestConfirm_TabToggle - tab switches between Yes/No
- TestConfirm_YKey - y immediately confirms
- TestConfirm_NKey - n immediately cancels
- TestConfirm_EscCancels - esc emits ConfirmNoMsg
- TestConfirm_EnterSubmits - enter emits based on selection

## Success Criteria
- Safe default (No selected initially)
- Clear visual distinction between Yes/No
- Warning styling communicates danger
- 80%+ test coverage

## Risk Assessment
- **Low**: Dialog might overlap with screen content. Mitigate: render dialog INSTEAD of normal content when active

## Security Considerations
- Confirm dialog is the security gate for destructive operations
- Must default to No to prevent accidental data loss

## Next Steps
Phase 6 wires this into screens for: drop DB, delete site, revoke cert, delete backup, restore backup.
