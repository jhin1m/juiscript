# Code Review: Phase 09 - Backup System

**Date:** 2026-03-02
**Reviewer:** code-reviewer subagent

## Scope

- `internal/backup/manager.go` (~452 lines)
- `internal/backup/manager_test.go` (~135 lines)
- `internal/tui/screens/backup.go` (~136 lines)
- `internal/tui/app.go` (backup-related additions only)
- Reference: `internal/database/`, `internal/tui/screens/services.go`

---

## Critical Issues (MUST FIX)

### 1. `Delete()` has no path containment check — path traversal

`manager.go:345-350`:
```go
func (m *Manager) Delete(backupPath string) error {
    if _, err := os.Stat(backupPath); err != nil {
        return fmt.Errorf("backup not found: %w", err)
    }
    return os.Remove(backupPath)
}
```

`backupPath` is caller-supplied (comes from `DeleteBackupMsg.Path` in the TUI, which originates from `BackupInfo.Path`). No check that the path is inside `m.backupDir()`. A crafted path like `/etc/cron.d/juiscript-example.com` would pass the stat check and be deleted.

**Fix:**
```go
func (m *Manager) Delete(backupPath string) error {
    // Ensure path is within the configured backup directory
    abs, err := filepath.Abs(backupPath)
    if err != nil {
        return fmt.Errorf("resolve path: %w", err)
    }
    if !strings.HasPrefix(abs, m.backupDir()+string(filepath.Separator)) {
        return fmt.Errorf("backup path outside backup directory")
    }
    if _, err := os.Stat(abs); err != nil {
        return fmt.Errorf("backup not found: %w", err)
    }
    return os.Remove(abs)
}
```

### 2. `Restore()` — no path containment for `backupPath` parameter

`manager.go:226-230`:
```go
func (m *Manager) Restore(ctx context.Context, backupPath, domain string) error {
    if err := validateDomain(domain); err != nil {
        return err
    }
    if _, err := os.Stat(backupPath); err != nil { ...
```

`backupPath` is not validated to be inside `m.backupDir()`. An attacker (or bug) could supply `/proc/sysrq-trigger` or any world-readable path. The archive is then extracted via `tar -xzf` into a temp dir — that part is safe, but accepting arbitrary paths from the TUI message bus is not.

**Fix:** Apply same containment check as `Delete()` above.

### 3. `SetupCron()` — `schedule` is not validated, enables cron injection

`manager.go:380-396`:
```go
cronLine := fmt.Sprintf("%s root /usr/local/bin/juiscript backup create --domain %s --type full\n",
    schedule, domain)
```

`domain` is validated by `safeNameRegex`, but `schedule` (e.g., `"0 2 * * *"`) is written verbatim to `/etc/cron.d/`. A schedule containing a newline injects an extra cron line. A schedule with unquoted shell metacharacters or env-var assignments corrupts the cron file format silently.

`/etc/cron.d/` format is strict: `<m> <h> <dom> <mon> <dow> <user> <command>`. Any deviation can be exploited.

**Fix:**
```go
var safeCronScheduleRegex = regexp.MustCompile(`^(\*|[0-9,\-*/]+)\s+(\*|[0-9,\-*/]+)\s+(\*|[0-9,\-*/]+)\s+(\*|[0-9,\-*/]+)\s+(\*|[0-9,\-*/]+)$`)

func validateCronSchedule(schedule string) error {
    if !safeCronScheduleRegex.MatchString(schedule) {
        return fmt.Errorf("invalid cron schedule %q", schedule)
    }
    return nil
}
```

Call before building `cronLine`.

### 4. `Restore()` trusts `meta.Type` from the archive without sanitization

`manager.go:262`:
```go
backupType := BackupType(meta.Type)
```

`meta.Type` comes from `metadata.toml` inside the user-supplied archive. It's cast directly to `BackupType` and used in branch logic. An adversarially crafted archive with `type = "full"` when it only contains a database dump causes silent partial restore with no error. Not injection, but a correctness/integrity concern. Also enables confusion attacks.

**Fix:** Validate against the known set:
```go
switch BackupType(meta.Type) {
case BackupFull, BackupFiles, BackupDatabase:
    backupType = BackupType(meta.Type)
default:
    return fmt.Errorf("unknown backup type in metadata: %q", meta.Type)
}
```

---

## Warnings (Should Consider)

### 5. `backupDir()` not cleaned/absolute — `filepath.Abs` never called

`manager.go:79-81` returns `m.config.Backup.Dir` raw. If the config value is a relative path (e.g., `./backups`), all `filepath.Join(backupDir, ...)` calls produce relative paths. `os.MkdirAll` succeeds but `filepath.HasPrefix` containment checks (critical fix above) would be unreliable.

**Fix:** Normalize in `NewManager` or `backupDir()`:
```go
func (m *Manager) backupDir() string {
    abs, _ := filepath.Abs(m.config.Backup.Dir)
    return abs
}
```

### 6. `Create()` — temp dir created as `os.TempDir()` root, not under `backupDir`

`manager.go:152`: `os.MkdirTemp("", "juiscript-backup-*")` uses the system temp dir. If `backupDir` is on a different filesystem from `/tmp`, the final `tar -czf` writes the archive to `backupDir` (correct) but the intermediate temp files live in `/tmp`. This is fine functionally but means intermediate DB dumps (sensitive data) sit in `/tmp` with default permissions (`0700` for the dir, but contents may be `0600`). Another process running as root could access them.

**Low risk given root-only tool**, but worth noting for hardening.

### 7. `chown` in `Restore()` uses `s.User + ":" + s.User` — assumes group == user

`manager.go:277`:
```go
_, err = m.executor.Run(ctx, "chown", "-R", s.User+":"+s.User, siteDir)
```

Standard Linux practice: user `www-data` often belongs to group `www-data`, so `user:user` usually works. But if the site user's primary group differs (e.g., site created with custom group), this silently sets wrong group ownership. Should use `s.User+":"+s.Group` if the `Site` struct has a group field, or read the user's actual primary group via `id -gn <user>`.

### 8. No confirmation dialog before destructive operations in TUI

`backup.go:68-75`: pressing `d` immediately emits `DeleteBackupMsg`. Pressing `r` immediately emits `RestoreBackupMsg`. Both are destructive (delete archive; overwrite live site). Existing screens (`services.go`, `queues.go`) have the same pattern — but for backup this is higher stakes.

This is a UX issue, not a security issue (manager still validates), but a miskey destroys a backup permanently.

### 9. `List()` does not populate `BackupInfo.Type` field

`manager.go:328-334`:
```go
backups = append(backups, BackupInfo{
    Path:      filepath.Join(backupDir, entry.Name()),
    Domain:    domain,
    Size:      info.Size(),
    CreatedAt: createdAt,
    // Type: missing!
})
```

`BackupInfo.Type` is always zero-value (`""`). The TUI displays domain/size/date but not type (so no user-visible bug currently). However, any downstream code checking `bk.Type` will be incorrect. Reading the embedded `metadata.toml` for each backup in `List()` would fix this but has I/O cost; alternatively, encode type into the filename.

---

## Notes (Nice to Have)

### 10. `rows` string built with `+=` — minor perf

`backup.go:100`: `var rows string` + `rows += row + "\n"` allocates on each iteration. Use `strings.Builder`. Same pattern exists in `services.go` — low priority, consistent with codebase style.

### 11. Test coverage gaps

`manager_test.go` only unit-tests pure functions (`backupFilename`, `parseBackupFilename`, `validateDomain`, `FormatSize`, `Cleanup` validation). No tests for:
- `Create()` / `Restore()` (expected given OS dependency, but mock `executor` exists)
- `SetupCron()` / `RemoveCron()`
- `Delete()` path containment (critical fix #1 above)
- `List()` with non-empty dir

The existing `database/manager_test.go` pattern uses a mock executor. Backup should follow suit for `Create`/`Restore`/`SetupCron`.

### 12. `writeMetadata` uses `os.Create` (truncate) rather than atomic write

`manager.go:413`: `os.Create(path)` writes directly. In the backup flow this is inside a temp dir so a partial write is harmless (temp dir is removed on error). No change needed here — just noting it differs from `system.FileManager.WriteAtomic` used elsewhere.

### 13. `app.go` — `previous` field unused in backup flow

`app.go:48`: `previous Screen` is set in `NavigateMsg` handler but `goBack()` ignores it — hardcodes `ScreenDashboard` for all non-sub-screens. This predates Phase 09. `ScreenBackup` would benefit from using `a.previous` in `goBack()`. Low priority.

---

## Architecture Assessment

**Follows existing patterns:** Yes.
- `Manager` struct with injected `executor`/`files`/`db` dependencies mirrors `database.Manager`, `service.Manager`.
- `BackupScreen` structure, key bindings, message types, and `SetBackups`/`SetError` setters mirror `ServicesScreen` exactly.
- `app.go` additions (screen enum, `screenNames` map, `backupScreen` field, `updateActiveScreen` case, `View` case, `currentBindings` case) all consistent with prior screens.

**YAGNI/KISS/DRY:** No violations found.
- `backupFilename`/`parseBackupFilename` are the right abstraction for the naming scheme.
- `FormatSize` is a standalone utility — appropriate.
- `SetupCron`/`RemoveCron` are simple file-write operations, no over-engineering.
- No dead code, no speculative abstractions.

**Metadata TOML inside archive:** Good design — self-describing backups portable across servers.

**15-minute timeout (`opTimeout`):** Reasonable for large sites.

---

## Summary

| Severity | Count | Items |
|---|---|---|
| Critical | 4 | #1 path traversal in Delete, #2 path traversal in Restore, #3 cron injection, #4 untrusted meta.Type |
| Warning | 5 | #5 relative backupDir, #6 temp dir in /tmp, #7 chown group assumption, #8 no confirm dialog, #9 missing Type in List |
| Note | 4 | #10 strings.Builder, #11 test gaps, #12 writeMetadata, #13 goBack unused previous |

Critical issues #1-#4 must be fixed before merge. All are in `manager.go` and have straightforward fixes.
