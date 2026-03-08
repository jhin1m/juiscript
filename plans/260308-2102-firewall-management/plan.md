---
title: "Firewall Management (UFW + Fail2ban)"
description: "Add firewall port/IP management via UFW and Fail2ban with CLI commands and TUI screen"
status: done
priority: P1
effort: 6h
branch: main
tags: [firewall, ufw, fail2ban, cli, tui, phase-3]
created: 2026-03-08
---

# Firewall Management (UFW + Fail2ban)

## Overview

Add UFW firewall rule management and Fail2ban IP blocking with full CLI + TUI support. Follows existing manager/CLI/TUI/wiring pattern used by service, backup, ssl, etc.

## Phases

| Phase | File | Description | Effort | Status |
|-------|------|-------------|--------|--------|
| 1 | [phase-01-backend-manager.md](phase-01-backend-manager.md) | `internal/firewall/manager.go` + tests | 2h | done |
| 2 | [phase-02-cli-commands.md](phase-02-cli-commands.md) | `cmd/juiscript/cmd-firewall.go` | 1h | done |
| 3 | [phase-03-tui-screen.md](phase-03-tui-screen.md) | `internal/tui/screens/firewall.go` | 2h | done |
| 4 | [phase-04-integration-wiring.md](phase-04-integration-wiring.md) | Wire into main.go, app.go, messages, handlers | 1h | done |

## Dependencies

- `internal/system` (Executor interface for command execution)
- UFW and Fail2ban installed on target Ubuntu system
- Existing TUI component library (form, confirm, toast, spinner)

## Key Decisions

- Single `firewall.Manager` struct handles both UFW and Fail2ban (KISS)
- Whitelist approach for commands - no shell injection possible
- Port validation 1-65535, IP validation via net.ParseIP
- Parse `ufw status numbered` for structured rule data
- Parse `fail2ban-client status <jail>` for banned IPs

## Out of Scope

- Fail2ban jail configuration (editing jail.local)
- Custom UFW application profiles
- iptables direct manipulation
- IPv6-specific rules (UFW handles transparently)
