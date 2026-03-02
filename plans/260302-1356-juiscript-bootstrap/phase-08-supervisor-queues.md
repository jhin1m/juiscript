# Phase 08: Supervisor/Queue Workers

## Context
Laravel queue worker management via Supervisor. Each Laravel site can have one or more queue workers managed by Supervisor, with per-site config files and isolated process groups.

## Overview
- **Effort**: 3h
- **Priority**: P3
- **Status**: pending
- **Depends on**: Phase 01, Phase 02

## Key Insights
- Supervisor config per site at `/etc/supervisor/conf.d/{domain}-worker.conf`.
- Worker command: `php /home/{user}/{domain}/artisan queue:work --sleep=3 --tries=3 --max-time=3600`.
- `supervisorctl reread && supervisorctl update` after config changes.
- `supervisorctl status {program}` for monitoring.
- WordPress sites typically don't need queue workers (skip for WP project type).

## Requirements
1. Create/delete Supervisor worker configs for Laravel sites
2. Start/stop/restart workers
3. View worker status (running, stopped, fatal)
4. Configure worker params (connection, queue, processes, tries, timeout)
5. TUI screen for worker management

## Architecture

### Supervisor Package (`internal/supervisor/`)
```go
type WorkerConfig struct {
    Domain     string
    Username   string
    SitePath   string
    PHPBinary  string   // /usr/bin/php{ver}
    Connection string   // default: "redis"
    Queue      string   // default: "default"
    Processes  int      // default: 1
    Tries      int      // default: 3
    MaxTime    int      // default: 3600
    Sleep      int      // default: 3
}

type WorkerStatus struct {
    Name    string
    State   string  // RUNNING, STOPPED, FATAL
    PID     int
    Uptime  time.Duration
}

type Manager struct {
    executor system.Executor
    files    system.FileManager
    tpl      *template.Engine
}

func (m *Manager) Create(cfg WorkerConfig) error
func (m *Manager) Delete(domain string) error
func (m *Manager) Start(domain string) error
func (m *Manager) Stop(domain string) error
func (m *Manager) Restart(domain string) error
func (m *Manager) Status(domain string) (*WorkerStatus, error)
func (m *Manager) ListAll() ([]WorkerStatus, error)
```

### Supervisor Config Template
```ini
[program:{{ .Domain }}-worker]
process_name=%(program_name)s_%(process_num)02d
command={{ .PHPBinary }} {{ .SitePath }}/artisan queue:work {{ .Connection }} --queue={{ .Queue }} --sleep={{ .Sleep }} --tries={{ .Tries }} --max-time={{ .MaxTime }}
autostart=true
autorestart=true
stopasgroup=true
killasgroup=true
user={{ .Username }}
numprocs={{ .Processes }}
redirect_stderr=true
stdout_logfile=/home/{{ .Username }}/logs/worker.log
stopwaitsecs=3600
```

## Related Files
```
internal/supervisor/manager.go
internal/supervisor/manager_test.go
internal/tui/screens/supervisor.go
templates/supervisor-worker.conf.tmpl
```

## Implementation Steps

1. **WorkerConfig struct**: All configurable worker parameters with defaults
2. **Supervisor template**: Generate valid Supervisor program config
3. **Manager.Create()**: Render template, write to conf.d, `supervisorctl reread && update`
4. **Manager.Delete()**: Remove config, `supervisorctl reread && update` (auto-stops removed programs)
5. **Manager.Start/Stop/Restart()**: `supervisorctl {action} {domain}-worker:*`
6. **Manager.Status()**: Parse `supervisorctl status {program}` output
7. **Manager.ListAll()**: Parse `supervisorctl status` for all managed workers
8. **TUI supervisor screen**: Worker list with status, create/delete/restart actions
9. **Integration with site creation**: Optionally create worker during Laravel site setup

## Todo
- [ ] Supervisor worker template
- [ ] Manager CRUD
- [ ] Start/Stop/Restart via supervisorctl
- [ ] Status parsing
- [ ] TUI supervisor screen
- [ ] Integration with site create flow
- [ ] Tests

## Success Criteria
- Creating a worker starts a running Supervisor process
- Worker runs as the site user, not root
- Delete stops and removes the worker cleanly
- Status accurately reflects RUNNING/STOPPED/FATAL

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Worker consuming too much memory | Medium | `--max-time` ensures periodic restart |
| Supervisor not installed | Low | Check on startup, prompt to install |
| Worker crash loop (FATAL) | Medium | Display status prominently; cap restart attempts |

## Security Considerations
- Workers run as site user (`user={{ .Username }}`)
- `stopasgroup` + `killasgroup` ensure clean process termination
- Log files in user's home directory, not world-readable

## Next Steps
Phase 09 (Backup) is the final feature phase.
