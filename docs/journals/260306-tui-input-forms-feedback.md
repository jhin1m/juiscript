# Journal: TUI Input Forms & Feedback

**Date:** 2026-03-06
**Plan:** `plans/260306-2056-tui-input-forms/plan.md`
**Status:** Completed (all 7 phases)

## Summary

Added reusable form, confirmation, toast, and spinner components to TUI. Wired them into all screens (PHP, Database, SSL, Backup, SiteDetail). All 6 placeholder handlers now accept real user input.

## Key Changes

- **FormModel** (`components/form.go`): Data-driven step-by-step form with 3 field types (text, select, confirm). Screens embed it and toggle `formActive` bool.
- **ConfirmModel** (`components/confirm.go`): Inline yes/no dialog for destructive actions. Safe default: No.
- **ToastModel** (`components/toast.go`): Auto-dismissing notification with ID-based stale prevention. 3s success, 5s error.
- **SpinnerModel** (`components/spinner.go`): Thin wrapper around bubbles/spinner with theme styling.
- **SiteCreate refactor**: Replaced manual step enum with FormModel, reducing code complexity.
- **Screen wiring**: PHP (version picker + remove confirm), Database (create/import forms + drop confirm), SSL (obtain form + revoke confirm), Backup (create form + delete/restore confirm).
- **App integration**: Toast for all OpDone/OpErr results, spinner stop on data refresh.

## Key Decisions

1. Custom key handling over `huh` library - matches existing sitecreate.go pattern
2. Inline form overlays instead of separate screen states
3. Messages carry form data (e.g., `InstallPHPMsg{Version}`) - clean separation
4. Toast lives in App (single instance), spinners live in individual screens

## Issues Found & Fixed

- **Data race in TEA**: State mutations inside `tea.Cmd` closures run on goroutines. Fixed by moving all mutations to synchronous `Update()` body.
- **Cursor capture by reference**: Database import closure captured `d.cursor` by reference. Fixed by capturing `dbName` value before closure.
- **Style allocation per frame**: `lipgloss.NewStyle()` called every render in confirm View. Moved to package-level var.
- **Toast stale dismiss**: Without ID matching, old tick could dismiss new toast. Added incrementing ID.
- **FormModel Active() after submit**: Never returned false. Fixed by advancing step past field count.

## Impact

- All destructive TUI actions now require confirmation
- Long-running operations show progress spinners
- Success/error feedback via auto-dismissing toasts
- 28 new unit tests (13 form + 9 confirm + 6 toast)
- ~1000 LOC added across 7 new files + 7 modified files
