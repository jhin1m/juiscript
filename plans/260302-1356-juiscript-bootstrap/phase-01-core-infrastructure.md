# Phase 01: Core Infrastructure

## Context
Foundation layer: config management, system command execution, template engine, and TUI skeleton. Everything else depends on this phase.

## Overview
- **Effort**: 6h
- **Priority**: P1
- **Status**: pending
- **Depends on**: Nothing (first phase)

## Key Insights
- Bubble Tea uses Elm architecture: Model/Update/View. Root model acts as router; child models per screen.
- Go `embed` package bundles templates into binary at compile time.
- Interface-driven system layer enables unit testing without root or real services.
- TOML chosen over YAML for simpler syntax, good Go support.

## Requirements
1. TOML config at `/etc/juiscript/config.toml` with sensible defaults
2. System exec wrapper with timeout, logging, stdout/stderr capture
3. Template engine loading embedded `.tmpl` files, rendering to string
4. TUI app with root model, screen router, dashboard screen, theme
5. Cobra CLI with `tui` (default) and future subcommand stubs
6. Makefile with `build`, `install`, `clean`, `test` targets

## Architecture

### Config Package (`internal/config/`)
```go
// config.go
type Config struct {
    General  GeneralConfig  `toml:"general"`
    Nginx    NginxConfig    `toml:"nginx"`
    PHP      PHPConfig      `toml:"php"`
    Database DatabaseConfig `toml:"database"`
    Backup   BackupConfig   `toml:"backup"`
}

type GeneralConfig struct {
    SitesRoot   string `toml:"sites_root"`    // default: /home
    LogFile     string `toml:"log_file"`      // default: /var/log/juiscript.log
    BackupDir   string `toml:"backup_dir"`    // default: /var/backups/juiscript
}

func Load(path string) (*Config, error)
func (c *Config) Save(path string) error
func Default() *Config
```

### System Package (`internal/system/`)
```go
// executor.go - Interface for testability
type Executor interface {
    Run(ctx context.Context, name string, args ...string) (string, error)
    RunWithInput(ctx context.Context, input string, name string, args ...string) (string, error)
}

type executor struct {
    logger *slog.Logger
}

// fileops.go
type FileManager interface {
    WriteAtomic(path string, data []byte, perm os.FileMode) error
    Symlink(target, link string) error
    Remove(path string) error
    Exists(path string) bool
}

// usermgmt.go
type UserManager interface {
    Create(username, homeDir string) error
    Delete(username string) error
    Exists(username string) bool
}
```

### Template Package (`internal/template/`)
```go
//go:embed templates/*
var templateFS embed.FS

type Engine struct {
    templates *template.Template
}

func New() (*Engine, error)                              // parse all embedded templates
func (e *Engine) Render(name string, data any) (string, error) // render template to string
```

### TUI Package (`internal/tui/`)
```go
// app.go - Root model
type Screen int
const (
    ScreenDashboard Screen = iota
    ScreenSites
    ScreenNginx
    // ...
)

type App struct {
    current    Screen
    screens    map[Screen]tea.Model
    width      int
    height     int
    theme      *theme.Theme
}

// theme/theme.go - Lip Gloss styles
type Theme struct {
    Primary    lipgloss.Style
    Secondary  lipgloss.Style
    Error      lipgloss.Style
    Success    lipgloss.Style
    Border     lipgloss.Style
    Title      lipgloss.Style
}
```

## Related Files
```
cmd/juiscript/main.go
internal/config/config.go
internal/config/config_test.go
internal/system/executor.go
internal/system/fileops.go
internal/system/usermgmt.go
internal/tui/app.go
internal/tui/theme/theme.go
internal/tui/screens/dashboard.go
internal/tui/components/header.go
internal/tui/components/statusbar.go
internal/template/engine.go
templates/nginx-vhost.conf.tmpl
templates/php-fpm-pool.conf.tmpl
templates/supervisor-worker.conf.tmpl
go.mod
Makefile
```

## Implementation Steps

1. **Init Go module**: `go mod init github.com/jhin1m/juiscript`
2. **Create directory structure**: All `internal/` packages, `cmd/`, `templates/`
3. **Config package**: Define structs, `Load()`, `Save()`, `Default()` with TOML tags; write tests with temp files
4. **System executor**: Implement `Executor` interface using `exec.CommandContext`; log command + duration + exit code; capture combined output
5. **System fileops**: `WriteAtomic` using `os.CreateTemp` + `os.Rename` in same dir; `Symlink`, `Remove`, `Exists`
6. **System usermgmt**: `useradd -m -d {home} -s /bin/bash {user}`; `userdel -r {user}`; check `/etc/passwd`
7. **Template engine**: Embed `templates/` dir; `template.ParseFS` on init; `Render()` executes named template into `bytes.Buffer`
8. **Create stub templates**: Nginx vhost, PHP-FPM pool, Supervisor worker (minimal placeholders)
9. **TUI theme**: Define color palette, styles for title/border/primary/error/success
10. **TUI root model**: Implement `App` with `Init()`, `Update()`, `View()`; handle window size, key routing, screen switching
11. **Dashboard screen**: Show system info stub (hostname, uptime, service statuses placeholder)
12. **TUI components**: Header (app title + navigation), StatusBar (key hints)
13. **Cobra CLI**: Root command, `tui` subcommand (default), version flag
14. **Makefile**: `build` (go build with ldflags for version), `install` (copy to /usr/local/bin), `test`, `clean`

## Todo
- [ ] `go mod init` + add dependencies
- [ ] Config package with tests
- [ ] System executor with context timeout
- [ ] System fileops with atomic write
- [ ] System usermgmt
- [ ] Template engine with embed
- [ ] Stub config templates (nginx, php-fpm, supervisor)
- [ ] TUI theme definition
- [ ] TUI root model + screen router
- [ ] Dashboard screen (stub)
- [ ] Header + StatusBar components
- [ ] Cobra CLI setup
- [ ] Makefile

## Success Criteria
- `make build` produces single binary
- Binary launches TUI with dashboard screen
- Config loads from TOML, falls back to defaults
- System executor runs a command and returns output
- Template engine renders a stub template with data
- All unit tests pass

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Bubble Tea learning curve | Medium | Follow official examples, keep screens simple |
| Template syntax errors at runtime | Low | Parse all templates at init, fail fast |
| Config file permissions | Medium | Ensure 0600 perms on config with sensitive data |

## Security Considerations
- Config file may contain DB passwords later -> enforce 0600 permissions
- System executor must NEVER use shell expansion (`sh -c`); always pass args separately
- Validate all template data before rendering

## Next Steps
Phase 02 (Site Management) builds directly on system executor, fileops, usermgmt, and template engine from this phase.
