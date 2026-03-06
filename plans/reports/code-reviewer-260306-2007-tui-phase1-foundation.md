# Code Review Summary

## Scope
- Files reviewed: `internal/tui/app_messages.go` (NEW), `internal/tui/app.go`, `cmd/juiscript/main.go`
- Lines of code analyzed: ~350
- Review focus: Phase 1 Foundation — manager injection, message types, main wiring
- Updated plans: `plans/260306-1949-tui-backend-wiring/phase-01-foundation.md`

## Overall Assessment

Phase 1 is structurally sound. Build passes (`go build ./...`), `go vet` reports zero issues, existing tests unaffected. The `AppDeps` struct pattern is idiomatic and the nil-safety approach is reasonable. A few medium-severity issues require attention before Phase 2 begins; no critical/blocking security vulnerabilities.

---

## Critical Issues

None.

---

## High Priority Findings

### H1 — Data race on `a.installSummary` (app.go:319-336)

The goroutine spawned for `RunSetupMsg` writes `a.installSummary` without synchronization:

```go
go func() {
    summary, _ := a.prov.InstallSelected(...)
    a.installSummary = summary   // write from goroutine
    close(ch)
}()
```

The Bubble Tea `Update` loop reads `a.installSummary` (line 335) when `SetupDoneMsg` arrives. In practice the channel close happens-before the `SetupDoneMsg` read (Go memory model: channel send/close synchronizes with receive), so the race is technically safe *here*, but `go race` will flag it because the happens-before chain goes through a channel receive inside a `tea.Cmd` closure, not directly through the struct field assignment. Run `go test -race` (or `go run -race`) to confirm. **Fix:** assign `summary` as a return value delivered via the message itself rather than via a shared field:

```go
// In RunSetupMsg handler — deliver summary through the channel instead
go func() {
    summary, _ := a.prov.InstallSelected(...)
    ch <- provisioner.ProgressEvent{} // sentinel (or use a separate done channel)
    // Return summary in SetupDoneMsg directly
    close(ch)
}()
```

Or simpler: make `waitForProgress` return a `SetupDoneMsg{Summary: summary}` by adding summary to the channel type / using a dedicated done-channel. This also removes the `installSummary` field from `App` entirely (YAGNI win).

### H2 — `tplEngine` error silently discarded (main.go:76)

```go
tplEngine, _ := template.New()
```

If template loading fails (missing embedded files, parse errors) every downstream manager (`phpMgr`, `nginxMgr`, `siteMgr`, `supervisorMgr`) receives a broken engine and will produce subtle runtime failures with no diagnostic at startup. At minimum log the error:

```go
tplEngine, err := template.New()
if err != nil {
    logger.Error("failed to init template engine", "error", err)
    // either os.Exit(1) or proceed with nil guard in managers
}
```

---

## Medium Priority Improvements

### M1 — `NginxTestOkMsg` lost its `Output string` field (app_messages.go:47)

The plan spec says:
```go
type NginxTestOkMsg struct { Output string }
```

Implemented as:
```go
type NginxTestOkMsg struct{}
```

The output string is needed to display nginx test results to the user. Phases 2+ will implement `TestNginxMsg` handler and will need this field. Fix now before phases 2-5 depend on the type.

### M2 — `ServiceOpDoneMsg` defined in plan but missing from `app_messages.go`

The plan spec lists `ServiceOpDoneMsg struct{}` alongside `ServiceOpErrMsg`. Only `ServiceOpErrMsg` was added (line 81). `ServiceOpDoneMsg` is referenced nowhere yet, but its absence means Phase 4 (services wiring) will add it ad-hoc, risking inconsistency. Add it now alongside the other `*OpDoneMsg` types.

### M3 — Inconsistent nil-safety: `fetchServiceStatus` returns error cmd, others return nil (app.go:182-207)

```go
// svcMgr nil → returns error cmd (shows in serviceBar)
if a.svcMgr == nil {
    return func() tea.Msg { return ServiceStatusErrMsg{...} }
}

// phpMgr nil → returns nil (silent no-op)
if a.phpMgr == nil {
    return nil
}
```

Decide one convention. Returning `nil` is fine for optional managers; returning a synthetic error cmd for `svcMgr` will show a spurious "service manager not available" in the service bar when running without privileges. Prefer `nil` for all optional managers and let the UI show a "not available" state only when the user navigates to that screen.

### M4 — `goBack()` ignores `a.previous` field (app.go:481-489)

`a.previous` is set on navigation but `goBack()` uses a hardcoded switch instead of `a.previous`. This means navigating Dashboard → Nginx → (future sub-screen) → back goes to Dashboard instead of Nginx. The `previous` field is dead code right now. Either use it or remove it (YAGNI).

### M5 — All `TODO:` stub handlers call `a.fetchServiceStatus()` even for unrelated domains (app.go:431-444)

Service screen actions correctly re-fetch service status. But the pattern will be misapplied if copy-pasted for site/nginx/database handlers in phases 2-5. Not a bug now, but a maintenance trap. The comment in phase 2-5 plans should note: only service-action handlers should chain `fetchServiceStatus()`.

---

## Low Priority Suggestions

### L1 — Log file path hardcoded (main.go:52)

`/var/log/juiscript.log` will fail on non-root dev machines silently (fallback to discard). Consider `os.UserCacheDir()` or `~/.local/share/juiscript/juiscript.log` as primary with `/var/log` as fallback.

### L2 — `screenTitle()` reverse map lookup is O(n) on every render (app.go:597-602)

```go
for name, screen := range screenNames { if screen == a.current { return name } }
```

A second `map[Screen]string` would be O(1). Minor since the map has ~9 entries, but the pattern should not be copied for larger maps.

### L3 — Formatting inconsistency in App struct (app.go:106-114)

Mixed alignment styles between existing fields (single-space) and new fields (tab-aligned column). `gofmt` normalizes this but it reads inconsistently in diffs. No action needed — `gofmt` handles it.

---

## Positive Observations

- `AppDeps` struct pattern avoids a 10-parameter constructor — correct choice.
- Nil checks on all manager accessors before use — good defensive pattern.
- `waitForProgress` channel-based streaming is clean and idiomatic Bubble Tea.
- `go build ./...` and `go vet ./...` pass with zero output.
- `DetectPackagesMsg` logic (PHP treated as a group rather than per-version) is correct domain logic.
- Log-to-file pattern correctly avoids corrupting the alt-screen TUI.

---

## Recommended Actions

1. **(H1)** Eliminate `installSummary` field — pass summary through channel/message to remove the race window.
2. **(H2)** Log `template.New()` error; don't silently discard.
3. **(M1)** Restore `Output string` field to `NginxTestOkMsg` before Phase 2 implements the handler.
4. **(M2)** Add `ServiceOpDoneMsg struct{}` to `app_messages.go`.
5. **(M4)** Either use `a.previous` in `goBack()` or delete the field.
6. **(M3)** Standardize nil-manager convention (prefer `nil` return, not synthetic error).

---

## Metrics

- Type Coverage: N/A (no generics/interfaces changed)
- Test Coverage: No TUI unit tests exist; not regressions introduced
- Linting Issues: 0 (`go vet` clean, `go build` clean)
- Blocking issues: 0
- Notable (fix before next phase): H1, M1, M2

---

## Unresolved Questions

- Should `tplEngine` failure be fatal at startup or degrade gracefully? Depends on whether all managers handle a nil/broken engine safely.
- Is `a.previous` intentionally kept for future multi-level back navigation (e.g. Dashboard → Sites → Detail → back → Sites)? If yes, `goBack()` must be updated in Phase 2.
