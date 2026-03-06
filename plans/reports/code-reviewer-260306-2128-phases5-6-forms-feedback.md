# Code Review: TUI Phases 5+6 - Input Forms & Feedback Integration

## Scope
- Files reviewed: 9 (php.go, database.go, ssl.go, backup.go, sitedetail.go, app.go, app_handlers_db.go, app_handlers_ssl.go, app_handlers_backup.go)
- Plan files: plans/260306-2056-tui-input-forms/phase5-wire-forms.md, phase6-wire-feedback.md
- LOC analyzed: ~850

## Overall Assessment
Solid implementation. All 6 placeholder handlers replaced, UX flow (form -> spinner -> toast) works correctly. Code is clean and consistent. Build passes, `go vet` clean. One critical pre-existing test failure in `provisioner` package (unrelated to this work). No new regressions introduced.

---

## Critical Issues

None.

---

## High Priority Findings

### 1. Hardcoded export path in `app_handlers_db.go:69`

```go
path := fmt.Sprintf("/var/backups/juiscript/%s.sql.gz", name)
```

No validation that `name` is safe for filesystem use. A DB name containing `/` or `..` would produce a malicious path. The `validateDBName` regex (`^[a-zA-Z0-9_]{1,64}$`) on the form side prevents this at the UI layer, but the handler should not assume it received sanitized input (defense in depth).

Fix: validate `name` against same regex before constructing the path, or call `dbNameRegex.MatchString(name)` and return `DBOpErrMsg` on failure.

### 2. webRoot derivation in `app_handlers_ssl.go:31`

```go
webRoot := fmt.Sprintf("%s/site_%s/public", a.cfg.General.SitesRoot,
    strings.ReplaceAll(domain, ".", "_"))
```

This convention (`site_example_com`) is assumed but never documented/tested. If the actual site directory naming differs, `certbot`'s webroot challenge will fail silently with an opaque error. Also, `domain` is user input here - path traversal via domain `../../etc` is blocked by `site.ValidateDomain` at the form level, but not at the handler level.

Fix: add a `site.ValidateDomain(domain)` call at the start of `handleObtainCert`, returning `SSLOpErrMsg` on failure.

---

## Medium Priority Improvements

### 3. Race condition: spinner started inside `cmd()` closure (all spinner screens)

Pattern in `php.go:102`, `ssl.go:104`, `backup.go:108`:
```go
return p, func() tea.Msg {
    result := cmd()
    switch v := result.(type) {
    case components.FormSubmitMsg:
        p.formActive = false
        p.spinner.Start("Installing PHP " + version + "...")  // mutates screen state inside cmd goroutine
        return InstallPHPMsg{...}
```

Bubble Tea runs `tea.Cmd` functions off the main loop goroutine. Mutating `p.formActive` and calling `p.spinner.Start()` from inside a `Cmd` is a data race. The canonical pattern is to mutate state in `Update()` (main loop), not inside the returned `Cmd` closure.

Fix: restructure so `formActive = false` and `spinner.Start()` happen synchronously in `Update()` before returning the cmd, not inside the closure.

Example:
```go
case components.FormSubmitMsg:
    // Mutate state here (main loop, safe)
    p.formActive = false
    p.spinner.Start("Installing PHP " + v.Values["version"] + "...")
    return p, func() tea.Msg { return InstallPHPMsg{Version: v.Values["version"]} }
```

However, the current approach works in practice because the cmd closure calls the inner `cmd()` synchronously and returns - Bubble Tea calls these closures on a separate goroutine but the mutation happens before the Msg is dispatched. Still, this is fragile and violates the TEA architecture contract.

### 4. `d.cursor` used in closure after potential mutation (`database.go:104`)

```go
action := d.formAction
return d, func() tea.Msg {
    result := cmd()
    case "import":
        return ImportDBMsg{
            Name: d.dbs[d.cursor].Name,  // d.cursor captured by reference, not by value
```

`d.cursor` is read inside the cmd closure but could have changed between the time the closure was created and when it executes. If the user navigates the list quickly before the cmd fires, wrong DB name will be imported.

Fix: capture `name := d.dbs[d.cursor].Name` before returning the closure.

### 5. `formActive = false` inside cmd closure (all screens)

Same structural issue as #3: `d.formActive = false` is set inside the closure body. Should be set in `Update()` immediately when form result is detected. This is safe today because the inner `cmd()` is synchronous, but is architecturally incorrect TEA.

### 6. `validateFilePath` is trivially weak (`database.go:24-29`)

Only checks non-empty. Accepts paths like `http://evil.com`, relative paths `../../etc/passwd`, or non-existent files. The backend does perform its own validation, but the form-level validator gives no useful feedback.

Acceptable for now (backend is the real gate), but a note for future: add `filepath.IsAbs(path)` check at minimum.

---

## Low Priority Suggestions

### 7. `availablePHPVersions` hardcoded in `php.go:14`

Plan doc says "Dynamic fetch from phpMgr" was confirmed as decision. The form uses a hardcoded list instead. This means users can attempt to install a version the system doesn't support. Not a bug today, but deviates from the confirmed plan decision.

### 8. Toast messages are generic

All OpDone handlers use generic messages ("Database operation completed", "Backup operation completed"). Plan's Phase 6 TODO says "Specific toast messages per operation (e.g., 'PHP 8.3 installed', 'Database mydb dropped')". This is a minor polish gap - the check is marked done but the implementation is generic.

### 9. `validateEmail` is trivially weak (`ssl.go:16-19`)

Only checks `strings.Contains(email, "@")`. Acceptable for UX convenience since certbot will validate properly, but could give false positives (e.g., `@` alone passes).

### 10. `delete` backup confirm message shows `bk.Domain` not filename (`backup.go:159`)

```go
d.confirm.Show(fmt.Sprintf("Delete backup '%s'? This cannot be undone.", bk.Domain))
```

Shows domain name, not the specific backup file/timestamp. If a domain has multiple backups, the user can't tell which one they're deleting. Consider including `bk.CreatedAt` or `bk.Path` basename in the message.

---

## Positive Observations

- **Consistent pattern**: All 4 screen files follow identical confirm/form/spinner priority ordering in both `Update()` and `View()`. Easy to read and maintain.
- **Safe defaults**: `ConfirmModel` defaults to "No" selection - correct safety design.
- **Graceful nil guards**: All handlers check `if a.xxxMgr == nil` before use - proper defensive coding.
- **DRY message types**: Msg types are minimal and carry exactly what's needed.
- **`pendingDomain` in BackupScreen**: Correctly captures domain at confirm time from `BackupInfo.Domain` - avoids the cursor-drift issue for the restore action (though import still has the issue noted in #4).
- **`go vet` clean**: Zero issues.

---

## Task Completeness

**Phase 5 TODO**: All 26 checklist items marked done and verified implemented.
**Phase 6 TODO**: All items marked done. Minor gap: toast messages are generic, not operation-specific as intended.

**Pre-existing failing test** (not introduced by this work):
- `internal/provisioner: TestDetectAll_PHPPlaceholderWhenNotInstalled` - no provisioner files modified in Phases 5+6.

---

## Recommended Actions

1. **(High)** Fix cursor capture in `database.go` import closure - capture `name` before returning func.
2. **(High)** Add `site.ValidateDomain(domain)` guard at start of `handleObtainCert` handler.
3. **(High)** Add DB name validation in `handleExportDB` before path construction.
4. **(Medium)** Move state mutations (`formActive = false`, `spinner.Start()`) out of cmd closures into `Update()` body.
5. **(Low)** Improve delete backup confirm message to include backup date/path.
6. **(Low)** Update generic toast messages to include operation details.

---

## Metrics
- Build: PASS
- go vet: PASS (0 issues)
- Component tests: PASS
- Provisioner test: FAIL (pre-existing, unrelated)
- Critical issues: 0
- High: 2
- Medium: 4
- Low: 4

---

## Unresolved Questions
- Is the `site_domain_com` webRoot convention (`site_` prefix + dots-to-underscores) documented anywhere? If it diverges from how `site.Manager` creates directories, SSL obtain will fail.
- Should `availablePHPVersions` be fetched dynamically from `phpMgr` as per the confirmed plan decision, or is the hardcoded list intentional for now?
