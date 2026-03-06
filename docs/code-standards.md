# Go Code Standards & Guidelines

## Package Organization

```
cmd/juiscript/          CLI entry point & Cobra commands
internal/
  config/               TOML configuration, defaults, loading
  system/               OS abstractions (Executor, FileManager, UserManager)
  template/             Template engine with embedded files
  tui/                  Bubble Tea application root
    app.go              Router model, screen navigation
    components/         Reusable TUI components (Header, StatusBar)
    screens/            Full-screen views (Dashboard, Sites, etc)
    theme/              Color scheme and styling
  site/                 Site lifecycle
  nginx/                Vhost management
  php/                  PHP-FPM management & pool CRUD
  database/             MariaDB operations (future)
  ssl/                  Let's Encrypt automation (future)
  backup/               Backup & restore (future)
  service/              Systemd service control
templates/              Embedded config templates (.tmpl)
```

## Naming Conventions

- **Packages**: lowercase, single word when possible (e.g., `config`, `system`, `tui`)
- **Functions**: MixedCase, PascalCase for exported, camelCase for unexported
- **Variables**: MixedCase, short but descriptive (e.g., `cfg`, `exec`, `fm`)
- **Constants**: UPPER_SNAKE_CASE for config values, MixedCase for Iota enums
- **Interfaces**: PascalCase, -er suffix for behavior (e.g., `Executor`, `FileManager`)
- **Concrete Types**: Suffix with impl or Impl if private (e.g., `execImpl`)

## Error Handling

### Pattern: Error Wrapping
```go
if err != nil {
    return fmt.Errorf("operation description: %w", err)
}
```
Always wrap errors with context using `%w`. This preserves the error chain for debugging.

### Pattern: Validation Errors
```go
if len(username) == 0 {
    return fmt.Errorf("username cannot be empty")
}
```
Return descriptive validation errors without wrapping.

### Pattern: Context Timeouts
```go
if _, hasDeadline := ctx.Deadline(); !hasDeadline {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
    defer cancel()
}
```
Allow callers to set timeouts, apply defaults if not set.

## Interfaces & Abstractions

Use interfaces to enable testing without system access:

```go
type Executor interface {
    Run(ctx context.Context, name string, args ...string) (string, error)
    RunWithInput(ctx context.Context, input string, name string, args ...string) (string, error)
}
```

Benefits:
- Mock implementations in tests (no root required)
- Dependency injection for flexibility
- Clear API boundaries

Implement interfaces as private types:
```go
type execImpl struct { /* ... */ }
func NewExecutor(logger *slog.Logger) Executor { return &execImpl{} }
```

## File Operations Safety

### Atomic Writes Pattern
```go
// Use FileManager.WriteAtomic for config files
if err := fm.WriteAtomic(path, data, 0640); err != nil {
    return fmt.Errorf("write config: %w", err)
}
```

Why:
- Temp file in same directory ensures atomic rename
- Prevents partial writes on crash
- Permissions set before visibility

### Symlink Safety
```go
// Always check symlink before removal
if err := fm.RemoveSymlink(path); err != nil {
    return fmt.Errorf("remove symlink: %w", err)
}
```

## Domain-Specific Patterns

### PHP Version & Pool Management
All pool operations follow safe defaults and atomic patterns:

```go
// PoolConfig with defaults applied by CreatePool
cfg := php.DefaultPool("example.com", "exampleuser", "8.3")
cfg.MaxChildren = 10  // optional override
err := mgr.CreatePool(ctx, cfg)
// → /etc/php/8.3/fpm/pool.d/example.com.conf created & FPM reloaded

// Version switching: zero-downtime atomic sequence
err := mgr.SwitchVersion(ctx, newPoolCfg, "8.2", nginxReloadFn)
// → Creates new pool, reloads Nginx, removes old pool (rollback on failure)
```

Validation patterns:
- `validateVersion("8.3")`: Checks X.Y format
- `validateDomain(domain)`: Prevents path traversal via "/" or ".."
- `isVersionDir(name)`: Validates directory name is version-like

## Testing

### Guidelines
- No tests requiring root (mock system interfaces)
- 70%+ coverage for critical packages
- Table-driven tests for multiple scenarios
- Use testing.T exclusively, no fmt.Println

### Pattern: Mock Executor for PHP Manager
```go
exec := newMockExecutor()
exec.outputs["systemctl is-active php8.3-fpm"] = "active"
exec.failOn["apt-get"] = fmt.Errorf("network error")

mgr := php.NewManager(exec, fm, tpl)
// → All system calls intercepted; no root required
```

### Pattern: Table-Driven Tests
```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    {"valid case", "input", "output", false},
    {"error case", "", "", true},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := fn(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr = %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

## Logging

Use `log/slog` for structured logging:

```go
logger.Info("event",
    "component", "nginx",
    "action", "reload",
    "duration", duration,
)

logger.Error("operation failed",
    "cmd", "systemctl",
    "error", err,
)
```

## Bubble Tea TUI Patterns

### Message Types
Use custom types for custom messages with parameters (Phase 5+):
```go
// Action message with form data
type InstallPHPMsg struct {
    Version string
}

// Result message (success)
type PHPVersionsMsg struct {
    Versions []VersionInfo
}

// Result message (error)
type PHPVersionsErrMsg struct {
    Err error
}

// Component messages
type FormSubmitMsg struct {
    Values map[string]string
}
```

### Model Pattern
```go
type Screen struct {
    theme *theme.Theme
    form *components.FormModel        // Phase 5: form input
    spinner *components.SpinnerModel  // Phase 6: loading
    confirm *components.ConfirmModel  // Phase 6: destructive actions
    // ... other fields
}

func (s *Screen) Init() tea.Cmd { return nil }
func (s *Screen) Update(msg tea.Msg) (tea.Model, tea.Cmd) { /* ... */ }
func (s *Screen) View() string { /* ... */ }
```

### Form Pattern (Phase 5)
```go
// Show form on key
case "i":  // install action
    fields := []components.FormField{
        {Key: "version", Label: "PHP Version", Type: components.FieldSelect,
         Options: versions, Default: versions[0]},
    }
    s.form = components.NewForm(s.theme, "Install PHP", fields)
    s.formActive = true

// Handle form submission
case components.FormSubmitMsg:
    s.formActive = false
    version := v.Values["version"]
    s.spinner.Start("Installing...")
    return s, func() tea.Msg {
        return InstallPHPMsg{Version: version}
    }
```

### Spinner Pattern (Phase 6)
```go
// Start spinner
spinCmd := s.spinner.Start("Installing PHP 8.3...")
actionCmd := func() tea.Msg { return InstallPHPMsg{...} }
return s, tea.Batch(spinCmd, actionCmd)

// Stop spinner (app-level)
case PHPVersionsMsg:
    a.phpScreen.StopSpinner()
    a.phpScreen.SetVersions(msg.Versions)
```

### Confirmation Pattern (Phase 6)
```go
// Show confirmation
case "r":
    s.confirm.Show("Remove PHP 8.2? Sites using it will break.")
    s.pendingAction = "remove"
    s.pendingTarget = version

// Handle confirmation result
case components.ConfirmYesMsg:
    return s, func() tea.Msg {
        return RemovePHPMsg{Version: s.pendingTarget}
    }
case components.ConfirmNoMsg:
    s.pendingAction = ""
```

### Toast Pattern (Phase 6 - App level)
```go
// In app.go
case DBOpDoneMsg:
    cmd := a.toast.Show(components.ToastSuccess, "Database created")
    return a, tea.Batch(cmd, a.fetchDatabases())

case DBOpErrMsg:
    cmd := a.toast.Show(components.ToastError, msg.Err.Error())
    return a, cmd
```

### Screen Routing
Use `App` as router, delegate to child models via `Update()`:
```go
case screens.NavigateMsg:
    if screen, ok := screenNames[msg.Screen]; ok {
        a.current = screen
    }
```

### Key Priority Order (Phases 5-6)
Screens must check component states in priority order:
```go
if s.form.Active() {
    // Form has priority - intercept all keys
} else if s.confirm.Active() {
    // Confirmation dialog active
} else if s.spinner.Active() {
    // Spinner running - only forward tick messages
} else {
    // Normal key handling for screen
}
```

## Configuration

### TOML Structure
- Top-level sections (e.g., `[general]`, `[nginx]`)
- Nested structs with `toml` tags
- camelCase field names in Go, snake_case in TOML
- Defaults always applied, config file optional

```go
type Config struct {
    General GeneralConfig `toml:"general"`
    // ...
}

type GeneralConfig struct {
    SitesRoot string `toml:"sites_root"`
}
```

## Templates

- Stored in `templates/` directory
- Embedded via `//go:embed` directive
- Parsed once at startup (fail-fast)
- Data passed as structs with exported fields
- Use `{{ .Field }}` syntax

```go
//go:embed templates/*
var templateFS embed.FS

tmpl, _ := template.ParseFS(templateFS, "templates/*.tmpl")
result, _ := tmpl.ExecuteTemplate(&buf, "nginx.vhost.tmpl", data)
```

## Code Quality

### Format & Lint
```bash
make fmt     # gofmt + go vet
```

### Testing
```bash
make test    # Run all tests
make cover   # Coverage report
```

### Build
```bash
make build        # Current platform
make build-linux  # Ubuntu server binary
```

## Documentation

- Package comments for public APIs
- Function comments for exported functions
- Inline comments for complex logic
- Update docs when changing public APIs
- README for architecture overview

## Deprecation & Breaking Changes

Mark deprecated code:
```go
// Deprecated: Use NewExecutor instead.
func NewExec() Executor { /* ... */ }
```

Document breaking changes in commit messages and CHANGELOG.
