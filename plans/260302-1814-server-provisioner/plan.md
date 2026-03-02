---
title: "Server Provisioner"
description: "Auto-detect and install LEMP packages on fresh Ubuntu VPS via TUI checklist"
status: pending
priority: P1
effort: 6h
branch: feat/server-provisioner
tags: [provisioner, lemp, tui, setup]
created: 2026-03-02
---

# Server Provisioner

Auto-detect missing LEMP packages and install them via interactive TUI checklist. Enables JuiScript to work on fresh Ubuntu VPS without manual LEMP setup.

## Phase Overview

| # | Phase | File | Effort | Status |
|---|-------|------|--------|--------|
| 01 | Package Detection | [phase-01-detection.md](phase-01-detection.md) | 1h | done |
| 02 | Package Installation | [phase-02-installation.md](phase-02-installation.md) | 1.5h | done |
| 03 | Provisioner Orchestrator | [phase-03-orchestrator.md](phase-03-orchestrator.md) | 0.5h | pending |
| 04 | TUI Setup Screen | [phase-04-tui-setup.md](phase-04-tui-setup.md) | 2h | pending |
| 05 | App Integration | [phase-05-app-integration.md](phase-05-app-integration.md) | 1h | pending |

## Architecture

```
internal/provisioner/
  detector.go        # PackageInfo + Detector (dpkg-query)
  detector_test.go   # Mock executor tests
  installer.go       # Installer (apt-get, systemctl, SQL)
  installer_test.go  # Mock executor tests
  provisioner.go     # Orchestrator (Detector + Installer + progress)

internal/tui/screens/
  setup.go           # TUI checklist screen (state machine)

internal/tui/app.go       # +ScreenSetup, +setupScreen, +SetupDoneMsg
internal/tui/screens/dashboard.go  # +warning banner, +'s' key
```

## Dependencies

- `system.Executor` interface (existing)
- `php.Manager.InstallVersion()` (existing, reuse for PHP)
- `bubbles/spinner` (already imported)
- No new Go modules required

## Key Decisions

- `dpkg-query -W --showformat='${Status}'` for detection (reliable)
- MariaDB hardening via SQL, not `mysql_secure_installation`
- Keep unix_socket auth for root (matches existing DB manager)
- `DPkg::Lock::Timeout=120` for apt lock handling
- PHP delegates to existing `php.Manager` (DRY)
- Continue-on-failure with summary (not fail-fast)

## Validation Summary

**Validated:** 2026-03-02
**Questions asked:** 6

### Confirmed Decisions
- **PHP version selection**: Show dropdown in setup screen (7.4, 8.0, 8.1, 8.2, 8.3, 8.4)
- **Env vars for apt-get**: Use `env` command prefix (no Executor interface change)
- **Dashboard UX**: Both warning banner + menu item "Setup" (key "9")
- **MariaDB auth**: Set root password + keep unix_socket dual auth
- **Key binding**: Use "9" key on dashboard (follows 1-8 numbering)

### Action Items (update phase files before implementing)
- [ ] Phase 02: Add root password generation + `ALTER USER` SQL for MariaDB dual auth
- [ ] Phase 04: Add PHP version dropdown/selector in checklist state when PHP is selected
- [ ] Phase 04: PHP versions list: 7.4, 8.0, 8.1, 8.2, 8.3, 8.4
- [ ] Phase 05: Dashboard menu item "Setup" as item 9 + warning banner (both)
- [ ] Phase 05: Key "9" for setup navigation (not 's')
