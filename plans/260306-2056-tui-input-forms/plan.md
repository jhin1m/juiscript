---
title: "Phase 2: TUI Input Forms & Feedback"
description: "Add reusable form component, confirmation dialogs, spinners, and toast notifications to TUI screens"
status: done
priority: P1
effort: 9h
branch: main
tags: [tui, forms, ux, bubble-tea]
created: 2026-03-06
---

# Phase 2: TUI Input Forms & Feedback

## Problem
6 placeholder handlers return "not yet implemented" errors because screens lack input forms. No feedback components (spinner, toast) exist for async operations.

## Approach
Create a reusable `FormModel` component (DRY) following sitecreate.go's step-by-step pattern, then wire it into existing screens as inline overlays. Add spinner, toast, and confirmation dialog components.

## Phases

| # | Phase | Effort | Status | Files |
|---|-------|--------|--------|-------|
| 1 | [Reusable Form Component](./phase1-form-component.md) | 2h | - | `components/form.go`, `components/form_test.go` |
| 1.5 | [Refactor SiteCreate](./phase1.5-refactor-sitecreate.md) | 1.5h | DONE 2026-03-06 | `screens/sitecreate.go` |
| 2 | [Confirmation Dialog](./phase2-confirm-dialog.md) | 1h | DONE 2026-03-06 | `components/confirm.go`, `components/confirm_test.go` |
| 3 | [Toast Notification](./phase3-toast.md) | 1h | DONE 2026-03-06 | `components/toast.go`, `components/toast_test.go` |
| 4 | [Spinner Component](./phase4-spinner.md) | 30m | DONE 2026-03-06 | `components/spinner.go` |
| 5 | [Wire Form Screens](./phase5-wire-forms.md) | 2h | DONE 2026-03-06 | `php.go`, `database.go`, `ssl.go`, `backup.go`, handlers |
| 6 | [Wire Feedback Components](./phase6-wire-feedback.md) | 1h | DONE 2026-03-06 | `app.go`, screen files |

## Key Design Decisions
- **Custom keys, NOT huh library** - match sitecreate.go pattern
- **Inline form overlays** - forms render within existing screens, not separate screen states
- **Form as composable component** - screens embed `FormModel`, toggle `formActive bool`
- **Messages carry form data** - updated Msg types include user input (e.g., `InstallPHPMsg{Version}`)

## Dependencies
- Phase 1 must complete before Phase 1.5
- Phase 1.5 validates FormModel before Phase 5
- Phases 2-4 are independent of each other
- Phase 6 depends on Phases 2-5

## Success Criteria
- All 6 placeholder handlers accept real user input
- Destructive actions require confirmation
- Long-running ops show spinner
- Success/error results show toast
- All new code has unit tests
- sitecreate.go refactored to use FormModel

## Validation Summary

**Validated:** 2026-03-06
**Questions asked:** 6

### Confirmed Decisions
- **PHP versions**: Dynamic fetch from phpMgr (not hardcoded list)
- **Domain input**: Select from active sites list (pre-fetched on screen navigate)
- **sitecreate.go**: Refactor to use FormModel in dedicated Phase 1.5
- **Toast duration**: 3s success, 5s error (type-differentiated)
- **Async data for forms**: Pre-fetch on navigation, form uses cached data
- **Refactor timing**: Phase 1.5 after FormModel stable, before wiring other screens

### Action Items
- [ ] Create phase1.5-refactor-sitecreate.md phase file
- [ ] Update phase1 FormModel: FieldSelect options can be dynamic (passed at form creation)
- [ ] Update phase5: PHP picker uses dynamic version list from phpMgr
- [ ] Update phase5: SSL/Backup domain fields use FieldSelect with pre-fetched sites list
- [ ] Update phase3: Toast duration differentiated by type (3s success, 5s error)
