# Phase 1: Refactor main.go -- Extract Manager Init

## Objective

Extract manager construction into a shared function so both TUI and CLI subcommands can use the same managers.

## Changes to `cmd/juiscript/main.go`

### 1. Add `Managers` struct

```go
// Managers holds all backend managers shared between TUI and CLI commands.
type Managers struct {
    Cfg       *config.Config
    Logger    *slog.Logger
    Site      *site.Manager
    DB        *database.Manager
    SSL       *ssl.Manager
    Backup    *backup.Manager
    Super     *supervisor.Manager
    Service   *service.Manager
    PHP       *php.Manager
    Nginx     *nginx.Manager
}
```

### 2. Extract `initManagers()` function

Move logger setup + config load + all `NewManager()` calls from `runTUI()` into:

```go
func initManagers() (*Managers, error) {
    // Logger setup (file or discard)
    // Config load with fallback
    // All NewManager() calls (same order as current runTUI)
    // Return &Managers{...}
}
```

### 3. Simplify `runTUI()`

```go
func runTUI(cmd *cobra.Command, args []string) error {
    mgrs, err := initManagers()
    if err != nil {
        return err
    }
    app := tui.NewApp(mgrs.Cfg, tui.AppDeps{...})
    p := tea.NewProgram(app, tea.WithAltScreen())
    _, err = p.Run()
    return err
}
```

### 4. Root check via `PersistentPreRunE`

Add to rootCmd:

```go
rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
    // Skip root check for "version" subcommand
    if cmd.Name() == "version" {
        return nil
    }
    if os.Geteuid() != 0 {
        return fmt.Errorf("juiscript requires root privileges, run with sudo")
    }
    return nil
}
```

### 5. Wire all command groups

```go
mgrs, err := initManagers()
// handle err

rootCmd.AddCommand(versionCmd())
rootCmd.AddCommand(siteCmd(mgrs))
rootCmd.AddCommand(dbCmd(mgrs))
rootCmd.AddCommand(sslCmd(mgrs))
rootCmd.AddCommand(serviceCmd(mgrs))
rootCmd.AddCommand(phpCmd(mgrs))
rootCmd.AddCommand(backupCmd(mgrs))
rootCmd.AddCommand(queueCmd(mgrs))
```

**Note:** `initManagers()` is called in `main()` before `rootCmd.Execute()`. Manager init happens once regardless of which subcommand runs.

### 6. Update `runTUI()` to accept managers

Change signature to use closure or accept managers:

```go
// Option A: Closure in main()
rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
    return runTUI(mgrs)
}

// runTUI now takes managers directly
func runTUI(mgrs *Managers) error {
    app := tui.NewApp(mgrs.Cfg, tui.AppDeps{...})
    ...
}
```

## Acceptance Criteria

- [ ] `juiscript` still launches TUI (default behavior unchanged)
- [ ] `juiscript version` still works (no root required)
- [ ] Non-root users see clear error for any other command
- [ ] Manager construction happens exactly once
- [ ] All existing tests pass
- [ ] `go build` succeeds
