# Brainstorm: Service Status Bar

## Problem Statement

JuiScript TUI lacks at-a-glance visibility into LEMP service health. Users must navigate to Services screen to check if nginx/php/mariadb are running. Need persistent status indicator visible across all screens.

## Requirements

- **Position**: Dedicated bar below header, above content (every screen)
- **Data**: Service name + colored status dot + memory usage + PID
- **Refresh**: Fetch once on screen navigation (no auto-poll)
- **Failed state**: Color-only indication (red dot = failed, gray circle = inactive)
- **Services shown**: nginx, php-fpm (all versions), mariadb, redis

## Evaluated Approaches

### A. New `ServiceStatusBar` Component ✅ CHOSEN

**How**: Create `internal/tui/components/service-status-bar.go` — horizontal layout component. Inject into `App` struct, render between header and content in `App.View()`.

**Pros**:
- Single Responsibility — one component, one job
- No modification to existing ServicePanel or Header
- Clean integration point in App.View()
- Easy to test independently

**Cons**:
- One more component file (minimal cost)

### B. Extend Existing ServicePanel ❌

**How**: Add `horizontal` and `showMemory` modes to existing ServicePanel.

**Pros**: DRY — reuse existing code
**Cons**: Mode flags add complexity. ServicePanel is designed for dashboard vertical layout. Mixing concerns.

### C. Embed in Header ❌

**How**: Add service data directly to Header component.

**Pros**: No new files
**Cons**: Violates Single Responsibility. Header becomes bloated. Hard to test service rendering separately.

## Final Solution: New ServiceStatusBar

### Component Design

```
┌──────────────────────────────────────────────────┐
│  juiscript  > Dashboard                          │  ← Header
│  ● nginx 45MB  │  ● php8.3 32MB  │  ● mariadb…  │  ← ServiceStatusBar (NEW)
│                                                  │
│  > Sites Management                              │  ← Content
│    Nginx Configuration                           │
│    ...                                           │
│                                                  │
│ [q] quit  [↑↓] navigate  [enter] select         │  ← StatusBar
└──────────────────────────────────────────────────┘
```

### Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/components/service-status-bar.go` | CREATE | New component with horizontal service status rendering |
| `internal/tui/app.go` | MODIFY | Add `serviceBar` field, wire into View() and Update() |
| `internal/tui/messages.go` (or inline) | MODIFY | Add `ServiceStatusMsg` for async fetch results |

### Data Flow

```
App.Init() / NavigateMsg
  → tea.Cmd: call service.Manager.ListAll(ctx)
  → returns ServiceStatusMsg{statuses []service.Status}
  → App.Update() receives msg → serviceBar.SetServices(statuses)
  → App.View() renders serviceBar between header and content
```

### Visual Format

```
Active:   ● nginx 45MB    (green dot, white text)
Failed:   ● mariadb 0MB   (red dot, white text)
Inactive: ○ redis         (gray circle, gray text — no memory shown)
```

Separator: ` │ ` (dimmed pipe)

### Edge Cases

- **No services detected**: Show `"  ○ No services detected"` in subtle style
- **Manager unavailable** (not running as root): Show `"  ⚠ Cannot read service status"` in warning color
- **Terminal too narrow**: Truncate less important services (redis first), show `+N more`
- **Many PHP versions**: Show all detected versions from `/etc/php/`, may wrap

## Implementation Considerations

1. **`service.Manager` dependency**: App needs access to `service.Manager`. Currently App doesn't hold one — need to inject via `NewApp(mgr)` or create internally
2. **Async fetch**: `service.Manager.ListAll()` runs `systemctl` commands — must be async `tea.Cmd` to avoid blocking TUI
3. **Error handling**: If `ListAll()` fails, show warning state, don't crash
4. **Width calculation**: ServiceStatusBar needs `SetWidth()` like other components. Memory format should adapt: show MB on wide, hide on narrow
5. **Existing ServicePanel**: Keep it for future dashboard panel use. Don't delete — different purpose (vertical detailed view vs horizontal compact bar)

## Risks

| Risk | Mitigation |
|------|-----------|
| `systemctl` calls slow on startup | Async tea.Cmd, UI renders immediately with loading state |
| Status stale after service restart via TUI | Re-fetch after any service action msg (Start/Stop/Restart) |
| Needs root for systemctl | Graceful degradation — show warning, don't crash |

## Success Criteria

- [ ] Service dots visible on every screen below header
- [ ] Green/red/gray correctly reflects service state
- [ ] Memory shown for active services
- [ ] No TUI freeze during status fetch
- [ ] Works gracefully when not running as root

## Next Steps

1. Create `ServiceStatusBar` component
2. Add `ServiceStatusMsg` message type
3. Wire into `App` — Init, Update, View
4. Test with mock executor
