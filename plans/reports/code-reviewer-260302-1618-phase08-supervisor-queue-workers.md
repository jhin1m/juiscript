# Code Review: Phase 08 - Supervisor/Queue Workers

**Date:** 2026-03-02
**Reviewer:** code-reviewer subagent

---

## Code Review Summary

### Scope
- Files reviewed: 6
  - `internal/supervisor/manager.go` (366 LOC)
  - `internal/supervisor/manager_test.go` (476 LOC)
  - `internal/template/templates/supervisor-worker.conf.tmpl`
  - `internal/tui/screens/queues.go` (174 LOC)
  - `internal/tui/app.go` (393 LOC)
  - `internal/tui/theme/theme.go` (95 LOC)
- Build: PASS (`go build ./...`)
- Tests: PASS (all 18 test cases)
- Vet: PASS (`go vet ./...`)

### Overall Assessment

Solid implementation. Follows the established manager pattern (Executor/FileManager/TemplateEngine injection), atomic writes with rollback, domain path-traversal guards. No security vulnerabilities. Two medium-priority issues worth fixing; rest are low/informational.

---

### Critical Issues

None.

---

### High Priority Findings

None.

---

### Medium Priority Improvements

**M1 — `Delete` has no rollback on reload failure**

`Create` correctly rolls back (removes config) if `reload` fails. `Delete` does not restore the config if reload fails after removal. This is an asymmetry and leaves the worker stopped but config gone.

```go
// internal/supervisor/manager.go:147-163
func (m *Manager) Delete(ctx context.Context, domain string) error {
    ...
    if err := m.files.Remove(confPath); err != nil { ... }

    // No rollback here — if reload fails, config is gone but supervisor
    // still has the old state in memory until next manual intervention.
    if err := m.reload(ctx); err != nil {
        return fmt.Errorf("supervisor reload after delete: %w", err)
    }
    ...
}
```

This is lower severity than it looks (supervisor `reread`/`update` failure on delete is rare and idempotent to retry), but it's an inconsistency with `Create`. Acceptable to leave as-is with a comment explaining intentionality, or add: save content before remove, restore on reload failure.

**M2 — `stopwaitsecs=3600` is hardcoded to match `MaxTime` default, but not parameterized**

Template line 13: `stopwaitsecs=3600`. This is the max time supervisor waits for a worker to stop gracefully. It's hardcoded to 3600 regardless of user-configured `MaxTime`. If a user sets `MaxTime=7200`, supervisor will kill the process at 3600 seconds during a stop — before the worker's own `--max-time` guard triggers. Correct behavior: `stopwaitsecs` should be >= `MaxTime`.

Fix: pass `MaxTime` + buffer to template, or derive `stopwaitsecs` from `MaxTime`:

```
stopwaitsecs={{ add .MaxTime 60 }}
```

Or minimally: document the constraint in `WorkerConfig.MaxTime` field comment and cap MaxTime validation at some reasonable ceiling that respects the hardcoded `stopwaitsecs`.

---

### Low Priority Suggestions

**L1 — `Status` returns first process only, silently discarding others**

```go
// manager.go:222
return &statuses[0], nil
```

For `numprocs > 1`, only process_00 is returned. This is acceptable for the TUI display use case but worth a comment explaining the intent. The `ListAll` result will show all processes anyway.

**L2 — `NewManagerWithConfDir` only exists for tests**

The constructor split (`NewManager` / `NewManagerWithConfDir`) is the same pattern as other managers in the codebase — consistent. No issue.

**L3 — `queues.go` TUI: `'d'` deletes without confirmation**

`DeleteWorkerMsg` fires immediately on `d` keypress with no confirm prompt. This matches the pattern of other destructive actions in the codebase (same as `DeleteVhostMsg`, `DropDBMsg`). Flagging only because supervisor delete removes a running process. Acceptable given the TODO-wiring state.

**L4 — `mockExecutor.RunWithInput` ignores `input` parameter**

```go
// manager_test.go:48
func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
    return m.Run(context.Background(), name, args...)
}
```

Supervisor manager never calls `RunWithInput`, so this is fine as a stub. But passing `context.Background()` instead of the received `_` context is a minor inconsistency (would lose cancellation in other usage contexts).

---

### Positive Observations

- Pattern consistency: exactly mirrors `nginx.Manager`, `service.Manager` — Executor, FileManager, TemplateEngine injection. Clean.
- Domain validation: regex + explicit path-traversal checks. Correct.
- `stopasgroup=true` + `killasgroup=true`: correct for preventing orphaned PHP processes.
- Error wrapping is consistent and descriptive throughout (`fmt.Errorf("...: %w", err)`).
- `Status` correctly handles supervisorctl exit code 3 (non-running but output present) — non-trivial edge case handled properly.
- `parseUptime` handles malformed input gracefully (returns 0 duration, no panic).
- Template uses `%02d` process numbering, matching supervisorctl expectations.
- `WarnText` added to theme for `STOPPED` state — appropriate, not overengineered.
- `app.go` wiring is complete with proper TODO stubs for all 4 worker message types.
- All 18 tests pass; coverage includes rollback, validation, custom params, uptime parsing.

---

### Recommended Actions

1. **Fix M2 (stopwaitsecs)**: Add `StopWaitSecs` to template data, derived as `MaxTime + 60`, or cap MaxTime to < 3600 in validation. This prevents silent worker kill on graceful stop.
2. **Address M1 (Delete rollback)**: Add a comment explaining the asymmetry is intentional (since re-running delete is safe), or implement content-save-and-restore on reload failure.

---

### Metrics

- Build: PASS
- Tests: 18/18 PASS
- Vet: PASS
- Linting issues: 0 critical, 0 high, 2 medium, 4 low
- Pattern compliance: Full (matches existing manager conventions)
- Security: No vulnerabilities found; workers correctly run as site user via `user={{ .User }}`
