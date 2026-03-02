# Phase 05: Database Management

## Context
MariaDB database and user management. Each site optionally gets its own database and restricted user. Supports import/export for migration and backup use cases.

## Overview
- **Effort**: 4h
- **Priority**: P2
- **Status**: pending
- **Depends on**: Phase 01

## Key Insights
- Use MariaDB socket auth for root operations (no password needed when running as root).
- Per-site DB user gets `GRANT ALL ON {db_name}.* TO '{user}'@'localhost'`.
- `mysqldump` for export; `mysql < file.sql` for import.
- Password generation: `crypto/rand` for secure random passwords.
- Store DB credentials in site TOML (or optionally in the site's `.env` file for Laravel).

## Requirements
1. Create/drop databases
2. Create/drop database users with per-DB privileges
3. Import SQL files (plain or gzipped)
4. Export databases (plain or gzipped)
5. List databases and users
6. Reset user passwords
7. TUI screen for database management

## Architecture

### Database Package (`internal/database/`)
```go
type DBInfo struct {
    Name     string
    SizeMB   float64
    Tables   int
}

type Manager struct {
    executor system.Executor
}

func (m *Manager) CreateDB(name string) error
func (m *Manager) DropDB(name string) error
func (m *Manager) CreateUser(username, password, dbName string) error
func (m *Manager) DropUser(username string) error
func (m *Manager) ResetPassword(username, newPassword string) error
func (m *Manager) ListDBs() ([]DBInfo, error)
func (m *Manager) Import(dbName, filePath string) error    // handles .sql and .sql.gz
func (m *Manager) Export(dbName, outputPath string) error   // mysqldump | gzip
func (m *Manager) GeneratePassword(length int) (string, error)
```

### SQL Execution Pattern
```go
// All SQL executed via: mysql -u root -e "SQL"
// Using socket auth (no password) since running as root
func (m *Manager) exec(sql string) (string, error) {
    return m.executor.Run(ctx, "mysql", "-u", "root", "-e", sql)
}
```

## Related Files
```
internal/database/manager.go
internal/database/manager_test.go
internal/tui/screens/database.go
```

## Implementation Steps

1. **Manager struct**: Inject executor dependency
2. **CreateDB()**: `CREATE DATABASE IF NOT EXISTS \`{name}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`
3. **DropDB()**: Confirm name not in system DBs list; `DROP DATABASE IF EXISTS \`{name}\``
4. **CreateUser()**: Generate password, `CREATE USER`, `GRANT ALL ON {db}.*`, `FLUSH PRIVILEGES`
5. **DropUser()**: `REVOKE ALL`, `DROP USER IF EXISTS`
6. **ResetPassword()**: `ALTER USER '{user}'@'localhost' IDENTIFIED BY '{pass}'`
7. **ListDBs()**: `SHOW DATABASES` filtered (exclude information_schema, mysql, performance_schema, sys)
8. **Import()**: Detect `.gz` extension, pipe through `gunzip` if needed; `mysql {db} < file`
9. **Export()**: `mysqldump --single-transaction --quick {db} | gzip > {output}`
10. **GeneratePassword()**: `crypto/rand` based, 24 chars, mixed alphanumeric + symbols
11. **TUI database screen**: List DBs with size, actions (create/drop/import/export)

## Todo
- [ ] Manager with socket auth
- [ ] CreateDB / DropDB
- [ ] CreateUser / DropUser / ResetPassword
- [ ] ListDBs with size info
- [ ] Import (plain + gzip)
- [ ] Export with gzip
- [ ] Password generator
- [ ] TUI database screen
- [ ] Unit tests with mock executor

## Success Criteria
- Create DB + user in one operation, credentials returned
- Import 100MB+ SQL file without memory issues (streaming)
- Export produces valid gzipped dump
- System databases cannot be dropped
- Generated passwords meet complexity requirements

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| SQL injection in DB/user names | Critical | Strict regex validation: `^[a-z][a-z0-9_]{0,63}$` |
| Dropping system database | Critical | Hardcoded blocklist of system DBs |
| Large import OOM | Medium | Stream via pipe, don't load into memory |

## Security Considerations
- DB names and usernames validated with strict regex before SQL construction
- NEVER interpolate user input into SQL without validation
- Socket auth only (no passwords stored for root MariaDB access)
- Site DB passwords stored in site TOML with 0600 perms
- `--single-transaction` on export for consistency without locking

## Next Steps
Phase 06 (SSL) and Phase 09 (Backup) both interact with database operations.
