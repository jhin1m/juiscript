# Phase 04: TUI Setup Screen

## Context

Interactive Bubble Tea screen for selecting and installing LEMP packages. Uses state machine pattern consistent with existing TUI screens.

## Overview

Create `internal/tui/screens/setup.go` with 4 states: checklist (multi-select) -> confirm -> installing (spinner + progress) -> done (summary). Follows existing screen patterns (pointer receiver, theme, NavigateMsg/GoBackMsg).

## Key Insights

- State machine pattern: route Update/View by current state (matches research patterns)
- Multi-select: `map[int]bool` for selected items, space toggles, j/k navigates
- Spinner: `bubbles/spinner.Model` with `Tick` cmd for animation
- Progress: receive `ProgressEvent` as tea.Msg from background tea.Cmd
- Must NOT block Update() — install runs in tea.Cmd (background goroutine)
- Pre-select missing packages by default (user deselects what they don't want)

## Requirements

1. `SetupScreen` struct with state machine
2. States: `stateChecklist`, `stateConfirm`, `stateInstalling`, `stateDone`
3. Checklist state:
   - Display PackageInfo list with [x]/[ ] checkboxes
   - Show installed/missing status per item
   - Pre-select all missing packages
   - Space toggles, j/k navigates, Enter confirms selection
   - Esc goes back to dashboard
4. Confirm state:
   - Show selected packages for confirmation
   - Enter proceeds to install, Esc back to checklist
5. Installing state:
   - Spinner animation
   - Per-package progress lines: "Installing Nginx..." → "Nginx installed" / "Nginx failed: ..."
   - No keyboard input (blocked during install)
6. Done state:
   - Summary: N installed, N skipped, N failed
   - Per-package result lines
   - Enter/Esc returns to dashboard
7. Messages:
   - `SetupProgressMsg` wraps `ProgressEvent` from provisioner
   - `SetupDoneMsg` wraps `InstallSummary` (signals completion)
   - `RunSetupMsg` sent from app.go to trigger install
8. Constructor: `NewSetupScreen(t *theme.Theme) *SetupScreen`
9. `SetPackages(pkgs []PackageInfo)` to populate from detection

## Architecture

```go
// internal/tui/screens/setup.go

type setupState int
const (
    stateChecklist setupState = iota
    stateConfirm
    stateInstalling
    stateDone
)

type SetupScreen struct {
    theme    *theme.Theme
    state    setupState
    packages []provisioner.PackageInfo
    selected map[int]bool
    cursor   int
    spinner  spinner.Model
    progress []progressLine    // per-package status during install
    summary  *provisioner.InstallSummary
    width    int
    height   int
}

type progressLine struct {
    Name    string
    Status  string  // "installing", "done", "failed"
    Message string
}

// Messages
type SetupProgressMsg struct { Event provisioner.ProgressEvent }
type SetupDoneMsg struct { Summary *provisioner.InstallSummary }
type RunSetupMsg struct { Names []string }
```

### State Machine Flow

```
stateChecklist ──[enter]──> stateConfirm ──[enter]──> stateInstalling ──[done]──> stateDone
       ^──[esc]──┘                ^──[esc]──┘                                        │
       │                                                                             │
   GoBackMsg <────────────────────────────────────────────────────────[enter/esc]─────┘
```

### Install Command Pattern (in app.go, not setup.go)

```go
// app.go handles RunSetupMsg → spawns tea.Cmd that calls provisioner
case screens.RunSetupMsg:
    return a, func() tea.Msg {
        summary, _ := a.provisioner.InstallSelected(ctx, msg.Names, func(ev ProgressEvent) {
            // Problem: can't send tea.Msg from inside callback
            // Solution: use channel + polling cmd
        })
        return screens.SetupDoneMsg{Summary: summary}
    }
```

**Better approach**: Use channel for progress streaming (from research):

```go
case screens.RunSetupMsg:
    ch := make(chan provisioner.ProgressEvent, 10)
    go func() {
        a.provisioner.InstallSelected(ctx, msg.Names, func(ev) { ch <- ev })
        close(ch)
    }()
    return a, waitForProgress(ch)

func waitForProgress(ch <-chan provisioner.ProgressEvent) tea.Cmd {
    return func() tea.Msg {
        ev, ok := <-ch
        if !ok { return screens.SetupDoneMsg{} }
        return screens.SetupProgressMsg{Event: ev}
    }
}
// In Update: after SetupProgressMsg, re-issue waitForProgress(ch)
```

Store channel in App struct: `setupProgressCh chan provisioner.ProgressEvent`

## Related Code Files

- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/dashboard.go` - screen pattern reference
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/services.go` - table + cursor pattern
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/php.go` - install message pattern
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/theme/theme.go` - styles

## Implementation Steps

1. Create `setup.go`:
   - Define state enum, progressLine struct, messages
   - `SetupScreen` struct with all fields
   - `NewSetupScreen(t)`: init spinner (spinner.Dot), empty state
   - `SetPackages(pkgs)`: populate packages, pre-select missing ones
   - `Init()`: return `spinner.Tick`
   - `Update()`: route by state
     - stateChecklist: j/k nav, space toggle, enter → stateConfirm (if any selected), esc → GoBackMsg
     - stateConfirm: enter → emit RunSetupMsg + switch to stateInstalling, esc → stateChecklist
     - stateInstalling: handle SetupProgressMsg (update progress lines), SetupDoneMsg (switch to stateDone), spinner.TickMsg
     - stateDone: enter/esc → GoBackMsg
   - `View()`: route by state
     - stateChecklist: title + checklist items with cursor/checkbox/status
     - stateConfirm: "Install these packages?" + selected list + enter/esc hint
     - stateInstalling: spinner + progress lines
     - stateDone: summary stats + per-package results
   - `selectedNames()`: returns []string of selected package Names

2. Style considerations:
   - Installed items: green check, dimmed text
   - Missing items: red X, normal text
   - During install: spinner.Dot style, green for done, red for failed
   - Summary: green count for success, red for failed

## Todo

- [ ] Define state enum and message types
- [ ] Implement `SetupScreen` struct
- [ ] Implement `NewSetupScreen` + `SetPackages`
- [ ] Implement checklist state (Update + View)
- [ ] Implement confirm state (Update + View)
- [ ] Implement installing state with spinner + progress
- [ ] Implement done state with summary
- [ ] Implement `selectedNames()` helper
- [ ] Verify spinner animation works with Init() returning Tick

## Success Criteria

- Smooth state transitions between all 4 states
- Pre-selects missing packages, allows deselection
- Spinner animates during install
- Progress lines update in real-time per package
- Summary shows accurate counts
- Esc navigates back appropriately from each state

## Risk Assessment

- **Medium**: Progress streaming via channel + tea.Cmd chaining — must test that all events are received and channel is properly closed
- **Low**: Spinner stops animating if Tick cmd not re-issued — handle in stateInstalling Update
- **Low**: Large package list might need scroll — 4 packages max currently, safe

## Security

- No direct system calls — delegates to provisioner via messages
- Package names come from Detector, not user input

## Next Steps

Phase 05 wires SetupScreen into app.go and dashboard.go.
