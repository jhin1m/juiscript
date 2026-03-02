---
title: "juiscript - Go LEMP Server Management TUI"
description: "Implementation plan for a single-binary Go CLI with Bubble Tea TUI for managing LEMP stacks on Ubuntu"
status: pending
priority: P1
effort: 40h
branch: main
tags: [go, bubbletea, lemp, cli, server-management]
created: 2026-03-02
---

# juiscript Bootstrap Plan

Go CLI + Bubble Tea TUI for LEMP server management (Nginx, PHP-FPM, MariaDB, Redis) on Ubuntu 22/24. Single binary, runs as root, TOML config at `/etc/juiscript/`. Focus: Laravel & WordPress site hosting with per-site user isolation.

## Tech Stack

- **Go 1.22+** with embed for templates
- **Bubble Tea** (TUI), **Bubbles** (components), **Lip Gloss** (styling), **Huh** (forms)
- **BurntSushi/toml** for config
- **cobra** for CLI entry (subcommands: `tui`, `site add`, etc.)

## Phases

| # | Phase | Effort | Status | File |
|---|-------|--------|--------|------|
| 1 | Core Infrastructure | 6h | pending | [phase-01](phase-01-core-infrastructure.md) |
| 2 | Site Management | 5h | pending | [phase-02](phase-02-site-management.md) |
| 3 | Nginx/Vhost | 4h | pending | [phase-03](phase-03-nginx-vhost.md) |
| 4 | PHP Management | 5h | pending | [phase-04](phase-04-php-management.md) |
| 5 | Database Management | 4h | pending | [phase-05](phase-05-database-management.md) |
| 6 | SSL Management | 3h | pending | [phase-06](phase-06-ssl-management.md) |
| 7 | Service Control | 3h | pending | [phase-07](phase-07-service-control.md) |
| 8 | Supervisor/Queues | 3h | pending | [phase-08](phase-08-supervisor-queues.md) |
| 9 | Backup System | 4h | pending | [phase-09](phase-09-backup-system.md) |

## Key Decisions

- **Interface-driven**: Define interfaces for system ops (enables testing with mocks)
- **Embedded templates**: `//go:embed templates/*` for Nginx/PHP-FPM/Supervisor configs
- **Atomic config writes**: Write temp file, validate, `os.Rename`, reload service
- **Error propagation**: Return `error`, never `panic`; wrap with `fmt.Errorf("op: %w", err)`
- **Package naming**: lowercase, single-word (`nginx`, `php`, `database`, `ssl`)

## Architecture Overview

```
cmd/juiscript/main.go  ->  cobra root cmd
                            ├── tui cmd     -> bubbletea.NewProgram(tui.NewApp())
                            ├── site cmd    -> internal/site
                            └── ...

internal/
  config/     TOML load/save, validation, defaults
  system/     Exec wrapper, file ops, user mgmt (interfaces)
  template/   Embedded templates, render funcs
  tui/        App model, router, theme, screens, components
  nginx/      Vhost CRUD, enable/disable, test config
  php/        Version install, FPM pool CRUD, switch
  database/   DB/user CRUD, import/export
  ssl/        Certbot wrapper, cert status
  backup/     File + DB backup/restore, scheduling
  supervisor/ Worker config CRUD, process control
  service/    systemctl wrapper, status checks
```

## Dependencies

- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/huh`
- `github.com/spf13/cobra`
- `github.com/BurntSushi/toml`

## Risk Summary

- Root-only execution requires careful input validation & atomic operations
- PHP PPA availability on future Ubuntu versions
- certbot API changes across versions
- Large TUI state management complexity -> mitigate with stack-based navigation
