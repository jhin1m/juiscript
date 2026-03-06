# Code Review: sitecreate.go FormModel Refactor

## Scope
- Files reviewed: `internal/tui/screens/sitecreate.go`, `internal/tui/components/form.go`
- Lines analyzed: ~200
- Review focus: message handling correctness, security, YAGNI/KISS/DRY

## Overall Assessment
Clean, minimal refactor. Delegation to `FormModel` is correct. One **high** issue with synchronous `cmd()` execution pattern; remainder are low/informational.

---

## Critical Issues
None.

---

## High Priority Findings

### 1. Synchronous `cmd()` invocation breaks Bubbletea's command model
**File:** `sitecreate.go:73`

```go
result := cmd()   // <-- called directly inside Update()
switch v := result.(type) {
```

Bubbletea commands are `func() tea.Msg` — they are designed to be returned from `Update()` and executed by the runtime (potentially on a goroutine). Calling `cmd()` synchronously inside `Update()` is technically safe for pure/synchronous commands, but it:

- Discards the cmd after consuming its message (line 88 returns the same `cmd` again — the form's already-fired closure — meaning if the result was _not_ Submit/Cancel, the cmd fires **twice**: once manually here, and once by the runtime when returned).
- Is fragile: if FormModel ever returns an async command (e.g., a batch), it will deadlock or misbehave.

**The double-fire bug (line 88):**
```go
// if result was not Submit/Cancel:
return s, cmd   // runtime calls cmd() AGAIN → duplicate message
```

**Fix:** Use a proper message-interception pattern. Either:
- Have FormModel expose a `Result()` accessor that returns the last result type without firing a command, OR
- Return a wrapped command that maps the FormModel result:

```go
case tea.KeyMsg:
    _, cmd := s.form.Update(msg)
    if cmd == nil {
        return s, nil
    }
    return s, func() tea.Msg {
        result := cmd()
        switch v := result.(type) {
        case components.FormSubmitMsg:
            opts := site.CreateOptions{...}
            s.Reset()
            return CreateSiteMsg{Options: opts}
        case components.FormCancelMsg:
            s.Reset()
            return GoBackMsg{}
        default:
            return result
        }
    }
```

This fires `cmd()` exactly once, inside the returned command closure, and correctly forwards unknown messages.

---

## Medium Priority

### 2. `SetPHPVersions` does not rebuild the form
`SetPHPVersions` updates `s.phpVersions` but does not call `s.form = s.buildForm()`. The form was already constructed in `NewSiteCreate` with old defaults — the new versions are silently ignored unless `Reset()` is called afterwards.

**Fix:** Add `s.form = s.buildForm()` at end of `SetPHPVersions`, or document that callers must call `Reset()` after.

---

## Low Priority / Informational

### 3. `buildForm()` panics if `phpVersions` is empty
`s.phpVersions[0]` on line 48 with an empty slice will panic. The guard in `SetPHPVersions` (`len(versions) > 0`) prevents setting an empty list, but `defaultPHPVersions` is always non-empty so this is safe today. No action needed unless the default list can be externally mutated.

### 4. `WindowSizeMsg` not forwarded to `FormModel`
`FormModel.View()` uses fixed layout. If it ever needs responsive sizing, width/height will need passing. Non-issue currently.

---

## Positive Observations
- YAGNI/KISS/DRY: no leftover step enum, input buffer, or manual nav methods — full delegation to FormModel is correct.
- `Reset()` called on both submit and cancel paths — prevents stale state on re-entry.
- Field keys match exactly what `site.CreateOptions` expects; no magic string risk.
- `ValidateDomain` wired at field definition level — validation happens before domain reaches business logic.
- `CreateSiteMsg` kept in this file alongside its producer — good locality.

---

## Recommended Actions
1. **(High)** Fix double-fire bug: wrap `cmd()` call inside a returned command closure instead of calling it inline and then returning it again.
2. **(Medium)** Call `s.form = s.buildForm()` inside `SetPHPVersions` so version updates take effect immediately.

---

## Unresolved Questions
- None.
