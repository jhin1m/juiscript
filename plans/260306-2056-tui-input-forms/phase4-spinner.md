# Phase 4: Spinner Component

## Context
- [Plan overview](./plan.md)
- Independent of Phases 1-3

## Overview
Themed spinner wrapper for indicating long-running async operations (PHP install, backup create, SSL obtain). Uses bubbles spinner internally, styled with theme colors.

## Key Insights
- `charmbracelet/bubbles` is already an indirect dependency - can promote to direct
- Spinner needs: start/stop, custom message text, theme-consistent colors
- Each screen manages its own spinner state (embed SpinnerModel)
- Spinner replaces normal view content while active

## Requirements
1. Wraps bubbles spinner with theme styling
2. Configurable loading message (e.g., "Installing PHP 8.3...")
3. Start/Stop methods for clean lifecycle
4. Integrates as embedded component in screens

## Architecture

### File: `internal/tui/components/spinner.go`

```go
import "github.com/charmbracelet/bubbles/spinner"

// SpinnerModel wraps bubbles spinner with theme styling.
type SpinnerModel struct {
    theme   *theme.Theme
    spinner spinner.Model
    message string
    active  bool
}
```

### Key Methods
- `NewSpinner(t *theme.Theme) *SpinnerModel`
- `Start(message string) tea.Cmd` - activate spinner, return spinner.Tick
- `Stop()` - deactivate
- `Update(msg tea.Msg) (*SpinnerModel, tea.Cmd)` - forward spinner.TickMsg
- `View() string` - returns "spinner_char message" or empty
- `Active() bool`

### Integration Pattern
```go
// In screen's Update():
case tea.KeyMsg:
    if msg.String() == "i" && !s.spinner.Active() {
        return s, tea.Batch(
            s.spinner.Start("Installing PHP 8.3..."),
            func() tea.Msg { return InstallPHPMsg{Version: "8.3"} },
        )
    }

// In screen's View():
if s.spinner.Active() {
    return s.spinner.View()  // show spinner instead of normal content
}
```

## Related Code Files
- `go.sum` - bubbles already in dependency tree
- `internal/tui/screens/php.go` - will embed spinner for install
- `internal/tui/screens/backup.go` - will embed spinner for create/restore

## Implementation Steps

### TODO
- [x] Add `github.com/charmbracelet/bubbles` as direct dependency if needed (`go get`)
- [x] Define `SpinnerModel` struct wrapping bubbles spinner
- [x] Implement `NewSpinner()` with theme.Primary color and Dot style
- [x] Implement `Start(message)` returning spinner tick cmd
- [x] Implement `Stop()` to deactivate
- [x] Implement `Update()` forwarding spinner.TickMsg when active
- [x] Implement `View()` combining spinner char + message text
- [x] No separate test file needed (thin wrapper); test via integration in Phase 6

## Success Criteria
- Spinner animates during async operations
- Styled consistently with theme colors
- Clean start/stop lifecycle
- Does not block UI (non-blocking via tea.Cmd)

## Risk Assessment
- **Low**: bubbles version mismatch. Mitigate: already in go.sum as indirect dep
- **Low**: Spinner tick messages leak after stop. Mitigate: Active() check in Update()

## Security Considerations
- None - purely visual component

## Next Steps
Phase 6 embeds spinner into screens that perform long-running operations.
