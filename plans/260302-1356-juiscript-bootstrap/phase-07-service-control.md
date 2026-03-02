# Phase 07: Service Control

## Context
Unified interface for managing LEMP stack services (Nginx, PHP-FPM, MariaDB, Redis) via systemctl. Provides start/stop/restart/status operations and health monitoring.

## Overview
- **Effort**: 3h
- **Priority**: P2
- **Status**: done
- **Depends on**: Phase 01

## Key Insights
- All LEMP services managed via systemd on Ubuntu 22/24.
- PHP-FPM has per-version service names: `php8.3-fpm`, `php8.2-fpm`.
- `systemctl is-active {service}` returns "active"/"inactive"/"failed".
- `systemctl show {service} --property=ActiveState,SubState,MainPID,MemoryCurrent` for detailed status.
- Batch operations useful: restart all PHP-FPM versions at once.

## Requirements
1. Start/stop/restart/reload any LEMP service
2. Get service status (active, inactive, failed) with details
3. List all managed services with status overview
4. Health check: verify all critical services are running
5. TUI dashboard integration: service status panel

## Architecture

### Service Package (`internal/service/`)
```go
type ServiceName string
const (
    ServiceNginx   ServiceName = "nginx"
    ServiceMariaDB ServiceName = "mariadb"
    ServiceRedis   ServiceName = "redis-server"
)

// PHP-FPM uses dynamic names: "php8.3-fpm", "php8.2-fpm"
func PHPFPMService(version string) ServiceName {
    return ServiceName(fmt.Sprintf("php%s-fpm", version))
}

type Status struct {
    Name       ServiceName
    Active     bool
    State      string   // "active", "inactive", "failed"
    SubState   string   // "running", "dead", "exited"
    PID        int
    MemoryMB   float64
    Uptime     time.Duration
}

type Manager struct {
    executor system.Executor
}

func (m *Manager) Start(name ServiceName) error
func (m *Manager) Stop(name ServiceName) error
func (m *Manager) Restart(name ServiceName) error
func (m *Manager) Reload(name ServiceName) error
func (m *Manager) Status(name ServiceName) (*Status, error)
func (m *Manager) IsActive(name ServiceName) bool
func (m *Manager) ListAll() ([]Status, error)         // all managed services
func (m *Manager) HealthCheck() ([]Status, error)     // check critical services
```

## Related Files
```
internal/service/manager.go
internal/service/manager_test.go
internal/tui/screens/services.go
internal/tui/components/servicepanel.go   # reusable status panel for dashboard
```

## Implementation Steps

1. **Manager struct**: Inject executor
2. **Start/Stop/Restart/Reload**: `systemctl {action} {service}`, check exit code
3. **Status()**: Parse `systemctl show` properties into Status struct
4. **IsActive()**: Quick check via `systemctl is-active`
5. **ListAll()**: Detect installed services (Nginx, MariaDB, Redis, all PHP-FPM versions)
6. **HealthCheck()**: Check Nginx + MariaDB + at least one PHP-FPM are active
7. **Service panel component**: Compact status display for dashboard (green/red dots)
8. **TUI services screen**: Full service list with actions (start/stop/restart)
9. **PHP-FPM version detection**: Scan `/etc/php/` for installed versions

## Todo
- [x] Service manager with systemctl wrapper
- [x] Start/Stop/Restart/Reload
- [x] Status parsing from systemctl show
- [x] ListAll with auto-detection
- [x] HealthCheck
- [x] Service panel component for dashboard
- [x] TUI services screen
- [x] Tests

## Success Criteria
- Can start/stop/restart any LEMP service
- Status accurately reflects service state, PID, memory
- Health check detects when a critical service is down
- Dashboard shows live service status

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Stopping Nginx kills all sites | High | Confirm action in TUI; warn about impact |
| Service name differs across Ubuntu versions | Low | Verify service exists before operating |

## Security Considerations
- Only operate on known service names (whitelist approach)
- Validate service name against injection (no shell metacharacters)
- Log all service control actions

## Next Steps
Service control is consumed by all other phases (Nginx reload, PHP-FPM reload, etc.). Extracted as shared module.
