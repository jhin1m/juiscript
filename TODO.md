# TODO - juiscript

## Phase 1: TUI-Backend Wiring -- DONE (v1.3.0)

All 22 TODO handlers wired. 9 backend managers injected via AppDeps pattern.

### Site Operations
- [x] Load site detail from manager
- [x] Call site manager to create site
- [x] Call site manager to toggle site
- [x] Call site manager to delete site

### Nginx Operations
- [x] Call nginx manager to toggle vhost
- [x] Call nginx manager to delete vhost
- [x] Call nginx manager to test config

### PHP Operations
- [x] Call PHP manager to install version
- [x] Call PHP manager to remove version (with active sites safety check)

### Database Operations
- [x] Call database manager to create DB
- [x] Call database manager to drop DB
- [x] Call database manager to import DB
- [x] Call database manager to export DB

### SSL Operations
- [x] Call SSL manager to obtain cert
- [x] Call SSL manager to revoke cert
- [x] Call SSL manager to renew cert

### Service Operations
- [x] Call service manager to start/stop/restart/reload (consolidated via handleServiceAction)

### Queue/Supervisor Operations
- [x] Call supervisor manager to start/stop/restart worker (consolidated via handleWorkerAction)
- [x] Call supervisor manager to delete worker

### Backup Operations
- [x] Call backup manager to create backup
- [x] Call backup manager to restore backup
- [x] Call backup manager to delete backup

## Phase 2: TUI Input Forms & Feedback -- DONE (v1.4.0)

Reusable FormModel, ConfirmModel, ToastModel, SpinnerModel components. All 6 placeholder handlers now accept real user input.

### Form Screens
- [x] PHP version picker for InstallPHPMsg
- [x] Database name input for CreateDBMsg
- [x] File path input for ImportDBMsg
- [x] Domain + email input for ObtainCertMsg
- [x] Domain + type selector for CreateBackupMsg
- [x] Confirmation dialog for destructive actions (delete site, drop DB, revoke cert, delete backup, restore backup)

### Progress & Feedback
- [x] Spinner cho long-running operations (backup, PHP install, SSL obtain)
- [x] Error toast/notification khi operation fail
- [x] Success toast sau mỗi action

## Phase 3: Missing Documentation (Priority: LOW)

- [ ] `docs/design-guidelines.md`
- [ ] `docs/deployment-guide.md`
- [ ] `docs/project-roadmap.md`

## Notes

- Backend modules: 11/11 complete voi test coverage
- TUI screens: 8/8 co UI rendering, 8/8 wired to backend (v1.3.0)
- TUI components: FormModel, ConfirmModel, ToastModel, SpinnerModel (v1.4.0)
- 9 managers injected via AppDeps: service, provisioner, php, site, nginx, database, ssl, supervisor, backup
- 28 handler methods across 8 domain files (app_handlers_*.go)
- 28 result/error message types in app_messages.go
- Data auto-fetched on screen navigation
- 28 component unit tests (13 form + 9 confirm + 6 toast)
