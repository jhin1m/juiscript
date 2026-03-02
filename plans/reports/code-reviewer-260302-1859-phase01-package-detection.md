# Code Review: Phase 01 Step 2 — Package Detection (provisioner/detector.go)

## Scope
- Files reviewed: `internal/provisioner/detector.go`, `internal/provisioner/detector_test.go`
- Reference: `internal/service/manager.go`, `internal/php/manager.go`
- LOC: ~148 (impl) + ~269 (test)
- Review focus: security, performance, architecture, YAGNI/KISS/DRY, error handling, test coverage

---

## Overall Assessment

Code is clean, well-structured, and follows existing patterns. No critical security issues. One medium DRY violation and one logic bug warrant attention before merge.

---

## Critical Issues

None.

---

## High Priority Findings

### 1. Logic Bug: PHP entries always report `Installed: true` regardless of dpkg-query result

**File:** `detector.go:63–70`

```go
for _, ver := range phpVersions {
    pkg := "php" + ver + "-fpm"
    _, version := d.isInstalled(ctx, pkg)   // return value `installed` is discarded
    results = append(results, PackageInfo{
        ...
        Installed: true,   // hardcoded — never false even if dpkg-query returns not-installed
        Version:  version,
    })
}
```

`/etc/php/<ver>/` existing does not guarantee `php<ver>-fpm` is installed (package could be removed without purging the config dir). The `installed` bool returned by `isInstalled` is silently dropped.

**Fix:**

```go
installed, version := d.isInstalled(ctx, pkg)
results = append(results, PackageInfo{
    Name:        "php",
    DisplayName: "PHP " + ver,
    Package:     pkg,
    Installed:   installed,
    Version:     version,
})
```

**Impact:** TUI will show PHP as installed when it is not, causing silent provisioning errors downstream.

---

## Medium Priority Improvements

### 2. DRY Violation: `isVersionDir` duplicated across three packages

`isVersionDir` is identically implemented in:
- `internal/provisioner/detector.go:131–147`
- `internal/php/manager.go:222–238`

`internal/service/manager.go` uses a slightly different form via `isNumeric`.

All three do the same thing. This violates DRY and means any future change (e.g. supporting `X.Y.Z` versions) must be made in three places.

**Recommendation:** Extract to `internal/system/version.go` or a shared `internal/lemp/version.go` helper and import from all three packages. This is a refactor opportunity, not a blocker for merge.

---

## Low Priority Suggestions

### 3. `dpkgCmd` helper in test leaks format detail

`detector_test.go:46` builds the mock key including the literal `\n`:

```go
func dpkgCmd(pkg string) string {
    return "dpkg-query -W --showformat=${Status}\n${Version} " + pkg
}
```

The `\n` is a literal newline in the Go string. The `Run` mock joins args with `strings.Join(args, " ")` so the key would contain a real newline. This works because both the mock setup and lookup use the same helper, but it is fragile — any whitespace change in the `--showformat` arg would silently break all tests without a clear failure message. Low risk in practice since both sides use `dpkgCmd`, but worth noting.

### 4. No context cancellation test

No test verifies behavior when `ctx` is already cancelled. The existing mock ignores the context entirely (`_ context.Context`). Not a blocker given the executor interface contract, but worth adding one table-driven case if the executor is ever replaced with a real implementation.

---

## Positive Observations

- `isInstalled` correctly uses `strings.SplitN(..., 2)` — no unbounded split on multi-line dpkg output.
- `staticPackages` as a package-level `var` (not hardcoded strings in loop body) is clean.
- Mock executor design mirrors `service/manager_test.go` pattern exactly — consistent.
- `DetectAll` always returns a non-nil result and `nil` error, simplifying caller logic.
- Test coverage is solid: installed, not-installed, bad status, partial install, display names, PHP placeholder, and `isVersionDir` edge cases all covered.
- No shell string interpolation — all args passed as separate variadic strings to `executor.Run`, preventing command injection.

---

## Recommended Actions

1. **[Fix before merge]** Use the `installed` bool returned by `isInstalled` in the PHP loop (`detector.go:63`).
2. **[Refactor, non-blocking]** Extract `isVersionDir`/`isNumeric` to a shared internal package to eliminate three-way duplication.
3. **[Optional]** Add a cancelled-context test case if real executor integration tests are planned.

---

## Metrics

- Linting issues: 0 expected (standard Go idioms used throughout)
- Test cases: 9 test functions, adequate coverage
- DRY violations: 1 (medium, non-blocking)
- Bugs: 1 (high — `Installed: true` hardcoded for PHP entries)

---

## Unresolved Questions

- Is `/etc/php/<ver>/` presence considered sufficient proof of PHP installation in the provisioner's domain, or should the dpkg state always be authoritative? (Determines whether the bug fix above is the correct policy or intentional.)
