# Debugger Report: PHP Installation Hang & Version List Empty

Date: 2026-03-04
Slug: php-bugs

---

## Executive Summary

Two separate bugs found in the PHP installation and display flow:

1. **Bug 1 (Install Hang)**: The goroutine spawned for `InstallSelected` discards the `*InstallSummary` return value, so `SetupDoneMsg{}` is sent with a nil `Summary`. More critically, the TUI **stops listening** for progress events the moment the channel is drained but the goroutine is still running a long `apt-get` subprocess. The real hang comes from `php.Manager.InstallVersion` calling `EnsurePPA` (apt-get update) inside a 5-minute timeout, blocking the goroutine silently while the TUI has no pending `waitForProgress` cmd queued — the spinner stops ticking and no output appears.

2. **Bug 2 (No PHP Versions Shown)**: `phpScreen.SetVersions()` is **never called anywhere in the codebase**. The PHP screen's `versions` slice is always `nil`/empty, so it always renders "No PHP versions installed" regardless of what is actually on disk.

---

## Bug 1: Script Hangs During Installation

### Root Cause Chain

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go`
**Lines 241–251** — `RunSetupMsg` handler:

```go
case screens.RunSetupMsg:
    ch := make(chan provisioner.ProgressEvent, 10)
    a.setupProgressCh = ch
    go func() {
        a.prov.InstallSelected(context.Background(), msg.Names, func(ev provisioner.ProgressEvent) {
            ch <- ev
        })
        close(ch)
    }()
    return a, waitForProgress(ch)
```

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go`
**Lines 253–256** — `SetupProgressMsg` handler:

```go
case screens.SetupProgressMsg:
    a.setupScreen.Update(msg)
    return a, waitForProgress(a.setupProgressCh)
```

The flow only keeps `waitForProgress` alive when it receives `SetupProgressMsg`. The hang window:

1. User confirms install → `RunSetupMsg` sent → `waitForProgress(ch)` queued.
2. `InstallSelected` goroutine starts → calls `AptUpdate` (apt-get update, can take 10–30s on a cold cache).
3. During `AptUpdate`, **no progress events are emitted** (the `emit()` calls only happen per-package, after update).
4. `waitForProgress` is blocking on the channel. If the channel buffer (size 10) fills during a later step, the goroutine itself blocks on `ch <- ev`.
5. The **spinner stops ticking** because `spinner.TickMsg` is only looped when `updateInstalling` returns a cmd, and `waitForProgress` is a separate cmd that returns nothing to drive the spinner until it fires.

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/provisioner/provisioner.go`
**Lines 59–99** — `InstallSelected`:

```go
func (p *Provisioner) InstallSelected(ctx context.Context, names []string, progressFn func(ProgressEvent)) (*InstallSummary, error) {
    ...
    if err := p.installer.AptUpdate(ctx); err != nil {  // line 70 — no progress event emitted here
        return nil, err
    }

    for _, name := range names {
        emit(ProgressEvent{PackageName: name, Status: ProgressStarting})  // line 81
        ...
    }
```

The `AptUpdate` step (line 70) runs with zero feedback to the channel. Combined with `EnsurePPA` in `php.Manager.InstallVersion` (which may also run `apt-get update` a second time), the goroutine can be silent for 30–60s.

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/php/manager.go`
**Lines 74–113** — `InstallVersion`:

```go
func (m *Manager) InstallVersion(ctx context.Context, version string) error {
    if err := m.EnsurePPA(ctx); err != nil {  // line 79 — may run apt-get update again
        return err
    }
    ...
    installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)  // line 91
    ...
    _, err := m.executor.Run(installCtx, "apt-get", args...)  // line 94 — blocks up to 5 min
```

`InstallPHP` in `installer.go` (line 99) calls `phpMgr.InstallVersion` directly, which adds another `EnsurePPA`/`apt-get update` on top of the one already run in `InstallSelected.AptUpdate`. For a second PHP version install, this double apt-get update (one in `AptUpdate`, one in `EnsurePPA`) adds silent blocking time.

**Secondary issue** — `SetupDoneMsg` carries nil Summary:

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go`, **line 249**:
```go
go func() {
    a.prov.InstallSelected(context.Background(), msg.Names, func(ev provisioner.ProgressEvent) {
        ch <- ev
    })
    close(ch)  // summary return value is DISCARDED
}()
```

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go`, **line 170**:
```go
if !ok {
    return screens.SetupDoneMsg{}  // Summary field is nil
}
```

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/setup.go`, **line 434–436**:
```go
case SetupDoneMsg:
    s.summary = msg.Summary  // msg.Summary is always nil
    s.state = stateDone
```

So the "done" screen will always show "0 installed, 0 skipped, 0 failed" even when packages were installed.

---

## Bug 2: "No PHP Versions Installed" Despite Versions Being Present

### Root Cause

`phpScreen.SetVersions()` is **never called** in `app.go`. The method exists at:

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/php.go`
**Lines 29–32**:
```go
func (p *PHPScreen) SetVersions(versions []php.VersionInfo) {
    p.versions = versions
    p.err = nil
}
```

But a search across the entire codebase shows **zero call sites** for `phpScreen.SetVersions`. The `phpScreen.versions` field is initialized as nil in `NewPHPScreen` and never populated.

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/screens/php.go`, **line 96**:
```go
if len(p.versions) == 0 {
    empty := p.theme.Subtitle.Render("  No PHP versions installed.")
```

This branch is always taken because `p.versions` is always `nil`.

The data source exists — `php.Manager.ListVersions()` at:

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/php/manager.go`, **lines 147–188** — correctly scans `/etc/php/` for version directories, checks for `fpm/pool.d`, and queries systemctl for status. It would return the correct list. But it is **never called from the TUI**.

The pattern used for other data (service status, package detection) is:
- `fetchServiceStatus()` → async cmd → `ServiceStatusMsg` → `serviceBar.SetServices()`
- `detectPackages()` → async cmd → `DetectPackagesMsg` → `setupScreen.SetPackages()`

The equivalent `fetchPHPVersions()` cmd + `PHPVersionsMsg` handler + `phpScreen.SetVersions()` call **does not exist**.

The PHP screen is also navigated to via `screens.NavigateMsg` (e.g. user selects PHP from dashboard), but on navigation the app only calls `a.fetchServiceStatus()` — never a PHP version fetch:

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go`, **lines 229–234**:
```go
case screens.NavigateMsg:
    if screen, ok := screenNames[msg.Screen]; ok {
        a.previous = a.current
        a.current = screen
    }
    return a, a.fetchServiceStatus()  // no PHP version fetch here
```

Similarly, after install completes (SetupDoneMsg), only `detectPackages` and `fetchServiceStatus` are called, not a PHP version refresh:

**File**: `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/app.go`, **lines 258–262**:
```go
case screens.SetupDoneMsg:
    a.setupScreen.Update(msg)
    a.setupProgressCh = nil
    return a, tea.Batch(a.detectPackages(), a.fetchServiceStatus())
    // missing: a.fetchPHPVersions()
```

---

## Affected Files Summary

| File | Lines | Issue |
|------|-------|-------|
| `internal/tui/app.go` | 241–251 | goroutine discards `InstallSummary` return value |
| `internal/tui/app.go` | 165–173 | `waitForProgress` sends `SetupDoneMsg{}` with nil Summary |
| `internal/tui/app.go` | 229–234 | Navigation to PHP screen doesn't trigger version fetch |
| `internal/tui/app.go` | 258–262 | `SetupDoneMsg` handler doesn't refresh PHP versions |
| `internal/tui/app.go` | 305–307 | `InstallPHPMsg` handler is TODO stub, returns `nil` |
| `internal/tui/screens/php.go` | 29–32 | `SetVersions()` exists but is never called |
| `internal/tui/screens/php.go` | 96–99 | Always shows "No PHP versions installed" due to nil slice |
| `internal/provisioner/provisioner.go` | 70 | `AptUpdate` emits no progress event (silent blocking) |
| `internal/php/manager.go` | 79 | `EnsurePPA` may run redundant `apt-get update` after provisioner already ran one |

---

## Recommended Fixes (High Level)

### Bug 1 — Install Hang / No Progress

1. Add a `ProgressAptUpdate` status and emit it from `InstallSelected` before calling `AptUpdate`.
2. Capture `InstallSummary` return value from `InstallSelected` goroutine and pass it into `SetupDoneMsg`.
3. Optionally: suppress redundant `EnsurePPA` call when called from provisioner context (or emit a "Checking PPA" progress event from within).

### Bug 2 — PHP Screen Empty

1. Add `fetchPHPVersions() tea.Cmd` in `app.go` that calls `php.Manager.ListVersions()` and returns a `PHPVersionsMsg`.
2. Add `PHPVersionsMsg` handler in `Update()` that calls `a.phpScreen.SetVersions(msg.Versions)`.
3. Call `a.fetchPHPVersions()` on: navigation to `ScreenPHP`, after `SetupDoneMsg`, after any `InstallPHPMsg`/`RemovePHPMsg` action.
4. Implement the `InstallPHPMsg` handler (currently a TODO at line 305–307) to actually trigger install and refresh.

---

## Unresolved Questions

- Is `php.Manager` available on the `App` struct (it is not currently stored there — only `prov *provisioner.Provisioner` is)? Need to either expose `phpMgr` on `App` or add a `ListPHPVersions` method to `Provisioner` that delegates to the detector/manager.
- Should `EnsurePPA` be called on every `InstallVersion` call, or only once per session? Currently called per-install, which causes double `apt-get update` when going through provisioner.
