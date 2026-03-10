# Code Review: Cache Management Feature

## Scope
- Files reviewed: 9 (5 new, 4 edited)
- New: `internal/cache/manager.go`, `internal/cache/manager_test.go`, `cmd/juiscript/cmd-cache.go`, `internal/tui/screens/cache.go`, `internal/tui/app_handlers_cache.go`
- Edited: `cmd/juiscript/main.go`, `internal/tui/app.go`, `internal/tui/app_messages.go`, `internal/tui/screens/dashboard.go`
- Build: PASS (`go build ./...`)
- Vet: PASS (`go vet ./...`)
- Tests: PASS (14/14)
- Updated plans: none (no plan file provided)

## Overall Assessment

Solid, consistent implementation. Follows the established firewall manager pattern (exec-based, no external library deps). Test coverage is good and all 14 tests pass. One security issue (unvalidated string injected into systemctl command), one logic issue (TUI DB range hardcoded vs config-driven), one dead-code issue (DisableRedis no-op with no comment clarity for callers).

---

## Critical Issues

None that would block deployment, but the following is a security concern that should be addressed before any production use.

---

## High Priority

### 1. Unvalidated `phpVersion` passed to systemctl — command injection risk

**File:** `internal/cache/manager.go:108`

```go
svc := fmt.Sprintf("php%s-fpm", phpVersion)
_, err := m.executor.Run(ctx, "systemctl", "restart", svc)
```

`phpVersion` is passed verbatim from user input (TUI text buffer, CLI flag) into the service name. Although `system.Executor.Run` uses `exec.Command` (not a shell), a crafted version like `../../bin/sh` would still be passed as a systemctl argument and systemctl would attempt to restart the unit `php../../bin/sh-fpm` — not exploitable for RCE, but could panic or produce confusing errors. More importantly, **the firewall manager validates all user-provided strings with regex before use** (e.g., `jailNameRe`); the cache manager skips this for `phpVersion`.

**Fix:** Add a version format validator consistent with the firewall pattern:

```go
// phpVersionRe matches valid PHP version strings: 8.0, 8.1, 8.2, 8.3, etc.
var phpVersionRe = regexp.MustCompile(`^\d+\.\d+$`)

func (m *Manager) ResetOpcache(ctx context.Context, phpVersion string) error {
    if phpVersion == "" {
        return fmt.Errorf("php version required")
    }
    if !phpVersionRe.MatchString(phpVersion) {
        return fmt.Errorf("invalid php version %q (expected format: 8.3)", phpVersion)
    }
    // ...
}
```

---

## Medium Priority

### 2. TUI flush-db validation hardcodes `0-15`, ignores config

**File:** `internal/tui/screens/cache.go:131`

```go
if err != nil || db < 0 || db > 15 {
```

The manager's `validateDB` respects `cfg.Redis.MaxDatabases`; the TUI hard-codes `15`. If the operator sets `max_databases = 4` in config, the TUI allows submitting `db=10` which the manager then rejects with a confusing error. The screen has no access to config, but the upper bound should at minimum be `> 15` (i.e., `>= 16`) to match the default, not just a duplicated magic number.

The real fix is to pass `maxDB int` into `NewCacheScreen` or to rely entirely on the manager's validation (just `db < 0` check in the screen, let the manager surface the range error). Given the current architecture, the simplest KISS fix:

```go
// In submitInput, only guard against obviously invalid input; let manager validate range
if err != nil || db < 0 {
    s.err = fmt.Errorf("invalid database number: %s", buf)
    return s, nil
}
```

### 3. `DisableRedis` is a permanent no-op with a success response

**File:** `internal/cache/manager.go:78`, `cmd/juiscript/cmd-cache.go:72-74`

```go
func (m *Manager) DisableRedis(ctx context.Context, domain string) error {
    return nil
}
```

The CLI prints `"Redis disabled for %s"` on success, but nothing actually happens. The comment says "Config-level disable handled by caller" but no caller does anything either. This is misleading to the operator.

**Fix:** Either remove the CLI subcommand for MVP, or print a warning that this is a no-op:

```go
fmt.Fprintf(os.Stderr, "Note: disable-redis is not implemented in this version. Remove Redis config manually.\n")
```

---

## Low Priority

### 4. `fetchCacheStatus` never auto-refreshes after `CacheOpDoneMsg`

**File:** `internal/tui/app.go:681-684`

```go
case CacheOpDoneMsg:
    toastCmd := a.toast.Show(components.ToastSuccess, "Cache operation completed")
    return a, tea.Batch(toastCmd, a.fetchCacheStatus())
```

This is correct — status is re-fetched after each operation. No issue.

### 5. `DisableRedis` handler in TUI (`app_handlers_cache.go`) is absent

No TUI handler exists for `DisableRedis` — consistent with the no-op CLI, no screen exposes this action. Intentional and fine for MVP.

---

## Positive Observations

- Manager pattern is identical to firewall manager: exec-based, nil-safe, no external deps.
- Error wrapping with context is consistent throughout (`"flush redis db %d: %w"`, `"restart %s: %w"`).
- `--force` guard on `flush --all` CLI is correctly implemented.
- TUI confirm dialog for `FlushAll` (`F` key) works correctly before dispatching `FlushRedisAllMsg`.
- All handlers nil-check `cacheMgr` before use.
- `parseRedisField` correctly handles Redis `INFO` CRLF line endings via `strings.TrimSpace`.
- `AppDeps` + nil-graceful-degradation pattern is followed consistently.
- 14 tests cover all code paths including boundary conditions and failure cases.

---

## Recommended Actions

1. **[High]** Add `phpVersionRe` validation to `ResetOpcache` in `manager.go` — mirrors firewall pattern, prevents bad systemctl calls.
2. **[Medium]** Fix TUI `flush-db` input validation to not hardcode `15` — either pass `maxDB` through or drop the upper-bound check and let the manager handle it.
3. **[Medium]** Either remove `disable-redis` CLI subcommand or emit a clear no-op warning so operators are not misled.

---

## Metrics
- Type Coverage: full (no `interface{}`, all typed)
- Test Coverage: 14 tests, all pass
- Build: clean
- Vet: clean
- Linting Issues: 0 blocking
