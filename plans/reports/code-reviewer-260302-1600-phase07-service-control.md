# Code Review Report: Phase 07 Service Control

**Date:** 2026-03-02
**Reviewer:** code-reviewer subagent
**Build:** PASS | **Tests:** 16/16 PASS | **Vet:** CLEAN

---

## Scope

- Files reviewed: 5 changed files + 2 reference files
- `internal/service/manager.go` (259 lines)
- `internal/service/manager_test.go` (285 lines)
- `internal/tui/screens/services.go` (183 lines)
- `internal/tui/components/servicepanel.go` (59 lines)
- `internal/tui/app.go` (359 lines)
- Reference: `internal/php/manager.go`, `internal/tui/screens/php.go`

---

## Overall Assessment

Solid implementation. Security posture is good (no shell injection, whitelist enforced, exec.Run uses argv not shell). Consistent with existing patterns. Tests pass at 49.5% coverage. Two medium and two low issues found; no critical issues.

---

## Critical Issues

None.

---

## High Priority Findings

None.

---

## Medium Priority Improvements

### M1: PHP-FPM whitelist regex is too permissive

**File:** `internal/service/manager.go:44-46`

```go
return strings.HasPrefix(s, "php") && strings.HasSuffix(s, "-fpm")
```

This accepts `phpXXXXXXXX-fpm`, `php-fpm`, `php.evil-fpm`. The version segment is not validated. A crafted `ServiceName` (if one were somehow injected) could slip through.

The `detectPHPVersions()` already uses `isNumeric` validation. `PHPFPMService()` constructs the name from a version string, but `isAllowed` accepts any string matching the pattern without checking the middle part.

**Fix:** Add a format check consistent with the existing `isVersionDir` logic from `internal/php/manager.go`:

```go
func isAllowedPHPFPM(s string) bool {
    // Must be php{major}.{minor}-fpm
    s = strings.TrimPrefix(s, "php")
    s = strings.TrimSuffix(s, "-fpm")
    return isVersionDir(s) // reuse existing version validator
}
```

Note: `isVersionDir` is in `internal/php` package, so you'd either duplicate the logic or extract it to `internal/system`. Duplicating `isNumeric`-based check is acceptable given YAGNI — but at minimum tighten the middle segment check.

### M2: `HealthCheck` is a pass-through wrapper — YAGNI violation

**File:** `internal/service/manager.go:190-196`

```go
func (m *Manager) HealthCheck(ctx context.Context) ([]Status, error) {
    all, err := m.ListAll(ctx)
    if err != nil {
        return nil, err
    }
    return all, nil
}
```

This is identical to `ListAll`. No distinct health-check logic exists. It adds surface area without value. Either remove it or give it a meaningful body (e.g., return only critical services, or return an error when unhealthy).

---

## Low Priority Suggestions

### L1: `Status.Uptime` field is declared but never populated

**File:** `internal/service/manager.go:57`

```go
Uptime time.Duration // how long the service has been running
```

`parseStatus` never sets `Uptime`. `systemctl show` does not expose a direct uptime property — it would require `ActiveEnterTimestamp` and time arithmetic. Either remove the field (YAGNI) or populate it.

### L2: Test coverage at 49.5% — below 70% project standard

**File:** `internal/service/manager_test.go`

Missing test coverage for:
- `ListAll` (happy path + not-installed service fallback)
- `IsHealthy` (true/false scenarios)
- `HealthCheck`
- `detectPHPVersions` (mocking `/etc/php` is hard, but `ListAll` drives it)

The `code-standards.md` states 70%+ for critical packages. Service control is critical infrastructure. Add table-driven tests for `ListAll` using the mock executor with pre-seeded `Status` outputs.

---

## Positive Observations

- No shell injection: `executor.Run(ctx, "systemctl", action, string(name))` uses argv, not `sh -c`. Correct.
- Whitelist enforcement (`isAllowed`) is applied consistently across all four action methods and `Status`/`IsActive`.
- `isNumeric` guards `detectPHPVersions` against directory traversal names.
- TUI screen pattern (`services.go`) is consistent with `php.go` — same cursor/Update/View structure, same message dispatch style. Clean.
- `servicepanel.go` is minimal and single-purpose. Good KISS adherence.
- `app.go` integration is clean — service messages handled with same TODO-stub pattern as other screens, no regressions.
- All 16 tests pass, build and vet are clean.

---

## Recommended Actions

1. **[Medium]** Tighten `isAllowed` PHP-FPM check to validate version segment format (not just prefix/suffix).
2. **[Medium]** Remove or implement `HealthCheck` — currently dead wrapper over `ListAll`.
3. **[Low]** Remove `Uptime` from `Status` struct or implement `ActiveEnterTimestamp` parsing.
4. **[Low]** Add `ListAll`/`IsHealthy` tests to reach 70% coverage target.

---

## Metrics

- Build: PASS
- Vet: CLEAN
- Tests: 16/16 PASS
- Coverage (`internal/service`): 49.5% (below 70% standard)
- Linting Issues: 0 (vet clean)
- Critical Issues: 0
- High Issues: 0
- Medium Issues: 2
- Low Issues: 2
