# Code Review Summary

## Scope
- Files reviewed: 8 files across `internal/config`, `internal/site`, `internal/backup`, `internal/system`
- Lines of code analyzed: ~1,200 (production + test)
- Review focus: Step 2 of "Test Coverage Boost" plan
- Updated plans: `plans/260310-1508-test-coverage-boost/plan.md`

## Overall Assessment

Solid implementation. All 4 packages pass tests (0 failures). Mock infrastructure follows the established project pattern. Test quality is high — table-driven where appropriate, rollback scenarios covered, no TODOs or commented-out tests.

## Critical Issues

None.

## High Priority Findings

**1. `EnsureDirs` ignores `Config.SitesPath()` — uses package-level `SitesPath()` instead**

`/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/config/config.go:155`

```go
dirs := []string{
    DefaultConfigDir,
    SitesPath(),          // BUG: ignores cfg.General.SitesDir override
    cfg.General.BackupDir,
}
```

Should be `cfg.SitesPath()`. This means `EnsureDirs` will always use `/etc/juiscript/sites` regardless of config, defeating the testability improvement from adding the method. No test covers `EnsureDirs`.

**2. Duplicate mock code across every test package (pre-existing, now expanded)**

`mockExecutor` and `mockFileManager` are now duplicated in at least: `nginx`, `database`, `php`, `ssl`, `service`, `supervisor`, `cache`, `firewall`, `site`, `backup`. Plan notes this is intentional per project convention, but the `backup/manager_test.go` variant adds an `onRun` hook not present in others — making the implementations diverge. Low DRY violation impact since test files, but increases maintenance cost.

## Medium Priority Improvements

**3. `TestLookupUID_CurrentUser` uses `os.Getenv("USER")` — fragile on CI**

`/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/system/usermgmt_test.go:93`

`USER` env is not guaranteed on all CI systems. Should use `os.CurrentUser()` or `os/user.Current()` which is the actual implementation path.

**4. `TestCreate_FilesOnly_Success` uses `os.WriteFile`/`os.MkdirAll` without checking errors**

`/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/backup/manager_test.go:605,619`

```go
os.MkdirAll(siteDir, 0750)
os.WriteFile(filepath.Join(sitesDir, "example.com.toml"), ...)
```

Missing `if err != nil { t.Fatal(...) }`. Silent test setup failures will cause confusing downstream errors. Same pattern repeated in `TestRestore_FilesOnly_Success`.

**5. `mockExecutor.failOn` keyed by command name, not full command**

In both `site/manager_test.go` and `backup/manager_test.go`, `failOn["systemctl"]` fails ALL `systemctl` calls, including `nginx -t` via chained calls. This is a design limitation — can cause false negatives if multiple `systemctl` calls exist in one flow and only one should fail. Current tests work because each test is isolated, but it's a footgun for future tests.

## Low Priority Suggestions

**6. `TestRun_DefaultTimeout` is a tautology**

`/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/system/executor_test.go:61`

Tests that `echo test` succeeds, which doesn't validate the default timeout behavior. To test the timeout is applied, would need a slow command + verify it succeeds within expected time bound. Minor — the test does add coverage.

**7. `createTestBackupFiles` creates files with 1-hour intervals**

`/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/backup/manager_test.go:115`

`time.Duration(-i) * time.Hour` — `backupFilename` truncates to seconds precision, so two files created within the same second (i=0 and i=0... but i starts at 0 and increments) — actually fine since `i` starts at 0 and the interval is 1 hour. No issue.

## Positive Observations

- Rollback tests (`TestCreate_RollbackOn*`) are thorough and test 4 distinct failure points.
- Path traversal protection is properly tested in both `backup/manager_test.go` (`TestIsInsideBackupDir`, `TestDelete_OutsideBackupDir`) and `TestValidateDomain`.
- `TestCronScheduleValidation` includes newline injection test case — good security coverage.
- `TestMetadata_Roundtrip` uses `Truncate(time.Second)` to handle TOML time serialization precision — correct.
- `onRun` hook in `backup/mockExecutor` is an elegant solution for simulating `tar` file creation without executing real tar.
- All tests use `t.TempDir()` — no leftover temp files, no OS-path hardcoding in assertions.

## Recommended Actions

1. **[High]** Fix `EnsureDirs` to use `cfg.SitesPath()` instead of package-level `SitesPath()` — 1-line fix.
2. **[Medium]** Add `t.Fatal` / `if err != nil` checks in test setup `os.WriteFile`/`os.MkdirAll` calls in `backup/manager_test.go`.
3. **[Medium]** Replace `os.Getenv("USER")` with `user.Current().Username` in `usermgmt_test.go`.
4. **[Low]** Either accept `failOn` keyed by binary name as a documented limitation, or enhance to accept full-command keys.

## Metrics

- Type Coverage: N/A (Go)
- Test Coverage: All 4 packages pass; coverage numbers not re-measured (plan target >70%)
- Linting Issues: 0 critical; 2 medium (unchecked errors in test setup)
- Build: `go test ./internal/...` — all pass

---

## Verification

```
ok  github.com/jhin1m/juiscript/internal/config   0.390s
ok  github.com/jhin1m/juiscript/internal/site     0.908s
ok  github.com/jhin1m/juiscript/internal/backup   1.175s
ok  github.com/jhin1m/juiscript/internal/system   0.648s
```

---

## Unresolved Questions

None. Plan unresolved question #1 ("`config.SitesPath()` global path") was resolved by adding `Config.SitesPath()` method — but `EnsureDirs` still uses the package-level version (finding #1 above).
