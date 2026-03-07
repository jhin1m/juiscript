# Phase 5: Wire Form Screens

## Context
- [Plan overview](./plan.md)
- Depends on: Phase 1 (FormModel component)
- Handler stubs: `app_handlers_php.go`, `app_handlers_db.go`, `app_handlers_ssl.go`, `app_handlers_backup.go`

## Overview
Embed `FormModel` into 4 screens (PHP, Database, SSL, Backup) and update message types + handlers to carry form data through to backend managers.

## Key Insights
- 6 placeholder handlers need forms, but they map to only 4 screens
- Database screen needs 2 forms (create, import) - only one active at a time
- Message types need updating: `InstallPHPMsg{}` -> `InstallPHPMsg{Version string}`
- SSL obtain needs 3 params: domain, webRoot (derived from domain), email
- Backup create needs: domain (string), type (BackupType enum)
- Backup restore already has path; just needs domain context from backup's metadata

## Requirements
1. PHP screen: version picker form on 'i' keypress
2. Database screen: name input form on 'c', file path form on 'i'
3. SSL screen: domain+email form on 'o'
4. Backup screen: domain+type form on 'c'
5. Update Msg types to carry form data
6. Update handlers to use form data for actual backend calls
7. Backup restore: extract domain from BackupInfo metadata (already available in backup list)

## Architecture

### Form Definitions Per Screen

**PHP - Version Picker** (php.go)
```go
fields := []FormField{
    {Key: "version", Label: "PHP Version", Type: FieldSelect,
     Options: []string{"8.3", "8.2", "8.1", "8.0"}, Default: "8.3"},
}
// On submit: InstallPHPMsg{Version: values["version"]}
```

**Database - Create** (database.go)
```go
fields := []FormField{
    {Key: "name", Label: "Database Name", Type: FieldText,
     Placeholder: "mydb", Validate: validateDBName},
}
// On submit: CreateDBMsg{Name: values["name"]}
```

**Database - Import** (database.go)
```go
fields := []FormField{
    {Key: "path", Label: "SQL File Path", Type: FieldText,
     Placeholder: "/path/to/dump.sql.gz", Validate: validateFilePath},
}
// On submit: ImportDBMsg{Name: selectedDB, Path: values["path"]}
```

**SSL - Obtain Cert** (ssl.go)
```go
fields := []FormField{
    {Key: "domain", Label: "Domain", Type: FieldText, Validate: validateDomain},
    {Key: "email", Label: "Email", Type: FieldText, Validate: validateEmail},
}
// On submit: ObtainCertMsg{Domain: values["domain"], Email: values["email"]}
```

**Backup - Create** (backup.go)
```go
fields := []FormField{
    {Key: "domain", Label: "Domain", Type: FieldText, Validate: validateDomain},
    {Key: "type", Label: "Backup Type", Type: FieldSelect,
     Options: []string{"full", "files", "database"}, Default: "full"},
}
// On submit: CreateBackupMsg{Domain: values["domain"], Type: values["type"]}
```

### Screen State Changes
Each screen adds:
```go
type PHPScreen struct {
    // ...existing fields...
    form       *components.FormModel
    formActive bool
    formAction string  // which action triggered the form (for DB screen: "create"|"import")
}
```

### Key Handling Pattern
```go
case tea.KeyMsg:
    if s.formActive {
        // Delegate all keys to form
        updated, cmd := s.form.Update(msg)
        s.form = updated
        return s, cmd
    }
    // Normal screen key handling...
    case "i":
        s.form = components.NewForm(s.theme, "Install PHP", phpFields)
        s.formActive = true
```

### Message Type Updates

| Current | Updated |
|---------|---------|
| `InstallPHPMsg{}` | `InstallPHPMsg{Version string}` |
| `CreateDBMsg{}` | `CreateDBMsg{Name string}` |
| `ImportDBMsg{Name}` | `ImportDBMsg{Name string, Path string}` |
| `ObtainCertMsg{}` | `ObtainCertMsg{Domain string, Email string}` |
| `CreateBackupMsg{}` | `CreateBackupMsg{Domain string, Type string}` |
| `RestoreBackupMsg{Path}` | `RestoreBackupMsg{Path string, Domain string}` |

### Handler Updates

**app_handlers_php.go** - `InstallPHPMsg` case calls `a.handleInstallPHP(msg.Version)`

**app_handlers_db.go**:
- `handleCreateDB()` -> `handleCreateDB(name string)` - calls `a.dbMgr.CreateDB(ctx, name)`
- `handleImportDB(name string)` -> `handleImportDB(name, path string)` - calls `a.dbMgr.Import(ctx, name, path)`

**app_handlers_ssl.go**:
- `handleObtainCert()` -> `handleObtainCert(domain, email string)` - derives webRoot from config, calls `a.sslMgr.Obtain(domain, webRoot, email)`

**app_handlers_backup.go**:
- `handleCreateBackup()` -> `handleCreateBackup(domain, backupType string)` - converts type string to `backup.BackupType`, calls `a.backupMgr.Create(ctx, opts)`
- `handleRestoreBackup(path string)` -> `handleRestoreBackup(path, domain string)` - calls `a.backupMgr.Restore(ctx, path, domain)`

### App.Update() Changes
```go
case screens.InstallPHPMsg:
    return a, a.handleInstallPHP(msg.Version)  // was placeholder error

case screens.CreateDBMsg:
    return a, a.handleCreateDB(msg.Name)

case screens.ImportDBMsg:
    return a, a.handleImportDB(msg.Name, msg.Path)

case screens.ObtainCertMsg:
    return a, a.handleObtainCert(msg.Domain, msg.Email)

case screens.CreateBackupMsg:
    return a, a.handleCreateBackup(msg.Domain, msg.Type)

case screens.RestoreBackupMsg:
    return a, a.handleRestoreBackup(msg.Path, msg.Domain)
```

## Related Code Files
- `internal/tui/components/form.go` - FormModel from Phase 1
- `internal/tui/screens/php.go` - embed form, emit InstallPHPMsg with version
- `internal/tui/screens/database.go` - embed form, two form modes (create/import)
- `internal/tui/screens/ssl.go` - embed form, emit ObtainCertMsg with domain+email
- `internal/tui/screens/backup.go` - embed form, emit CreateBackupMsg with domain+type
- `internal/tui/app.go` - update message case handlers
- `internal/tui/app_handlers_php.go` - wire InstallPHPMsg to handleInstallPHP(version)
- `internal/tui/app_handlers_db.go` - wire CreateDBMsg/ImportDBMsg with params
- `internal/tui/app_handlers_ssl.go` - wire ObtainCertMsg with domain+email
- `internal/tui/app_handlers_backup.go` - wire CreateBackupMsg/RestoreBackupMsg with params
- `internal/ssl/manager.go` - `Obtain(domain, webRoot, email)` signature
- `internal/database/import-export.go` - `Import(ctx, dbName, filePath)` signature
- `internal/backup/manager.go` - `Create(ctx, Options{Domain, Type})` signature

## Implementation Steps

### TODO
- [x] Update `InstallPHPMsg` to include `Version string` field
- [x] Update `CreateDBMsg` to include `Name string` field
- [x] Update `ImportDBMsg` to include `Path string` field
- [x] Update `ObtainCertMsg` to include `Domain string` and `Email string` fields
- [x] Update `CreateBackupMsg` to include `Domain string` and `Type string` fields
- [x] Update `RestoreBackupMsg` to include `Domain string` field
- [x] Add `form *FormModel`, `formActive bool` to PHPScreen; wire 'i' key to show form
- [x] Handle `FormSubmitMsg` in PHPScreen to emit `InstallPHPMsg{Version}`
- [x] Handle `FormCancelMsg` in PHPScreen to deactivate form
- [x] Add form fields to DatabaseScreen; wire 'c' key to create form, 'i' to import form
- [x] Handle form results in DatabaseScreen for both create and import actions
- [x] Add form fields to SSLScreen; wire 'o' key to show domain+email form
- [x] Handle form results in SSLScreen to emit `ObtainCertMsg{Domain, Email}`
- [x] Add form fields to BackupScreen; wire 'c' key to show domain+type form
- [x] Handle form results in BackupScreen to emit `CreateBackupMsg{Domain, Type}`
- [x] Update BackupScreen restore to extract domain from selected backup's metadata
- [x] Update `app.go` handler cases to pass msg fields to handler functions
- [x] Update `handleCreateDB()` signature and implementation
- [x] Update `handleImportDB()` signature and implementation
- [x] Update `handleObtainCert()` signature and implementation
- [x] Update `handleCreateBackup()` signature and implementation
- [x] Update `handleRestoreBackup()` signature and implementation
- [x] Update `InstallPHPMsg` handler in app.go to call `handleInstallPHP(msg.Version)`
- [x] Verify all forms render correctly (manual test or screenshot)
- [x] Run `make test` to ensure no regressions

## Success Criteria
- All 6 placeholder handlers replaced with real implementations using form data
- Forms collect valid input before triggering backend operations
- Backend manager methods called with correct parameters
- No compilation errors; all existing tests pass
- Form UX matches sitecreate.go style (step-by-step, progressive reveal)

## Risk Assessment
- **Medium**: Changing Msg struct fields is a breaking change across files. Mitigate: update all references in one pass, compile check
- **Low**: Form key handling conflicts with screen keys. Mitigate: `formActive` flag gates key delegation
- **Low**: SSL webRoot derivation - need to compute from config.SitesRoot + domain. Mitigate: `cfg.General.SitesRoot + "/" + domain + "/public"`

## Security Considerations
- All form inputs pass through backend manager validation (validateName, validateDomain, validatePath, validateEmail)
- Form-level validation is UX convenience; backend validation is the security boundary
- File path input for DB import should validate path exists and is readable

## Next Steps
After this phase, all backend operations are fully functional through TUI. Phase 6 adds polish (spinner, toast, confirmation).
