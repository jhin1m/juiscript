# Phase 09: Backup System

## Context
Backup and restore for sites including files and database. Supports full and partial backups, with optional scheduled backups via cron.

## Overview
- **Effort**: 4h
- **Priority**: P3
- **Status**: pending
- **Depends on**: Phase 01, Phase 02, Phase 05

## Key Insights
- Backup = tar.gz of site files + mysqldump of DB, stored as a single archive.
- Backup naming: `{domain}_{timestamp}.tar.gz` in `/var/backups/juiscript/`.
- Restore must handle: extract files, import DB, fix permissions.
- Scheduled backups via cron job writing to backup dir.
- Retention policy: keep N most recent backups, delete older.
- Large sites may need streaming (tar piped to gzip) to avoid memory issues.

## Requirements
1. Full backup: files + database
2. Files-only backup
3. Database-only backup
4. Restore from backup archive
5. List backups per site with size and date
6. Retention policy (keep last N backups)
7. Scheduled backups via cron
8. TUI screen for backup management

## Architecture

### Backup Package (`internal/backup/`)
```go
type BackupType string
const (
    BackupFull     BackupType = "full"
    BackupFiles    BackupType = "files"
    BackupDatabase BackupType = "database"
)

type BackupInfo struct {
    Path      string
    Domain    string
    Type      BackupType
    Size      int64
    CreatedAt time.Time
}

type Options struct {
    Domain     string
    Type       BackupType
    OutputDir  string
    Compress   bool     // default: true
}

type Manager struct {
    config   *config.Config
    executor system.Executor
    files    system.FileManager
    db       *database.Manager
}

func (m *Manager) Create(opts Options) (*BackupInfo, error)
func (m *Manager) Restore(backupPath, domain string) error
func (m *Manager) List(domain string) ([]BackupInfo, error)
func (m *Manager) Delete(backupPath string) error
func (m *Manager) Cleanup(domain string, keepLast int) error  // retention
func (m *Manager) SetupCron(domain string, schedule string) error
func (m *Manager) RemoveCron(domain string) error
```

### Backup Archive Structure
```
{domain}_{timestamp}.tar.gz
├── files/           # site directory contents
├── database.sql.gz  # DB dump (if full or database type)
└── metadata.toml    # backup metadata (domain, type, php version, timestamp)
```

### Create Flow (Full)
1. Create temp directory
2. `mysqldump --single-transaction {db} | gzip > temp/database.sql.gz`
3. `tar -czf temp/files.tar.gz -C /home/{user}/{domain} .`
4. Write metadata.toml with site info
5. `tar -czf {output}/{domain}_{ts}.tar.gz -C temp .`
6. Remove temp directory
7. Apply retention policy

### Restore Flow
1. Extract archive to temp dir
2. Read metadata.toml for site info
3. Stop site services (disable Nginx vhost)
4. Extract files to site directory, fix ownership
5. Import database dump
6. Re-enable site, reload services
7. Cleanup temp

## Related Files
```
internal/backup/manager.go
internal/backup/manager_test.go
internal/tui/screens/backup.go
```

## Implementation Steps

1. **Backup archive format**: Define tar.gz structure with metadata
2. **Manager.Create()**: Orchestrate file backup + DB dump + packaging
3. **Manager.Restore()**: Extract, restore files, import DB, fix permissions
4. **Manager.List()**: Scan backup directory, parse filenames for metadata
5. **Manager.Delete()**: Remove backup file
6. **Manager.Cleanup()**: Sort by date, keep N newest, delete rest
7. **Cron management**: Write/remove cron entry in `/etc/cron.d/juiscript-{domain}`
8. **TUI backup screen**: List backups per site, create/restore/delete actions
9. **Progress reporting**: Stream tar/gzip output for progress indication in TUI
10. **Metadata TOML**: Store enough info to restore on a different server

## Todo
- [ ] Backup archive format with metadata
- [ ] Manager.Create (full/files/database)
- [ ] Manager.Restore with permission fix
- [ ] Manager.List
- [ ] Manager.Cleanup (retention)
- [ ] Cron job setup/removal
- [ ] TUI backup screen
- [ ] Tests

## Success Criteria
- Full backup creates a valid tar.gz with files + DB dump
- Restore from backup brings site back to working state
- Retention policy keeps exactly N backups
- Cron schedule creates proper cron entry
- Backup/restore works for both Laravel and WordPress sites

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Backup during high traffic | Medium | `--single-transaction` for DB; file backup is point-in-time |
| Disk space exhaustion | High | Check available space before backup; retention policy |
| Restore overwrites current data | High | Confirm in TUI; optionally backup current state first |
| Large sites slow to backup | Medium | Stream tar+gzip; show progress |

## Security Considerations
- Backup files contain DB dumps with credentials -> restrict permissions (0600, root-owned)
- Backup directory not web-accessible
- Cron jobs run as root (needed for cross-user file access)
- Validate backup archive integrity before restore (check tar exit code)

## Next Steps
This is the final feature phase. After all phases: integration testing, documentation, release packaging.
