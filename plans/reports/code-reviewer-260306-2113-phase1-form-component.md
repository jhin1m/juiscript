# Code Review: Phase 1 - TUI Form Component

## Scope
- Files reviewed: `internal/tui/components/form.go`, `internal/tui/components/form_test.go`
- Pattern reference: `internal/tui/screens/sitecreate.go`
- Lines analyzed: ~306 (form.go) + ~298 (form_test.go)
- Review focus: Phase 1 implementation against plan requirements

## Overall Assessment

Solid implementation. All 13 tests pass, build clean. Architecture correctly matches sitecreate.go's step-by-step pattern with good generalization. No security issues. Minor concerns below.

## Critical Issues

None.

## High Priority Findings

### 1. `confirmOn` multi-field collision (form.go:79-81, 121-123)

`confirmOn` is a single bool on `FormModel`. If multiple `FieldConfirm` fields are defined, each one overwrites the shared toggle state — last field's `Default` wins during `NewForm`/`Reset`, and all confirm fields share the same toggle during navigation.

```go
// Current - single bool shared across all FieldConfirm fields
confirmOn bool

// Fix - store per field like selectIdx
confirmOn map[string]bool  // field key -> state
```

Not a bug today (no existing form uses 2+ confirm fields), but a silent trap for Phase 5 wiring.

### 2. `Active()` comment mismatch (form.go:86-89)

Doc says "Active returns true if the form is still in progress" but the method name in the plan spec says it should return `true` while `step < len(fields)+1`. The condition `step <= len(fields)` is correct but the comment at line 45 says `len(fields) = confirm step` which is misleading — confirm is step `len(fields)`, submit fires at that step on enter. Post-submit the form never sets `step > len(fields)`, so `Active()` never returns `false` after submit. Parent screen must check for `FormSubmitMsg` to deactivate, not rely on `Active()`.

Not a bug if used correctly, but the API contract is unclear. Consider setting `m.step = len(m.fields)+1` after submit fires, so `Active()` returns `false`.

## Medium Priority Improvements

### 3. DRY violation: default-application logic duplicated (form.go:64-82 vs 108-124)

`NewForm` and `Reset` contain identical loops for applying defaults. Extract to a private method:

```go
func (m *FormModel) applyDefaults() {
    for _, f := range m.fields {
        if f.Default != "" {
            m.values[f.Key] = f.Default
        }
        if f.Type == FieldSelect && f.Default != "" {
            for i, opt := range f.Options {
                if opt == f.Default {
                    m.selectIdx[f.Key] = i
                    break
                }
            }
        }
        if f.Type == FieldConfirm {
            m.confirmOn = f.Default == "yes"
        }
    }
}
```

### 4. `GetValues()` naming inconsistency

Plan spec says `Values()`, implementation uses `GetValues()`. Go convention prefers no `Get` prefix. `sitecreate.go` doesn't expose values at all (caller gets them from the Msg). Inconsistency is fine if used only internally, but the exported name `GetValues` deviates from Go idiom.

### 5. `sendKey` test helper uses `tea.KeyRunes` for single chars (form_test.go:17)

This works but mismatches `Update()` which detects single-char input via `len(msg.String()) == 1`. Special keys sent via `tea.KeyRunes` also have `len == 1` behavior for single runes — technically fine, but using named key types for special keys (tab, enter, etc.) in `sendSpecialKey` while using `KeyRunes` for chars creates an asymmetry. Tests pass, low risk.

## Low Priority Suggestions

### 6. View renders `errLine` as empty string when no error

`lipgloss.JoinVertical` with an empty `errLine` string inserts a blank line gap in the layout. Consider using a conditional slice instead:

```go
parts := []string{title, "", fields}
if m.err != nil {
    parts = append(parts, m.theme.ErrorText.Render(fmt.Sprintf("  Error: %v", m.err)))
}
parts = append(parts, help)
return lipgloss.JoinVertical(lipgloss.Left, parts...)
```

This matches how `sitecreate.go` handles it (same pattern, same minor issue there too).

### 7. `width` field set but never used in `View()`

`SetWidth()` / `m.width` stored but `View()` doesn't use it for truncation or wrapping. Fine for Phase 1 (YAGNI), but consider removing `SetWidth` until needed or at least applying it to the title/field rendering.

### 8. No test for `View()` content (form_test.go:256-265)

`TestFormModel_View` only checks `view != ""`. Given this is a UI component, a snapshot/contains test for placeholder, cursor char `_`, active label, and error line would add value. Low priority but improves regression safety for Phase 5.

## Positive Observations

- Clean separation of field-type logic into `handleEnter/CycleForward/CycleBackward`
- Correct modular arithmetic for backward cycling: `(idx - 1 + len) % len`
- `GetValues()` returns a copy — prevents caller mutation of internal state
- Default `selectIdx` correctly resolved from `Default` string match
- Test coverage is comprehensive: 13 tests covering all field types, edge cases, submit, cancel, reset
- Style usage perfectly mirrors `sitecreate.go` (`Active/Subtitle/OkText/ErrorText/HelpDesc`) — visual consistency guaranteed
- `FormSubmitMsg.Values` is a plain `map[string]string` — easy for callers to consume without importing extra types

## Task Completeness

All Phase 1 TODO items from `plans/260306-2056-tui-input-forms/phase1-form-component.md` are complete:

| Task | Status |
|------|--------|
| Define FieldType, FormField, FormSubmitMsg, FormCancelMsg | DONE |
| FormModel struct with step tracking, values map, input buffer | DONE |
| NewForm() constructor with defaults | DONE |
| Update() key handling | DONE |
| Text field: char input, backspace, validation | DONE |
| Select field: tab/j/k cycle, enter confirms | DONE |
| Confirm field: tab toggles, enter confirms | DONE |
| Confirm step (auto after all fields) | DONE |
| View() with progressive field reveal | DONE |
| Reset() | DONE |
| Active() bool | DONE |
| Unit tests (all listed cases + more) | DONE |

## Recommended Actions

1. **Fix `confirmOn` to be a map** (High) — before Phase 5 wiring if any screen uses 2+ confirm fields
2. **Clarify `Active()` post-submit contract** (High) — set `step = len(fields)+1` after submit or document that callers must not rely on it
3. **Extract `applyDefaults()`** (Medium) — eliminates DRY violation before Phase 5 adds more reset paths
4. **Rename `GetValues()` to `Values()`** (Low) — Go idiom alignment

## Metrics

- Tests: 13/13 pass
- Build: clean
- Linting issues: 0 blocking
- Type coverage: 100% (Go static typing)
- DRY violations: 1 (applyDefaults duplication)

---

Unresolved questions:
- Will any Phase 5 screen use multiple `FieldConfirm` fields? If yes, issue #1 is critical.
- Should `FormModel` ever be used as `tea.Model` (with `Init()`)? Currently it's a sub-component only — no `Init()` method. Fine for embedding pattern but worth confirming with Phase 5 plan.
