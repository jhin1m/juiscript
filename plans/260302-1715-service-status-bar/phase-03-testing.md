---
phase: 3
title: "Testing"
status: done
effort: 0.5h
---

# Phase 03: Testing

## Context

- Parent plan: [plan.md](plan.md)
- Depends on: [Phase 01](phase-01-service-status-bar-component.md), [Phase 02](phase-02-app-integration.md)
- Reference: `internal/service/manager_test.go` (mock executor pattern), `internal/database/manager_test.go` (table-driven tests)

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-02 |
| Description | Unit tests for ServiceStatusBar component + regression check |
| Priority | P2 |
| Impl Status | done |
| Review Status | pending |

## Key Insights

- ServiceStatusBar is a pure view component — no I/O, no mocks needed
- Test by constructing Status slices and asserting View() output contains expected strings
- Table-driven tests per project pattern
- No need to test App integration separately (covered by existing test patterns)

## Requirements

1. Test all service states: active, failed, inactive/stopped
2. Test empty services list → "No services detected"
3. Test error state → warning message
4. Test mixed states (some active, some failed, some inactive)
5. Test memory display: shown for active, hidden for inactive
6. Test PHP name formatting: `php8.3-fpm` → `php8.3`
7. Test redis name formatting: `redis-server` → `redis`
8. Test narrow width truncation
9. Run `make test` to verify no regressions

## Architecture

```go
// internal/tui/components/service-status-bar_test.go
package components

func TestServiceStatusBar_EmptyServices(t *testing.T)
func TestServiceStatusBar_ActiveServices(t *testing.T)
func TestServiceStatusBar_FailedServices(t *testing.T)
func TestServiceStatusBar_InactiveServices(t *testing.T)
func TestServiceStatusBar_MixedStates(t *testing.T)
func TestServiceStatusBar_ErrorState(t *testing.T)
func TestServiceStatusBar_NarrowWidth(t *testing.T)
func TestServiceStatusBar_NameFormatting(t *testing.T)
```

## Related Code Files

| File | Relevance |
|------|-----------|
| `internal/tui/components/service-status-bar.go` | Component under test |
| `internal/service/manager.go` | `Status` struct used in test data |
| `internal/tui/theme/theme.go` | Theme required for constructor |

## Implementation Steps

### Step 1: Create test file
- `internal/tui/components/service-status-bar_test.go`
- Package `components` (same package for access to unexported helpers if needed)

### Step 2: Helper function for test service statuses
```go
func activeService(name service.ServiceName, memMB float64) service.Status {
    return service.Status{Name: name, Active: true, State: "active", MemoryMB: memMB}
}
func failedService(name service.ServiceName) service.Status {
    return service.Status{Name: name, Active: false, State: "failed"}
}
func inactiveService(name service.ServiceName) service.Status {
    return service.Status{Name: name, Active: false, State: "inactive"}
}
```

### Step 3: Test empty services
```go
func TestServiceStatusBar_EmptyServices(t *testing.T) {
    bar := NewServiceStatusBar(theme.New())
    bar.SetWidth(80)
    view := bar.View()
    if !strings.Contains(view, "No services detected") {
        t.Errorf("expected 'No services detected', got: %s", view)
    }
}
```

### Step 4: Test active services with memory
- Set services with nginx (45MB), php8.3-fpm (32MB), mariadb (120MB)
- Assert View() contains "nginx", "45MB", "php8.3", "32MB", "mariadb", "120MB"
- Assert contains "●" (filled dot)

### Step 5: Test failed services
- Set mariadb as failed
- Assert View() contains "●" (red dot rendered — can check string presence)
- Assert View() contains "mariadb"

### Step 6: Test inactive services
- Set redis-server as inactive
- Assert View() contains "○" (hollow dot)
- Assert View() contains "redis" (shortened)
- Assert View() does NOT contain "MB" for redis

### Step 7: Test mixed states
- Active nginx, failed mariadb, inactive redis, active php8.3-fpm
- Assert all names present
- Assert memory only for active services

### Step 8: Test error state
```go
func TestServiceStatusBar_ErrorState(t *testing.T) {
    bar := NewServiceStatusBar(theme.New())
    bar.SetWidth(80)
    bar.SetError("Cannot read service status")
    view := bar.View()
    if !strings.Contains(view, "Cannot read service status") {
        t.Errorf("expected error message in view")
    }
}
```

### Step 9: Test narrow width truncation
- Set 5 services, width=40
- Assert View() contains "+N more" or truncated output

### Step 10: Test name formatting
- Table-driven: `{"php8.3-fpm" → "php8.3", "redis-server" → "redis", "nginx" → "nginx", "mariadb" → "mariadb"}`

### Step 11: Run existing tests
```bash
cd /Users/jhin1m/Desktop/ducanh-project/juiscript && make test
```
- Verify no regressions from NewApp signature change
- Fix any callers if needed (likely none besides main.go)

## Todo

- [ ] Create test file
- [ ] Test helpers for building Status structs
- [ ] Test empty services
- [ ] Test active services with memory
- [ ] Test failed services
- [ ] Test inactive services
- [ ] Test mixed states
- [ ] Test error state
- [ ] Test narrow width truncation
- [ ] Test name formatting
- [ ] Run `make test` — all pass

## Success Criteria

- All new tests pass
- All existing tests pass (no regressions)
- Coverage of all View() branches: empty, error, active, failed, inactive, truncation
- Table-driven where applicable per code standards

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| ANSI codes in View() make string matching fragile | Low | Use `strings.Contains` on key text, not exact match |
| NewApp() signature change breaks existing tests | Low | Grep for NewApp calls, update to pass nil |

## Security Considerations

- Tests use no real systemctl — pure unit tests with constructed data
- No root required

## Next Steps

After all phases complete, update plan.md status to `completed`. Run full `make test` + `make fmt` for final validation.
