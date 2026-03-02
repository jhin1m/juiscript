---
title: "Service Status Bar"
description: "Persistent horizontal bar showing LEMP service health below header on all TUI screens"
status: completed
priority: P2
effort: 3h
branch: main
tags: [tui, components, service-monitoring]
created: 2026-03-02
---

# Service Status Bar

## Overview

Add a persistent `ServiceStatusBar` component rendered between header and content on every TUI screen. Shows colored dots + memory for nginx, php-fpm, mariadb, redis. Fetches on navigation, re-fetches after service actions.

## Visual

```
● nginx 45MB │ ● php8.3 32MB │ ● mariadb 120MB │ ○ redis
```

Green=active, Red=failed, Gray=inactive. Memory shown only for active services.

## Phases

| # | Phase | File | Status |
|---|-------|------|--------|
| 1 | ServiceStatusBar Component | [phase-01](phase-01-service-status-bar-component.md) | done (2026-03-02) |
| 2 | App Integration | [phase-02](phase-02-app-integration.md) | done (2026-03-02) |
| 3 | Testing | [phase-03](phase-03-testing.md) | done (2026-03-02) |

## Files Changed

| File | Action |
|------|--------|
| `internal/tui/components/service-status-bar.go` | CREATE |
| `internal/tui/components/service-status-bar_test.go` | CREATE |
| `internal/tui/app.go` | MODIFY |
| `cmd/juiscript/main.go` | MODIFY |

## Dependencies

- `internal/service/manager.go` — `Manager.ListAll()` returns `[]Status`
- `internal/tui/theme/theme.go` — OkText, ErrorText, Subtitle, WarnText styles
- `internal/tui/components/statusbar.go` — reference pattern for component structure

## Key Decisions

1. New component (not extending ServicePanel) — SRP, different layout purpose
2. Async fetch via `tea.Cmd` — prevents TUI freeze on systemctl calls
3. Fetch on navigation only (no polling) — simple, predictable
4. `service.Manager` injected into `NewApp()` — clean DI
