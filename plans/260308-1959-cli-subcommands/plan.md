---
title: "CLI Subcommands via Cobra"
description: "Add CLI subcommands for all backend managers to enable scripting and automation"
status: done
priority: P1
effort: 10h
branch: main
tags: [cli, cobra, automation, backup-cron]
created: 2026-03-08
---

# CLI Subcommands Plan

## Goal

Expose all backend managers as Cobra CLI subcommands so users can script operations (especially backup cron) without the TUI.

## Current State

- `juiscript` (no args) launches TUI
- `juiscript version` prints version
- All 9 backend managers fully implemented in `internal/`
- Backup manager already references CLI path in cron: `/usr/local/bin/juiscript backup create --domain X --type full`

## Architecture

```
cmd/juiscript/
  main.go              Refactored: shared initManagers(), rootCmd wires all groups
  cmd-site.go          site {list,create,delete,enable,disable,info}
  cmd-db.go            db {list,create,drop,user-create,user-drop,reset-password,import,export}
  cmd-ssl.go           ssl {list,obtain,revoke,renew}
  cmd-service.go       service {start,stop,restart,reload,status,list}
  cmd-php.go           php {list,install,remove}
  cmd-backup.go        backup {list,create,restore,delete,cleanup,cron-setup,cron-remove}
  cmd-queue.go         queue {list,create,delete,start,stop,restart,status}
```

### Key Decisions

1. **Shared manager init** -- Extract manager construction from `runTUI()` into `initManagers()` returning a struct. Both TUI and CLI commands use same struct.
2. **One file per command group** -- `cmd-` prefix for discoverability, kebab-case per project convention.
3. **Each file exports `func xxxCmd(mgrs *Managers) *cobra.Command`** -- Returns fully configured command tree.
4. **TUI stays default** -- `rootCmd.RunE = runTUI` unchanged. Subcommands only fire when explicitly invoked.
5. **Root check** -- `PersistentPreRunE` on rootCmd checks `os.Geteuid() == 0` (except `version`).
6. **Output** -- Human-readable tabular output via `fmt.Fprintf(os.Stdout, ...)` with aligned columns. No JSON for now (YAGNI).
7. **Exit codes** -- 0 success, 1 error. Cobra handles this via `RunE`.
8. **Zero business logic in cmd/** -- Thin wrappers that parse flags, call manager methods, format output.

### Managers Struct

```go
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

## Phases

| Phase | Scope | Effort | Files | Status |
|-------|-------|--------|-------|--------|
| 1 | Refactor main.go -- extract `initManagers()`, wire command groups, root check | 1.5h | `main.go` | DONE |
| 2 | Site + PHP commands | 2h | `cmd-site.go`, `cmd-php.go` | DONE |
| 3 | Database commands | 2h | `cmd-db.go` | DONE |
| 4 | SSL + Service commands | 1.5h | `cmd-ssl.go`, `cmd-service.go` | DONE |
| 5 | Backup + Queue commands | 3h | `cmd-backup.go`, `cmd-queue.go` | DONE |

All 5 phases completed in single implementation run.

## Unresolved Questions

- None. All manager APIs confirmed from source code analysis.
