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
  service/              Systemd service control (future)
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
Use custom types for custom messages:
```go
type NavigateMsg struct {
    Screen string
}
```

### Model Pattern
```go
type Screen struct {
    theme *theme.Theme
    // ... fields
}

func (s *Screen) Init() tea.Cmd { return nil }
func (s *Screen) Update(msg tea.Msg) (tea.Model, tea.Cmd) { /* ... */ }
func (s *Screen) View() string { /* ... */ }
```

### Screen Routing
Use `App` as router, delegate to child models via `Update()`:
```go
case screens.NavigateMsg:
    if screen, ok := screenNames[msg.Screen]; ok {
        a.current = screen
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
