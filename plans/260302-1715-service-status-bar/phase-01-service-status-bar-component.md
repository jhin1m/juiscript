---
phase: 1
title: "ServiceStatusBar Component"
status: done
effort: 1h
---

# Phase 01: ServiceStatusBar Component

## Context

- Parent plan: [plan.md](plan.md)
- Dependencies: `internal/service` (Status struct), `internal/tui/theme` (Theme styles)
- Reference: `internal/tui/components/statusbar.go` (simple component pattern), `internal/tui/components/servicepanel.go` (status indicator logic)

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-02 |
| Description | Create horizontal service status bar component |
| Priority | P2 |
| Impl Status | done |
| Review Status | pending |

## Key Insights

- ServicePanel already has `statusIndicator()` logic (green/red/gray dots) — reuse pattern, not code (different layout)
- StatusBar component pattern: struct with theme+width, SetWidth(), View() — follow same shape
- `service.Status` struct provides: Name, Active, State, MemoryMB — all data needed
- Lipgloss horizontal join with separators for inline rendering

## Requirements

1. Horizontal bar showing all LEMP services in single line
2. Colored dot: `●` green=active, `●` red=failed, `○` gray=inactive
3. Memory in MB shown only for active services (format: `45MB`)
4. Services separated by ` │ ` (dimmed pipe)
5. SetServices(), SetWidth(), SetError(), View() methods
6. Edge: empty state shows "No services detected"
7. Edge: error state shows warning icon + message
8. Edge: narrow terminal truncates with `+N more`

## Architecture

```go
// internal/tui/components/service-status-bar.go
type ServiceStatusBar struct {
    theme    *theme.Theme
    services []service.Status
    width    int
    errMsg   string
}

func NewServiceStatusBar(t *theme.Theme) *ServiceStatusBar
func (b *ServiceStatusBar) SetServices(services []service.Status)
func (b *ServiceStatusBar) SetWidth(w int)
func (b *ServiceStatusBar) SetError(msg string)
func (b *ServiceStatusBar) View() string
```

### View() Logic

1. If `errMsg != ""` → render `"  ⚠ {errMsg}"` with `theme.WarnText`
2. If `len(services) == 0` → render `"  ○ No services detected"` with `theme.Subtitle`
3. For each service, build segment:
   - Determine dot+style via state (active→green `●`, failed→red `●`, default→gray `○`)
   - Format name: strip `-fpm` suffix from PHP services for brevity (e.g. `php8.3-fpm` → `php8.3`)
   - If active and MemoryMB > 0: append `" {N}MB"` (rounded to int)
   - If inactive: no memory shown
4. Join segments with ` │ ` separator (dimmed via `theme.Inactive`)
5. If total width exceeds `b.width - 4`, truncate from right, append `" +N more"`
6. Render with left padding (2 spaces) for alignment with content area

## Related Code Files

| File | Relevance |
|------|-----------|
| `internal/tui/components/servicepanel.go` | statusIndicator() pattern — dot + style by state |
| `internal/tui/components/statusbar.go` | Component shape: struct, SetWidth(), View() |
| `internal/service/manager.go` | `Status` struct: Name, Active, State, MemoryMB |
| `internal/tui/theme/theme.go` | OkText (green), ErrorText (red), Subtitle (gray), WarnText (amber), Inactive (muted) |

## Implementation Steps

### Step 1: Create file scaffold
- Create `internal/tui/components/service-status-bar.go`
- Package `components`, imports: `fmt`, `math`, `strings`, `service`, `theme`

### Step 2: Define struct and constructor
```go
type ServiceStatusBar struct {
    theme    *theme.Theme
    services []service.Status
    width    int
    errMsg   string
}

func NewServiceStatusBar(t *theme.Theme) *ServiceStatusBar {
    return &ServiceStatusBar{theme: t}
}
```

### Step 3: Implement setter methods
- `SetServices(svcs []service.Status)` — stores service list
- `SetWidth(w int)` — stores terminal width
- `SetError(msg string)` — stores error message (empty clears)

### Step 4: Implement formatServiceName()
- Helper: strips `-fpm` suffix from PHP service names
- `"php8.3-fpm"` → `"php8.3"`, `"nginx"` → `"nginx"`, `"redis-server"` → `"redis"`

### Step 5: Implement statusIndicator()
- Same logic as ServicePanel but returns (dot string, lipgloss.Style):
  - `"active"` → `("●", theme.OkText)`
  - `"failed"` → `("●", theme.ErrorText)`
  - default → `("○", theme.Subtitle)`

### Step 6: Implement View()
- Error state check → warning render
- Empty state check → "No services detected"
- Build segments slice, join with separator
- Width truncation logic with `+N more`

## Todo

- [ ] Create `service-status-bar.go` file
- [ ] Struct + constructor
- [ ] SetServices, SetWidth, SetError
- [ ] formatServiceName helper
- [ ] statusIndicator helper
- [ ] View() with all states
- [ ] Width truncation logic

## Success Criteria

- Component renders correct dots/colors for all 3 states
- Memory shown only for active services
- PHP names shortened (`php8.3-fpm` → `php8.3`)
- Redis name shortened (`redis-server` → `redis`)
- Empty and error states render correctly
- Narrow width triggers truncation with `+N more`

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Lipgloss width calculation off with ANSI | Low | Use lipgloss.Width() for measuring rendered strings |
| Too many PHP versions overflow | Low | Truncation logic handles gracefully |

## Security Considerations

- No user input accepted — display-only component
- Service names from trusted `service.Manager` whitelist

## Next Steps

After this phase, proceed to [Phase 02](phase-02-app-integration.md) to wire into App.
