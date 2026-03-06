# Phase 1: Reusable Form Component

## Context
- [Plan overview](./plan.md)
- Pattern reference: `internal/tui/screens/sitecreate.go`
- Theme: `internal/tui/theme/theme.go`

## Overview
Create a generic, composable `FormModel` that screens can embed to collect user input. Follows sitecreate.go's step-by-step pattern but is data-driven via field definitions.

## Key Insights
- sitecreate.go uses manual step enum + switch for each field type - works but won't scale
- All forms in this project need only 3 field types: text input, single-select, confirm (yes/no)
- Form should be embeddable (not a separate screen) - screens toggle `formActive` bool
- On submit, form returns field values; screen constructs the appropriate Msg

## Requirements
1. Generic form with configurable fields (title, fields slice, submit callback)
2. Field types: `TextInput`, `Selection`, `Confirm`
3. Step-by-step navigation matching sitecreate.go UX (enter=next, tab/j/k=cycle, esc=cancel)
4. Per-field validation with inline error display
5. Styled with existing theme (Active/Inactive labels, ErrorText for errors)
6. Returns `FormSubmitMsg{Values map[string]string}` on confirm
7. Returns `FormCancelMsg{}` on esc

## Architecture

### File: `internal/tui/components/form.go`

```go
// FieldType identifies the kind of form field.
type FieldType int
const (
    FieldText FieldType = iota    // free text input
    FieldSelect                    // pick from options list
    FieldConfirm                   // yes/no toggle
)

// FormField defines a single form field.
type FormField struct {
    Key         string            // unique key for value lookup
    Label       string            // display label
    Type        FieldType
    Options     []string          // for FieldSelect
    Default     string            // initial value
    Placeholder string            // for FieldText
    Validate    func(string) error // optional validator
}

// FormSubmitMsg carries completed form data.
type FormSubmitMsg struct {
    Values map[string]string
}

// FormCancelMsg signals the user cancelled the form.
type FormCancelMsg struct{}

// FormModel is a reusable step-by-step form component.
type FormModel struct {
    theme    *theme.Theme
    title    string
    fields   []FormField
    step     int
    values   map[string]string   // field key -> value
    input    string              // current text buffer
    selectIdx map[string]int     // current selection index per field key
    err      error
    width    int
}
```

### Key Methods
- `NewForm(t *theme.Theme, title string, fields []FormField) *FormModel`
- `Update(msg tea.Msg) (*FormModel, tea.Cmd)` - handles keys, returns submit/cancel msgs
- `View() string` - renders current step with field label, value, cursor, error
- `Active() bool` - returns true if form is in progress (step < len(fields)+1)
- `Reset()` - clears all state for reuse

### Navigation Logic
```
enter  -> validate current field -> advance step
         on last field + confirm -> FormSubmitMsg{Values}
tab/j  -> cycle forward (select/confirm fields)
shift+tab/k -> cycle backward
backspace  -> trim text input buffer
esc    -> FormCancelMsg{}
single char -> append to text buffer (text fields only)
```

### View Layout
```
  Title

  Label:  value_          <- active field with cursor
  Label:  [selected]      <- completed select field

  Press Enter to confirm, Esc to cancel   <- confirm step

  Error: validation message               <- if error

  enter:next  tab:cycle  esc:cancel       <- help line
```

## Related Code Files
- `internal/tui/screens/sitecreate.go` - existing form pattern to match
- `internal/tui/theme/theme.go` - Active, Inactive, ErrorText styles
- `internal/tui/components/helpers.go` - shared component utilities

## Implementation Steps

### TODO
- [x] Define `FieldType`, `FormField`, `FormSubmitMsg`, `FormCancelMsg` types
- [x] Implement `FormModel` struct with step tracking, values map, input buffer
- [x] Implement `NewForm()` constructor; set defaults from `FormField.Default`
- [x] Implement `Update()` with key handling matching sitecreate.go pattern
- [x] Implement text field: char input, backspace, validation on enter
- [x] Implement select field: tab/j/k cycles Options, enter confirms
- [x] Implement confirm field: tab toggles yes/no, enter confirms
- [x] Add confirm step (auto-appended after all fields) showing summary
- [x] Implement `View()` with progressive field reveal (only show completed + current)
- [x] Implement `Reset()` to clear all state
- [x] Add `Active() bool` method
- [x] Write unit tests: text input, select cycling, confirm toggle, validation, submit, cancel

### Phase Status: COMPLETE (2026-03-06, 14:00 UTC)
Phase 1 delivered. All 13 tests pass. FormModel ready for Phase 1.5 refactor and Phase 5 wiring.

### Known Issues (from code review)
- `confirmOn` is a single bool — breaks if multiple `FieldConfirm` fields used; fix before Phase 5 if needed
- `Active()` never returns `false` after submit (step not advanced past confirm); callers must use `FormSubmitMsg` to deactivate
- DRY: default-application logic duplicated in `NewForm` and `Reset`; extract to `applyDefaults()` before Phase 5
- `GetValues()` should be `Values()` per Go idiom

### Status: COMPLETE (2026-03-06)
All 13 tests pass. Build clean. Ready for Phase 1.5 refactor and Phase 5 wiring.

### Test File: `internal/tui/components/form_test.go`
- TestFormModel_TextInput - enter text, validate, advance
- TestFormModel_SelectCycle - tab cycles options forward/backward
- TestFormModel_ConfirmToggle - tab toggles yes/no
- TestFormModel_ValidationError - invalid input shows error, blocks advance
- TestFormModel_Submit - complete all fields, confirm produces FormSubmitMsg
- TestFormModel_Cancel - esc at any step produces FormCancelMsg
- TestFormModel_Reset - values cleared after reset

## Success Criteria
- FormModel usable by any screen via embedding
- All 3 field types work correctly
- Validation blocks step advancement on error
- Visual output matches sitecreate.go style
- 80%+ test coverage on form.go

## Risk Assessment
- **Low**: Form might need field-type-specific rendering tweaks per screen. Mitigate: keep View() simple, allow screens to wrap it
- **Low**: Key conflicts with parent screen. Mitigate: parent only forwards keys when `formActive` is true

## Security Considerations
- Text input validation prevents injection (each screen provides validator)
- No shell metacharacters should pass through to backend managers

## Next Steps
After this phase, screens can embed `FormModel` and toggle it on hotkey press. Phase 5 wires forms into actual screens.
