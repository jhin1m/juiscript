# Bug Report: J/K Keys Consumed During Text Input in TUI Screens

**Date:** 2026-03-10
**Severity:** Medium — characters silently dropped, no error shown to user

---

## Executive Summary

When a `FieldText` field is active in `FormModel`, pressing `j` or `k` does NOT append those characters. Instead they are consumed as navigation (cycle forward/backward). Root cause is in the shared `FormModel.Update()` — it binds `j`/`k` unconditionally without checking the current field type.

Screens that use `FormModel` with `FieldText` fields are all affected. Screens that implement their own inline text input buffer (`FirewallScreen`) correctly gate navigation keys via a mode flag and are NOT affected.

---

## Root Cause

**File:** `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/components/form.go`
**Lines:** 125–129

```go
case "tab", "down", "j":
    m.handleCycleForward()

case "shift+tab", "up", "k":
    m.handleCycleBackward()
```

`handleCycleForward` / `handleCycleBackward` (lines 190–218) are no-ops for `FieldText` fields — they only act on `FieldSelect` and `FieldConfirm`. However, crucially, the `j`/`k` keys are matched **before** the `default:` branch (line 136–140) that appends characters to `m.input`. Because Go's `switch` short-circuits on first match, `j` and `k` are consumed as navigation keys and never reach the text-append logic, silently dropping them.

The `default:` branch (line 138) correctly appends characters only when `FieldText` is active:
```go
if m.step < len(m.fields) && m.fields[m.step].Type == FieldText && len(msg.String()) == 1 {
    m.input += msg.String()
}
```
But it is unreachable for `j` and `k` due to the earlier case clauses.

---

## Affected Screens (use FormModel with FieldText fields)

| Screen file | Text fields in form | Line of j/k bind (delegated to FormModel) |
|---|---|---|
| `screens/sitecreate.go` | `domain` (FieldText) | line 71 — delegates all keys to `s.form.Update(msg)` |
| `screens/ssl.go` | `domain`, `email` (both FieldText) | line 83 — delegates to `s.form.Update(msg)` |
| `screens/database.go` | `name` (FieldText), `path` (FieldText) | line 90 — delegates to `d.form.Update(msg)` |
| `screens/backup.go` | `domain` (FieldText) | line 85 — delegates to `b.form.Update(msg)` |

**PHP screen** (`screens/php.go`) uses `FormModel` but only with `FieldSelect` — not affected functionally (j/k cycle options as intended).

---

## Screens NOT Affected

| Screen | Reason |
|---|---|
| `screens/firewall.go` | Custom `inputMode` flag; `updateInput()` uses `default:` branch for all printable chars — j/k correctly appended (lines 202–206) |
| `screens/sites.go` | No text input; j/k is pure list nav |
| `screens/sitedetail.go` | No text input; j/k is pure list nav |
| `screens/services.go`, `nginx.go`, `queues.go`, `dashboard.go` | No forms/text input |
| `screens/setup.go` | No text input; j/k is pure checklist nav |

---

## Fix Location

Single fix in `form.go` — guard `j`/`k` so they only navigate when the current field is NOT `FieldText`:

**`/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/tui/components/form.go`, lines 125–129**

Change the unconditional `case "tab", "down", "j":` and `case "shift+tab", "up", "k":` to skip (fall through to `default:`) when `m.step < len(m.fields)` and the current field is `FieldText`. All four affected screens will be fixed automatically since they delegate to `FormModel`.

---

## Unresolved Questions

None — root cause fully identified with single fix point.
