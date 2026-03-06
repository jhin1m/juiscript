# Phase 1.5: Refactor SiteCreate to Use FormModel

**Status: DONE** (2026-03-06)

## Context
- [Plan overview](./plan.md)
- Depends on: Phase 1 (FormModel component)
- Reference: `internal/tui/screens/sitecreate.go`

## Overview
Refactor sitecreate.go to use the new FormModel from Phase 1. Validates FormModel works correctly with a real screen before wiring into other screens (Phase 5). Acts as integration test for FormModel.

## Key Insights
- sitecreate.go currently has custom step enum (stepDomain=0..stepConfirm=4) with manual key handling
- FormModel provides the same UX but data-driven -- should be drop-in replacement
- 4 fields: domain (text), project type (select), PHP version (select), create DB (confirm)
- PHP version options should be dynamic (fetched from phpMgr via pre-fetch)
- This refactor validates FormModel before mass adoption in Phase 5

## Requirements
1. Replace custom step/input/selectIdx logic with embedded FormModel
2. Keep identical UX behavior (enter/tab/esc keybindings)
3. PHP version list from dynamic data (passed to SiteCreate on navigate)
4. All existing site creation functionality must work unchanged
5. Run `make test` -- no regressions

## Architecture

### Current SiteCreate Fields (to remove)
```go
step       int
input      string
projType   site.ProjectType
phpVersion string
createDB   bool
err        error
```

### New SiteCreate Fields
```go
form       *components.FormModel
formActive bool  // always true for this screen (it IS a form)
phpVersions []string  // passed from App on navigate
```

### Form Definition
```go
fields := []FormField{
    {Key: "domain", Label: "Domain", Type: FieldText, Validate: site.ValidateDomain},
    {Key: "projectType", Label: "Project Type", Type: FieldSelect,
     Options: []string{"laravel", "wordpress"}, Default: "laravel"},
    {Key: "phpVersion", Label: "PHP Version", Type: FieldSelect,
     Options: s.phpVersions, Default: s.phpVersions[0]},
    {Key: "createDB", Label: "Create Database", Type: FieldConfirm, Default: "yes"},
}
```

### On FormSubmitMsg
```go
opts := site.CreateOptions{
    Domain:      values["domain"],
    ProjectType: mapProjectType(values["projectType"]),
    PHPVersion:  values["phpVersion"],
    CreateDB:    values["createDB"] == "yes",
}
return s, func() tea.Msg { return CreateSiteMsg{Options: opts} }
```

## Related Code Files
- `internal/tui/screens/sitecreate.go` -- full rewrite
- `internal/tui/components/form.go` -- FormModel from Phase 1
- `internal/tui/app.go` -- pass phpVersions to SiteCreate on navigate

## Implementation Steps

### TODO
- [x] Add `phpVersions []string` field to SiteCreate, populate from App on navigate
- [x] Replace step/input/err fields with `form *FormModel`
- [x] Define form fields using FormField slice
- [x] Update `Update()` to delegate to form.Update()
- [x] Handle FormSubmitMsg: construct CreateSiteMsg from form values
- [x] Handle FormCancelMsg: emit GoBackMsg
- [x] Update `View()` to render form.View()
- [x] Remove old handleEnter/handleNext/handlePrev methods
- [x] Remove step enum constants
- [x] Run `make test` to verify no regressions
- [x] Manual test: create site flow works end-to-end

## Success Criteria
- SiteCreate uses FormModel internally
- Identical UX to current implementation
- PHP versions populated dynamically
- All existing tests pass
- FormModel validated for real-world use before Phase 5

## Risk Assessment
- **Medium**: Behavior regression in site creation flow. Mitigate: thorough manual testing, keep old code in git history
- **Low**: Dynamic PHP versions empty on first load. Mitigate: fallback to ["8.3", "8.2", "8.1"] if empty

## Security Considerations
- Domain validation still enforced via FormField.Validate callback
- No change to backend security -- only UI layer refactored

## Next Steps
After this phase, FormModel is validated. Phase 5 wires forms into remaining screens.
