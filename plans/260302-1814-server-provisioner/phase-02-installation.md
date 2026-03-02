# Phase 02: Package Installation

## Context

After detection (Phase 01), the installer handles apt-get operations, systemctl enable/start, and MariaDB hardening. Must be idempotent and continue-on-failure.

## Overview

Create `internal/provisioner/installer.go` with methods per package. Each checks if already installed before acting. MariaDB secured via direct SQL (not `mysql_secure_installation`). PHP delegates to existing `php.Manager`.

## Key Insights

- `DEBIAN_FRONTEND=noninteractive` + `--force-confdef/confold` for unattended install
- `DPkg::Lock::Timeout=120` handles apt lock contention (preferred over manual polling)
- MariaDB on Ubuntu 22/24 uses unix_socket auth by default — keep it (JuiScript DB manager uses socket auth)
- `apt-get update` only once per batch, not per package
- PHP install already implemented in `php.Manager.InstallVersion()` — reuse

## Requirements

1. `Installer` struct with `system.Executor` + optional `*php.Manager`
2. `NewInstaller(exec system.Executor, phpMgr *php.Manager) *Installer`
3. `AptUpdate(ctx) error` - runs `apt-get update -y` with DEBIAN_FRONTEND
4. `InstallNginx(ctx) (*InstallResult, error)` - apt install + systemctl enable/start
5. `InstallMariaDB(ctx) (*InstallResult, error)` - apt install + enable/start + SQL hardening
6. `InstallRedis(ctx) (*InstallResult, error)` - apt install + systemctl enable/start
7. `InstallPHP(ctx, version string) (*InstallResult, error)` - delegate to php.Manager
8. `InstallResult` struct: Package, Status (installed/skipped/failed), Message
9. Each method: check `isInstalled` first (idempotent), skip if already present
10. 5-minute timeout per package install

## Architecture

```go
// internal/provisioner/installer.go

type InstallStatus string
const (
    StatusInstalled InstallStatus = "installed"
    StatusSkipped   InstallStatus = "skipped"   // already installed
    StatusFailed    InstallStatus = "failed"
)

type InstallResult struct {
    Package string
    Status  InstallStatus
    Message string
}

type Installer struct {
    executor system.Executor
    phpMgr   *php.Manager    // nil-safe: checked before use
}

func NewInstaller(exec system.Executor, phpMgr *php.Manager) *Installer
func (i *Installer) AptUpdate(ctx context.Context) error
func (i *Installer) InstallNginx(ctx context.Context) (*InstallResult, error)
func (i *Installer) InstallMariaDB(ctx context.Context) (*InstallResult, error)
func (i *Installer) InstallRedis(ctx context.Context) (*InstallResult, error)
func (i *Installer) InstallPHP(ctx context.Context, version string) (*InstallResult, error)
```

### apt-get Command Template

```go
args := []string{"install", "-y",
    "-o", "Dpkg::Options::=--force-confdef",
    "-o", "Dpkg::Options::=--force-confold",
    "-o", "DPkg::Lock::Timeout=120",
    packageName,
}
// env: DEBIAN_FRONTEND=noninteractive
```

Note: since `system.Executor.Run()` doesn't support env vars, use `env` command prefix:
```go
m.executor.Run(ctx, "env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", ...)
```

### MariaDB Hardening SQL

```sql
DELETE FROM mysql.user WHERE User='';
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');
DROP DATABASE IF EXISTS test;
DELETE FROM mysql.db WHERE Db='test' OR Db='test\_%';
FLUSH PRIVILEGES;
```

Execute via: `m.executor.RunWithInput(ctx, sql, "mysql", "--user=root")`

No password change — keep unix_socket auth.

## Related Code Files

- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/system/executor.go` - Executor interface, `RunWithInput`
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/php/manager.go` - `InstallVersion()` (lines 74-114)
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/service/manager.go` - systemctl patterns

## Implementation Steps

1. Create `installer.go`:
   - Define `InstallStatus`, `InstallResult` types
   - Define `Installer` struct with executor + phpMgr
   - `isInstalled()` helper: reuse dpkg-query check (same as detector)
   - `aptInstall()` helper: common apt-get install logic with env, flags, timeout
   - `enableAndStart()` helper: systemctl enable --now
   - `AptUpdate()`: `env DEBIAN_FRONTEND=noninteractive apt-get update -y`
   - `InstallNginx()`: check → aptInstall("nginx") → enableAndStart("nginx")
   - `InstallMariaDB()`: check → aptInstall("mariadb-server") → enableAndStart("mariadb") → SQL hardening
   - `InstallRedis()`: check → aptInstall("redis-server") → enableAndStart("redis-server")
   - `InstallPHP()`: guard phpMgr != nil → delegate to phpMgr.InstallVersion()
2. Create `installer_test.go`:
   - Mock executor for all commands
   - Test: install when not installed → success result
   - Test: install when already installed → skipped result
   - Test: apt-get failure → failed result with message
   - Test: MariaDB hardening SQL execution
   - Test: PHP delegation to mock php.Manager
   - Test: AptUpdate success/failure

## Todo

- [ ] Implement `InstallResult` and `InstallStatus` types
- [ ] Implement `Installer` struct + `NewInstaller`
- [ ] Implement `isInstalled()` helper (shared with detector via function)
- [ ] Implement `aptInstall()` helper with DEBIAN_FRONTEND + lock timeout
- [ ] Implement `enableAndStart()` helper
- [ ] Implement `AptUpdate()`
- [ ] Implement `InstallNginx()`, `InstallMariaDB()`, `InstallRedis()`
- [ ] Implement MariaDB SQL hardening
- [ ] Implement `InstallPHP()` delegation
- [ ] Write unit tests (8+ test cases)

## Success Criteria

- Each Install method is idempotent (skip if installed)
- MariaDB hardened via SQL without interactive prompts
- PHP delegates to existing php.Manager (DRY)
- All testable with mock executor
- Continue-on-failure: individual failures don't block others

## Risk Assessment

- **Medium**: `env` prefix for DEBIAN_FRONTEND — verify executor handles it (test in integration). Alternative: modify Executor interface to accept env vars (YAGNI for now)
- **Low**: MariaDB SQL is idempotent (DELETE WHERE, DROP IF EXISTS)
- **Low**: systemctl enable --now is idempotent

## Security

- Root required for apt-get and systemctl (enforced at CLI entry)
- Package names hardcoded (no user input injection)
- MariaDB keeps unix_socket auth (no password stored)

## Next Steps

Phase 03 combines Detector + Installer into orchestrator with progress events.
