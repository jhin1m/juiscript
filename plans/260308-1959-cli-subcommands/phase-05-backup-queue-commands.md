# Phase 5: Backup + Queue Commands

## File: `cmd/juiscript/cmd-backup.go`

### Command Tree

```
juiscript backup list --domain example.com
juiscript backup create --domain example.com --type full
juiscript backup restore --path /var/backups/juiscript/example.com_20260308_140000.tar.gz --domain example.com
juiscript backup delete --path /var/backups/juiscript/example.com_20260308_140000.tar.gz
juiscript backup cleanup --domain example.com --keep 5
juiscript backup cron-setup --domain example.com --schedule "0 2 * * *"
juiscript backup cron-remove --domain example.com
```

**This is the highest-priority command group** -- backup cron already references CLI path in `SetupCron()`:
```go
cronLine := fmt.Sprintf("%s root /usr/local/bin/juiscript backup create --domain %s --type full\n", schedule, domain)
```

### Subcommand Details

**backup list --domain X**
- Calls `mgrs.Backup.List(domain)`
- Output table: PATH | SIZE | CREATED
- Size formatted via `backup.FormatSize()`

**backup create --domain X --type X**
- `--type` flag: choices full/files/database (default: full)
- Calls `mgrs.Backup.Create(ctx, backup.Options{Domain: domain, Type: backupType})`
- Print: "Backup created: /var/backups/juiscript/example.com_20260308_140000.tar.gz (4.2 MB)"

**backup restore --path X --domain X**
- Calls `mgrs.Backup.Restore(ctx, path, domain)`
- Print: "Backup restored: example.com"

**backup delete --path X**
- Calls `mgrs.Backup.Delete(path)`
- Print: "Backup deleted: /var/backups/juiscript/..."

**backup cleanup --domain X --keep N**
- `--keep` flag: required int, minimum 1
- Calls `mgrs.Backup.Cleanup(domain, keep)`
- Print: "Cleanup complete: keeping last N backups for example.com"

**backup cron-setup --domain X --schedule "X X X X X"**
- Calls `mgrs.Backup.SetupCron(domain, schedule)`
- Print: "Cron job created for: example.com (0 2 * * *)"

**backup cron-remove --domain X**
- Calls `mgrs.Backup.RemoveCron(domain)`
- Print: "Cron job removed for: example.com"

### API Notes

- BackupType is `string` type with constants: `backup.BackupFull`, `backup.BackupFiles`, `backup.BackupDatabase`
- `Create` returns `(*BackupInfo, error)` -- use BackupInfo.Path and BackupInfo.Size for output
- `List` returns sorted newest-first

---

## File: `cmd/juiscript/cmd-queue.go`

### Command Tree

```
juiscript queue list
juiscript queue create --domain example.com --username siteuser --site-path /home/siteuser/example.com --php /usr/bin/php8.3 [--connection redis] [--queue default] [--processes 1] [--tries 3] [--max-time 3600] [--sleep 3]
juiscript queue delete --domain example.com
juiscript queue start --domain example.com
juiscript queue stop --domain example.com
juiscript queue restart --domain example.com
juiscript queue status --domain example.com
```

### Subcommand Details

**queue list**
- Calls `mgrs.Super.ListAll(ctx)`
- Output table: NAME | STATE | PID | UPTIME

**queue create --domain X --username X --site-path X --php X [optional flags]**
- Build `supervisor.WorkerConfig` from flags
- Calls `mgrs.Super.Create(ctx, cfg)`
- Print: "Queue worker created for: example.com"
- Required flags: `--domain`, `--username`, `--site-path`, `--php`
- Optional with defaults: `--connection` (redis), `--queue` (default), `--processes` (1), `--tries` (3), `--max-time` (3600), `--sleep` (3)
- Manager's `applyDefaults()` handles zero values, but we set flag defaults for help text clarity

**queue delete --domain X**
- Calls `mgrs.Super.Delete(ctx, domain)`
- Print: "Queue worker deleted for: example.com"

**queue start/stop/restart --domain X**
- Calls corresponding method
- Print: "Queue worker started: example.com"

**queue status --domain X**
- Calls `mgrs.Super.Status(ctx, domain)`
- Output: Name, State, PID, Uptime

### API Notes

- WorkerConfig fields: Domain, Username, SitePath, PHPBinary, Connection, Queue, Processes, Tries, MaxTime, Sleep
- `ListAll` returns `[]WorkerStatus` -- Name, State, PID, Uptime (time.Duration)
- `Status` returns single `*WorkerStatus`

## Acceptance Criteria

- [ ] All 7 backup subcommands functional
- [ ] `backup create` works from cron (no TTY required)
- [ ] All 7 queue subcommands functional
- [ ] Queue create accepts all WorkerConfig fields as flags
- [ ] `go build` succeeds
- [ ] End-to-end: `juiscript backup cron-setup` creates cron that calls `juiscript backup create` successfully
