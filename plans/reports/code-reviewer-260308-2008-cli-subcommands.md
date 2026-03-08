# Code Review: CLI Subcommands Feature

## Scope
- Files reviewed: `cmd/juiscript/main.go`, `cmd-site.go`, `cmd-php.go`, `cmd-db.go`, `cmd-ssl.go`, `cmd-service.go`, `cmd-backup.go`, `cmd-queue.go`
- Lines of code analyzed: ~700
- Review focus: security, DRY/KISS/YAGNI, API correctness, error propagation
- Build status: `go build ./cmd/juiscript/...` — clean, zero errors

## Overall Assessment
Solid implementation. Architecture is clean, manager APIs match their signatures exactly, and errors propagate correctly via `RunE`. Two medium-priority issues and a handful of low-priority items found — nothing blocking.

---

## Critical Issues

None.

---

## High Priority Findings

None.

---

## Medium Priority Improvements

### 1. Root check bypass via `cmd.Name()` — fragile and bypassable

**File:** `cmd/juiscript/main.go:120`

```go
if cmd.Name() == "version" {
    return nil
}
```

`PersistentPreRunE` is inherited by subcommands. If any future subcommand also names a child `"version"`, the check silently skips root enforcement for it. Use the command's full path to be explicit:

```go
if cmd.CommandPath() == "juiscript version" {
    return nil
}
```

Or compare by pointer: `if cmd == versionCmd { return nil }` (store the result of `versionCmd()` before `AddCommand`). The pointer approach is zero-ambiguity.

### 2. Silently ignored error from `template.New()`

**File:** `cmd/juiscript/main.go:72`

```go
tplEngine, _ := template.New()
```

`template.New()` returns `(*Engine, error)`. Since templates are embedded at compile time, the error is unlikely in practice — but if the embedded FS is malformed or the `templates/*` glob is empty, all config generation silently breaks. Propagate the error:

```go
tplEngine, err := template.New()
if err != nil {
    return nil, fmt.Errorf("load templates: %w", err)
}
```

---

## Low Priority Suggestions

### 3. `serviceActionCmd` produces grammatically broken output for "reload"

**File:** `cmd-service.go:90`

```go
fmt.Printf("Service %sed: %s\n", action, name)
```

`action = "reload"` produces `"Service reloaded: nginx"` — correct by accident.
`action = "restart"` produces `"Service restarted: nginx"` — correct by accident.
`action = "start"` produces `"Service started: nginx"` — correct.
`action = "stop"` produces `"Service stopped: nginx"` — correct.

All four happen to be valid English. No change needed unless new actions are added that don't follow the `+ed` pattern (e.g., `"run"` → `"Service runed"`). Low risk; acceptable.

### 4. `queueDomainActionCmd` same past-tense pattern

**File:** `cmd-queue.go:120`

Same as above. Fine for the current three verbs (start/stop/restart).

### 5. `siteCreate` accepts any string for `--type` and `--php`

**File:** `cmd-site.go:110-111`

The flags accept freeform strings. `site.Manager.Create` calls `ValidateProjectType` and `ValidatePHPVersion` internally, so invalid values do return errors — but the error message is less discoverable than Cobra's built-in enum validation. YAGNI applies here; the internal validation is sufficient.

### 6. `backup cleanup --keep 0` is valid and deletes all backups

**File:** `cmd-backup.go:141`

```go
cmd.Flags().IntVar(&keep, "keep", 5, "Number of recent backups to keep")
```

No lower-bound guard. `--keep 0` silently deletes everything for the domain. Whether the underlying `Cleanup` handles this intentionally is an internal concern, but a CLI-level guard (`if keep < 1 { return fmt.Errorf(...) }`) would prevent operator error. Low severity since this requires explicit root access.

### 7. `phpRemoveCmd` passes `nil` for `activeSites`

**File:** `cmd-php.go:79`

```go
mgrs.PHP.RemoveVersion(context.Background(), ver, nil)
```

Comment says "CLI user is responsible for checking dependencies." This is a UX risk — a user removing a PHP version that active sites depend on will get a broken state, not a helpful error. The `activeSites` guard in the manager exists precisely to prevent this. Consider fetching `mgrs.Site.List()` and passing the active PHP versions, or at minimum warn the user. Not a security issue.

---

## Positive Observations

- **API signatures match exactly.** Every manager method called matches the actual implementation signature verified against source. No mismatches found.
- **Error propagation is correct throughout.** All `RunE` callbacks return errors properly; no swallowed errors in command handlers.
- **DRY applied appropriately.** `serviceActionCmd` and `queueDomainActionCmd` eliminate repetition for the action-verb pattern without over-engineering.
- **Input validation is layered correctly.** DB names validated via `validateName` in the manager, paths via `safePathRegex`, domains via `validateDomain`/`ValidateDomain` — command layer doesn't need to duplicate this.
- **Path traversal defended.** `backup.isInsideBackupDir` uses `filepath.Abs` + prefix check correctly. `import-export.go` uses `safePathRegex` before shell interpolation.
- **`initManagers` extracted cleanly.** Single initialization point shared between TUI and CLI — correct approach.
- **`backupCmd` uses `FormatSize` for human-readable output** — consistent with TUI display.
- **Build is clean** — zero compile errors.

---

## Recommended Actions

1. **(Medium)** Fix root check to use `cmd.CommandPath()` or pointer comparison — prevents future accidental bypass.
2. **(Medium)** Propagate `template.New()` error in `initManagers` — silent failure path is a latent bug.
3. **(Low)** Add `--keep` lower-bound guard (`>= 1`) in `backupCleanupCmd` to prevent accidental total wipe.
4. **(Low/Optional)** Consider querying active sites before `phpRemoveCmd` to surface dependency conflicts proactively.

---

## Metrics
- Type Coverage: N/A (Go, statically typed — build clean)
- Test Coverage: Not measured (no new test files in scope)
- Linting Issues: 0 compile errors; minor style items noted above
- API Correctness: 100% — all manager call signatures verified against source

---

## Unresolved Questions

None.
