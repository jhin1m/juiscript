# TUI Screen Message Types & Backend Manager APIs

## Screen Message Summary

### SiteList
- **Emits:** ShowCreateFormMsg{}, ShowSiteDetailMsg{Domain}, ToggleSiteMsg{Domain}, GoBackMsg{}
- **Data Methods:** SetSites([]*site.Site), SetError(error)
- **No responses expected** — screens are stateless view containers

### SiteCreate
- **Emits:** CreateSiteMsg{Options: site.CreateOptions}
  - Contains: Domain, ProjectType, PHPVersion, CreateDB (bool)
- **Data Methods:** Reset() (clears form)
- **Form fields:** domain (text), projType (select), phpVersion (select), createDB (toggle)

### SiteDetail
- **Emits:** ToggleSiteMsg{Domain}, DeleteSiteMsg{Domain}, GoBackMsg{}
- **Data Methods:** SetSite(*site.Site)
- **Read-only menu** — shows site data and action options

### NginxScreen
- **Emits:** ToggleVhostMsg{Domain, CurrentlyEnabled}, DeleteVhostMsg{Domain}, TestNginxMsg{}, GoBackMsg{}
- **Data Methods:** SetVhosts([]nginx.VhostInfo), SetError(error)

### DatabaseScreen
- **Emits:** CreateDBMsg{}, DropDBMsg{Name}, ImportDBMsg{Name}, ExportDBMsg{Name}, GoBackMsg{}
- **Data Methods:** SetDatabases([]database.DBInfo), SetError(error)

### SSLScreen
- **Emits:** ObtainCertMsg{}, RevokeCertMsg{Domain}, RenewCertMsg{Domain}, GoBackMsg{}
- **Data Methods:** SetCerts([]ssl.CertInfo), SetError(error)

### BackupScreen
- **Emits:** CreateBackupMsg{}, RestoreBackupMsg{Path}, DeleteBackupMsg{Path}, GoBackMsg{}
- **Data Methods:** SetBackups([]backup.BackupInfo), SetError(error)

### QueuesScreen
- **Emits:** StartWorkerMsg{Name}, StopWorkerMsg{Name}, RestartWorkerMsg{Name}, DeleteWorkerMsg{Name}, GoBackMsg{}
- **Data Methods:** SetWorkers([]supervisor.WorkerStatus), SetError(error)

## Backend Manager Constructors

### site.Manager
```go
NewManager(cfg *config.Config, exec system.Executor, files system.FileManager, users system.UserManager, tpl *template.Engine) *Manager
```
- **Dependencies:** Config, Executor, FileManager, UserManager, Template
- **Nested:** Initializes nginx.Manager internally with (executor, files, template, sitesAvailable, sitesEnabled paths)

### supervisor.Manager
- **Constructor params:** Expects system.Executor, system.FileManager, template.Engine
- **WorkerConfig struct fields:** Domain, Username, SitePath, PHPBinary, Connection, Queue, Processes, Tries, MaxTime, Sleep
- **Default values:** redis connection, "default" queue, 1 process, 3 tries, 3600s max time, 3s sleep

### backup.Manager
- **Dependencies:** config.Config, site.Manager, database.Manager, system.Executor
- **BackupInfo struct:** Path, Domain, Type (BackupType enum), Size, CreatedAt
- **Operation timeout:** 15 minutes
- **Metadata includes:** Domain, Type, ProjectType, PHPVersion, DBName, DBUser, SiteUser

## Key Observations

1. **Message Pattern:** All screens emit domain-specific messages (no generic handlers)
2. **Data Flow:** Screens → Messages → App Router → Managers → SetData/SetError responses
3. **Error Handling:** Every data-receiving screen has SetError(error) for failure states
4. **Form Pattern:** SiteCreate uses form state machine (steps 0-4), emits final CreateSiteMsg
5. **Manager Composition:** site.Manager creates nginx.Manager; both need templating/execution
6. **Worker Config Defaults:** Applied in WorkerConfig.applyDefaults() method

## Wire Requirements

- App router must handle 20+ message types across 8 screens
- Managers need Config, Executor, FileManager, UserManager, Template (shared across multiple managers)
- SetData/SetError pattern consistent; data structures: Site, VhostInfo, DBInfo, CertInfo, WorkerStatus, BackupInfo
