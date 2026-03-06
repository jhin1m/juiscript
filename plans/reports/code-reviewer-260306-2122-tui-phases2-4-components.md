# Code Review: TUI Phases 2-4 Components

## Scope
- Files reviewed: confirm.go, confirm_test.go, toast.go, toast_test.go, spinner.go (5 files, ~230 LOC)
- Review focus: Phase 2 (confirm dialog), Phase 3 (toast), Phase 4 (spinner)
- Plans: phase2-confirm-dialog.md, phase3-toast.md, phase4-spinner.md

## Overall Assessment
All three components are clean, minimal, and correctly implemented. Tests pass. No critical issues. A few low-priority observations below.

---

## Critical Issues
None.

---

## High Priority Findings
None.

---

## Medium Priority Improvements

### 1. `confirm.go`: Panel style recreated on every `View()` call
`lipgloss.NewStyle()` in `View()` allocates a new style struct each render cycle. Since BubbleTea calls `View()` on every frame, this is wasteful for a static style.

**Fix:** Move the panel style to `ConfirmModel` or package-level `var`.

```go
var confirmPanelStyle = lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    BorderForeground(theme.Warning).
    Padding(1, 2)
```

Then use `confirmPanelStyle.Render(content)` in `View()`.

### 2. `toast.go`: `toastDurations` map lookup returns zero-value for unknown `ToastType`
If an invalid `ToastType` is passed, `toastDurations[typ]` returns `0` and `tea.Tick(0, ...)` fires immediately, causing an instant dismiss with no feedback.

**Fix:** Add a fallback default duration:
```go
duration, ok := toastDurations[typ]
if !ok {
    duration = 3 * time.Second
}
```

Or add a `default` case in the `View()` switch too (currently silent for unknown types — renders empty prefix with message appended, which looks broken).

---

## Low Priority Suggestions

### 3. `confirm.go`: `y` key does not toggle selection — it immediately confirms
This is intentional per the plan, but the UX is subtly asymmetric: `y` fires `ConfirmYesMsg` instantly regardless of current selection, but `n` also fires immediately. This means if the cursor is on "Yes" and user presses `y`, they get expected behavior; but there's no way to "type y to move to Yes then enter to confirm" flow. Not a bug — just worth noting for UX consistency.

### 4. `spinner.go`: `theme` field stored but never used
`SpinnerModel.theme` is assigned in `NewSpinner()` but never referenced in any method (`View()` and `Update()` don't use it). The only theme usage is `theme.Primary` at construction time via `s.Style.Foreground(theme.Primary)`.

YAGNI suggests either removing the field or adding a `TODO` comment if future theme-switching is planned. Minor — no functional impact.

### 5. `spinner.go`: No test file
Phase 4 plan explicitly documents "No separate test file needed (thin wrapper)." Acceptable given the wrapper nature, but `Start()`/`Stop()`/`Active()` lifecycle could be covered with 3 unit tests at near-zero cost. Noted for future if coverage requirements tighten.

---

## Positive Observations
- **ID-based toast dismissal** is a clean solution to the stale-tick problem — well thought out.
- **Safe default (No)** in confirm dialog is the correct UX for destructive action gates.
- **`active` guard in `Update()`** on both confirm and spinner prevents spurious state changes when hidden.
- Test coverage is thorough for confirm (9 tests) and toast (6 tests), covering the critical ID-mismatch edge case.
- All three components follow consistent pattern: struct + constructor + Start/Show + Update + View + Active/Visible.
- Zero dependencies beyond bubbletea/bubbles/lipgloss — no unnecessary coupling.

---

## Task Completeness

| Phase | Plan TODOs | Status |
|-------|-----------|--------|
| Phase 2 (confirm) | 6 items | All complete |
| Phase 3 (toast) | 5 items | All complete |
| Phase 4 (spinner) | 7 items | All complete (bubbles already direct dep via bubbletea) |

All phase TODO checklists are marked complete. Tests pass (`ok github.com/jhin1m/juiscript/internal/tui/components 0.154s`).

---

## Recommended Actions
1. **(Medium)** Extract `confirmPanelStyle` to package-level var in `confirm.go` — avoid re-allocation per frame.
2. **(Medium)** Add fallback duration in `toast.go` for unknown `ToastType` to prevent instant dismiss.
3. **(Low)** Remove or annotate unused `theme` field in `SpinnerModel` per YAGNI.

---

## Metrics
- Test Coverage: 15 tests across 2 files; spinner untested (by design)
- Linting Issues: 0 compile errors, 0 test failures
- Build: passing
