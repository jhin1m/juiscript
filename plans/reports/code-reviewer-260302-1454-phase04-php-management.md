# Code Review: Phase 04 PHP Management

## Scope
- Files reviewed: `internal/php/manager.go`, `internal/php/pool.go`, `internal/php/manager_test.go`, `internal/tui/screens/php.go`, `internal/tui/app.go`, `internal/template/templates/php-fpm-pool.conf.tmpl`, `internal/template/engine_test.go`
- Lines analyzed: ~700
- Review focus: security, performance, architecture, YAGNI/KISS/DRY, error handling
- Build: PASS (`go build ./...`)
- Vet: PASS (`go vet ./...`)
- Tests: 17/17 PASS (`go test ./...`)
- Updated plans: `plans/260302-1356-juiscript-bootstrap/phase-04-php-management.md`

---

## Overall Assessment

Solid implementation. Security-critical path traversal is properly blocked. Architecture mirrors the nginx manager pattern cleanly. Tests cover rollback and edge cases. Two issues warrant attention before production use.

---

## Critical Issues

### 1. `poolConfigPath` does not validate `domain` — path traversal via domain string

**File:** `internal/php/pool.go:59-61`

```go
func poolConfigPath(version, domain string) string {
    return fmt.Sprintf("/etc/php/%s/fpm/pool.d/%s.conf", version, domain)
}
```

`version` is validated via `validateVersion`, but `domain` is **never validated** in `DeletePool` or `CreatePool` before constructing the path. A caller passing `domain = "../../../etc/cron.d/evil"` would write to `/etc/php/8.3/fpm/pool.d/../../../etc/cron.d/evil.conf`.

`SwitchVersion` and `CreatePool` both call `poolConfigPath(cfg.PHPVersion, cfg.SiteDomain)` without checking `SiteDomain`.

**Fix:** Add domain validation in `CreatePool` and `DeletePool`, reusing the nginx pattern:

```go
// in pool.go
func validateDomain(domain string) error {
    if domain == "" || strings.Contains(domain, "/") || strings.Contains(domain, "..") {
        return fmt.Errorf("invalid domain: %q", domain)
    }
    return nil
}

func (m *Manager) CreatePool(ctx context.Context, cfg PoolConfig) error {
    if err := validateVersion(cfg.PHPVersion); err != nil {
        return err
    }
    if err := validateDomain(cfg.SiteDomain); err != nil {
        return err
    }
    // ...
}

func (m *Manager) DeletePool(ctx context.Context, domain, version string) error {
    if err := validateVersion(version); err != nil {
        return err
    }
    if err := validateDomain(domain); err != nil {
        return err
    }
    // ...
}
```

---

### 2. `RemoveVersion` uses glob pattern `php8.3-*` in apt-get — potential unintended package removal

**File:** `internal/php/manager.go:137`

```go
_, err := m.executor.Run(removeCtx, "apt-get", "remove", "--purge", "-y",
    fmt.Sprintf("php%s-*", version))
```

`apt-get remove` does not expand shell globs — the `*` is passed literally to apt. On Debian/Ubuntu, `apt-get remove 'php8.3-*'` actually works because apt itself treats it as a glob, but this is apt-specific behavior and silently removes **all** matching packages including any custom `php8.3-*` packages. More importantly: the test `exec.hasCommand("apt-get remove --purge -y php8.2-*")` passes because the mock does string matching — this masks whether apt actually expands the glob in production.

**Risk:** Medium in practice (intentional behavior), but undocumented and not guarded. If `validateVersion` is somehow bypassed, the version string could be crafted to match unintended packages.

**Recommendation:** Document the apt glob behavior explicitly in the function comment. The `validateVersion` guard is sufficient mitigation, but add a note:

```go
// apt-get treats "php8.3-*" as a glob to remove all php8.3 packages.
// validateVersion above ensures version is numeric-only (e.g. "8.3").
```

---

## High Priority Findings

### 3. `SwitchVersion` rollback gap: DeletePool failure returns error but state is inconsistent

**File:** `internal/php/pool.go:172-175`

```go
// 3. Delete old pool (new one is already serving)
if err := m.DeletePool(ctx, cfg.SiteDomain, fromVersion); err != nil {
    // Non-fatal: log but continue (new pool is working)
    return fmt.Errorf("warning: failed to delete old pool for php%s: %w", fromVersion, err)
}
```

The comment says "non-fatal: log but continue" but the code **does return an error** — the caller sees a failure even though the switch succeeded. This is ambiguous: callers can't distinguish "new pool working, old orphaned" from "catastrophic failure". The `warning:` prefix in the error string is a code smell — error types exist for this.

Options:
- Return `nil` and log (if truly non-fatal)
- Return a typed sentinel or wrapped warning error
- At minimum: update comment to match code behavior

Current behavior causes `TestSwitchVersion_Success` to verify the old pool was removed, but if `Remove` fails in production the caller gets an error despite the switch succeeding.

### 4. `isVersionDir` allows empty parts — `"."` passes validation

**File:** `internal/php/manager.go:222-235`

```go
func isVersionDir(name string) bool {
    parts := strings.Split(name, ".")
    if len(parts) != 2 {
        return false
    }
    for _, p := range parts {
        for _, c := range p {
            if c < '0' || c > '9' {
                return false
            }
        }
    }
    return true
}
```

`strings.Split(".", ".")` → `["", ""]` — both parts are empty strings, so the inner loop never runs and `isVersionDir(".")` returns `true`. The test covers `".."` (2 dots → 3 parts, rejected) but not `"."`.

**Fix:**

```go
for _, p := range parts {
    if len(p) == 0 {
        return false
    }
    for _, c := range p {
        // ...
    }
}
```

Add test case `{".", false}` to `TestIsVersionDir`.

### 5. `ListVersions` hardcodes `/etc/php` — not injectable, untestable in isolation

**File:** `internal/php/manager.go:148`

```go
entries, err := os.ReadDir("/etc/php")
```

Every other path is configurable or mockable, but this is hardcoded. `TestListVersions_ScansDirectory` acknowledges this limitation with a comment: "We can't easily test ListVersions since it reads /etc/php directly." The test falls back to only testing the helper.

The nginx manager avoids this by accepting `sitesAvailable`/`sitesEnabled` in `NewManager`. PHP manager should accept a `phpBaseDir string` param (default `/etc/php`).

This is a medium concern now but becomes critical when integration testing or running on non-standard setups.

---

## Medium Priority Improvements

### 6. Dual default-value logic in `DefaultPool` and `CreatePool`

**File:** `internal/php/pool.go:41-56` and `64-96`

`DefaultPool` sets defaults; `CreatePool` also sets the same defaults for zero-values. This violates DRY and creates two sources of truth. If a default changes, it must be updated in two places.

**Fix:** Either remove the zero-value guards in `CreatePool` (assuming callers always use `DefaultPool`), or remove `DefaultPool` and only apply defaults in `CreatePool`. Given callers use `DefaultPool`, the zero-value guards in `CreatePool` are defensive but redundant in practice.

### 7. `poolTemplateData` internal struct duplicates `PoolConfig` fields

**File:** `internal/php/pool.go:26-38`

`poolTemplateData` is a near-identical copy of `PoolConfig` with renamed fields. The mapping at lines 99-111 is boilerplate. Since the template is internal to this package, `PoolConfig` could be used directly in the template (renaming fields to match), or the template field names could match `PoolConfig`.

Minor YAGNI/DRY issue — acceptable if field name divergence is intentional for template readability.

### 8. `apt-get update` context missing timeout

**File:** `internal/php/manager.go:64`

```go
_, err = m.executor.Run(ctx, "apt-get", "update", "-y")
```

`InstallVersion` applies a 5-minute timeout to `apt-get install` but passes the parent `ctx` (potentially unbounded) to `apt-get update`. The executor does add a 30s default if no deadline is set, so this is minor, but consistency would be better.

### 9. TUI PHP screen: `rows` string built with `+=` in loop

**File:** `internal/tui/screens/php.go:93-124`

```go
var rows string
for i, v := range p.versions {
    // ...
    rows += row + "\n"
}
```

Minor: use `strings.Builder` or `lipgloss.JoinVertical` for consistency with the surrounding style. For typical PHP version counts (< 10) this is negligible.

### 10. `app.go` PHP message handlers are stubs — no PHP manager wired

**File:** `internal/tui/app.go:152-158`

```go
case screens.InstallPHPMsg:
    // TODO: Call PHP manager to install version
    return a, nil

case screens.RemovePHPMsg:
    // TODO: Call PHP manager to remove version
    return a, nil
```

Expected for a phased implementation, but `App` holds no reference to `php.Manager`. The PHP screen is wired at the TUI level but the backend is disconnected. This is acceptable for Phase 04 scaffolding but must be addressed before this is functional.

### 11. `goBack()` in `app.go` does not handle `ScreenPHP`

**File:** `internal/tui/app.go:165-173`

```go
func (a *App) goBack() *App {
    switch a.current {
    case ScreenSiteCreate, ScreenSiteDetail:
        a.current = ScreenSites
    default:
        a.current = ScreenDashboard
    }
    return a
}
```

PHP screen's `esc` key sends `GoBackMsg`, which calls `goBack()`. The default case sends to `ScreenDashboard` which is correct — but if future sub-screens are added under PHP (e.g., version detail), `previous` tracking won't work properly. Low risk now; noted for future.

---

## Low Priority Suggestions

### 12. Template writes config with `0644` perms — consider `0640`

**File:** `internal/php/pool.go:120`

```go
if err := m.files.WriteAtomic(path, []byte(rendered), 0644); err != nil {
```

Pool config at `/etc/php/8.3/fpm/pool.d/` contains `open_basedir` paths and user/socket info. `0644` is world-readable. `0640` with root:www-data ownership would be more secure. Low risk since config values aren't secrets, but defense-in-depth.

### 13. `enable` systemctl call ignores error

**File:** `internal/php/manager.go:107`

```go
_, _ = m.executor.Run(ctx, "systemctl", "enable", svc)
_, err = m.executor.Run(ctx, "systemctl", "start", svc)
```

`enable` failure is silently ignored; `start` failure is reported. This means a system restart won't auto-start PHP-FPM without any user notification. Consider at least logging the enable failure.

### 14. Missing `isVersionDir` empty-string check in `TestIsVersionDir`

**File:** `internal/php/manager_test.go:141-155`

Test covers `""` → `false` (passes via `len(parts) != 2` since `Split("", ".")` → `[""]`), but the `"."` case (issue #4) is missing.

---

## Positive Observations

- `validateVersion` correctly blocks path traversal, shell metacharacters, and three-part versions — tested with `"../etc"` and `"8.3;rm -rf /"`.
- `exec.CommandContext` used throughout — no `exec.Command`, proper context propagation.
- Executor pattern: `exec.CommandContext` with separate args (never shell string interpolation) — no command injection possible.
- `SwitchVersion` zero-downtime strategy is correct: create new pool → reload nginx → delete old pool.
- Nginx rollback on failure in `SwitchVersion` is properly implemented and tested.
- Template uses `php_admin_value` (not `php_value`) — prevents `.user.ini` override.
- `security.limit_extensions = .php` and `open_basedir` present — good hardening.
- `listen.mode = 0660` with `listen.owner = www-data` — socket permissions correct.
- `WriteAtomic` used for all config writes — no partial-write corruption risk.
- 17 tests with good coverage of validation, defaults, rollback, and FPM service interactions.
- Architecture matches nginx manager pattern consistently.
- YAGNI observed: no over-engineered version registry, no premature abstraction.

---

## Recommended Actions

1. **[Must Fix]** Add `validateDomain` in `CreatePool` and `DeletePool` — path traversal vulnerability (#1)
2. **[Must Fix]** Fix `isVersionDir(".")` returns `true` — add empty-part check (#4)
3. **[Should Fix]** Clarify `SwitchVersion` step-3 error semantics: return nil or typed warning, not `fmt.Errorf("warning: ...")` (#3)
4. **[Should Fix]** Inject `phpBaseDir` into `NewManager` to make `ListVersions` testable (#5)
5. **[Should Fix]** Document apt glob behavior for `php8.3-*` removal (#2)
6. **[Minor]** Consolidate duplicate defaults between `DefaultPool` and `CreatePool` (#6)
7. **[Minor]** Consider `0640` perms for pool configs (#12)
8. **[Future]** Wire `php.Manager` into `App` for `InstallPHPMsg`/`RemovePHPMsg` handlers (#10)

---

## Metrics
- Type coverage: Full — all public APIs typed, no `interface{}` usage
- Test coverage: 17 tests, good path coverage; `ListVersions` integration untestable (issue #5)
- Linting issues: 0 (`go vet` clean)
- Build: Clean

---

## Unresolved Questions
- Should `SwitchVersion` step 3 (old pool deletion failure) be non-fatal with logging, or return a typed sentinel error? The current `fmt.Errorf("warning: ...")` string is not machine-parseable.
- Is `0644` for pool config files an intentional policy decision or an oversight?
- Will `php.Manager` be injected into `App` in Phase 04 or deferred to a later integration phase?
