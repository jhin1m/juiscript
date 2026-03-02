# Phase 01: Package Detection

## Context

JuiScript needs to detect which LEMP packages are installed on the system. This forms the foundation for the setup checklist and idempotent installation.

## Overview

Create `internal/provisioner/detector.go` with a `Detector` struct that uses `system.Executor` to query dpkg and filesystem for package status. Must be fully testable with mock executor.

## Key Insights

- `dpkg-query -W --showformat='${Status}' <pkg>` returns `install ok installed` for installed packages, exit code 1 if not found
- PHP detection reuses existing `/etc/php/` directory scan pattern from `service/manager.go`
- Version extraction: `dpkg-query -W --showformat='${Version}' <pkg>`

## Requirements

1. `PackageInfo` struct: Name (internal key), DisplayName (TUI label), Package (apt name), Installed (bool), Version (string)
2. `Detector` struct with `system.Executor` field
3. `NewDetector(exec system.Executor) *Detector`
4. `DetectAll(ctx) ([]PackageInfo, error)` - checks nginx, mariadb-server, redis-server, PHP
5. PHP detection: scan `/etc/php/` dirs (reuse `isVersionDir` pattern from `php/manager.go`)
6. Individual `isInstalled(ctx, pkgName) bool` helper
7. Version extraction via `dpkg-query -W --showformat='${Version}'`

## Architecture

```go
// internal/provisioner/detector.go

type PackageInfo struct {
    Name        string // "nginx", "mariadb", "redis", "php"
    DisplayName string // "Nginx", "MariaDB", "Redis", "PHP 8.3"
    Package     string // apt package name: "nginx", "mariadb-server", "redis-server"
    Installed   bool
    Version     string // e.g. "1.24.0-2ubuntu1"
}

type Detector struct {
    executor system.Executor
}

func NewDetector(exec system.Executor) *Detector
func (d *Detector) DetectAll(ctx context.Context) ([]PackageInfo, error)
func (d *Detector) isInstalled(ctx context.Context, pkg string) (bool, string)
```

## Related Code Files

- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/system/executor.go` - Executor interface
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/service/manager.go` - `detectPHPVersions()` pattern (lines 223-242)
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/php/manager.go` - `isVersionDir()` pattern (lines 222-238)

## Implementation Steps

1. Create `internal/provisioner/` package directory
2. Create `detector.go`:
   - Define `PackageInfo` struct
   - Define `Detector` struct with `system.Executor`
   - `NewDetector()` constructor
   - `isInstalled()`: run `dpkg-query -W --showformat='${Status}\n${Version}' <pkg>`, parse output
   - `DetectAll()`: iterate static packages (nginx, mariadb-server, redis-server) + scan `/etc/php/` for PHP versions
   - For PHP: each detected version dir becomes a PackageInfo with DisplayName "PHP X.Y"
   - For PHP not installed: single entry "PHP" with Installed=false, Package="" (handled by installer via php.Manager)
3. Create `detector_test.go`:
   - Mock executor returning success/failure for dpkg-query
   - Test: all installed → all PackageInfo.Installed=true
   - Test: none installed → all false
   - Test: partial (nginx yes, mariadb no)
   - Test: PHP versions detected from /etc/php/ scan
   - Test: dpkg-query error handling

## Todo

- [ ] Create `internal/provisioner/` directory
- [ ] Implement `PackageInfo` struct
- [ ] Implement `Detector` struct + `NewDetector`
- [ ] Implement `isInstalled()` with dpkg-query
- [ ] Implement `DetectAll()` with static + PHP scan
- [ ] Write unit tests (6+ test cases, table-driven)
- [ ] Verify mock executor pattern matches existing tests

## Success Criteria

- `DetectAll()` returns correct PackageInfo for all 4 services
- PHP versions detected from filesystem scan
- 100% testable without root (mock executor)
- Follows existing Manager/Executor patterns

## Risk Assessment

- **Low**: dpkg-query is stable across Ubuntu 22/24
- **Low**: /etc/php/ scan already proven in service/manager.go
- **None**: No system mutation in detection phase

## Security

- No privilege escalation needed (dpkg-query runs as any user)
- No user input in commands (package names hardcoded)

## Next Steps

Phase 02 depends on PackageInfo struct defined here for install targeting.
