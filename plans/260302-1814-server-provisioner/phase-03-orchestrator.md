# Phase 03: Provisioner Orchestrator

## Context

Combines Detector and Installer into a single facade. Provides progress callback for TUI integration. This is the API surface the TUI screen will call.

## Overview

Create `internal/provisioner/provisioner.go` — thin orchestrator that coordinates detection, apt-update-once, and sequential installation with progress events.

## Key Insights

- Single `apt-get update` before batch install (not per package)
- Progress callback (`func(ProgressEvent)`) lets TUI update spinner/status per package
- Sequential install (not parallel) — apt-get doesn't support concurrent runs
- Continue-on-failure: collect all results, report summary

## Requirements

1. `Provisioner` struct: Detector + Installer
2. `NewProvisioner(exec system.Executor, phpMgr *php.Manager) *Provisioner`
3. `DetectAll(ctx) ([]PackageInfo, error)` — forward to Detector
4. `ProgressEvent`: PackageName, Status (starting/done/error), Message
5. `InstallSummary`: Results []InstallResult, TotalTime
6. `InstallSelected(ctx, names []string, progressFn func(ProgressEvent)) (*InstallSummary, error)`
   - Calls AptUpdate once
   - For each name: emit starting event → install → emit done/error event
   - Maps name to correct Install method (nginx→InstallNginx, etc.)
   - Returns summary with all results

## Architecture

```go
// internal/provisioner/provisioner.go

type ProgressStatus string
const (
    ProgressStarting ProgressStatus = "starting"
    ProgressDone     ProgressStatus = "done"
    ProgressError    ProgressStatus = "error"
)

type ProgressEvent struct {
    PackageName string
    Status      ProgressStatus
    Message     string
}

type InstallSummary struct {
    Results   []InstallResult
    TotalTime time.Duration
}

type Provisioner struct {
    detector  *Detector
    installer *Installer
}

func NewProvisioner(exec system.Executor, phpMgr *php.Manager) *Provisioner
func (p *Provisioner) DetectAll(ctx context.Context) ([]PackageInfo, error)
func (p *Provisioner) InstallSelected(ctx context.Context, names []string, progressFn func(ProgressEvent)) (*InstallSummary, error)
```

### Name-to-Method Mapping

```go
switch name {
case "nginx":   result, err = p.installer.InstallNginx(ctx)
case "mariadb": result, err = p.installer.InstallMariaDB(ctx)
case "redis":   result, err = p.installer.InstallRedis(ctx)
default:        // treat as PHP version: "php8.3" → InstallPHP(ctx, "8.3")
    if strings.HasPrefix(name, "php") {
        ver := strings.TrimPrefix(name, "php")
        result, err = p.installer.InstallPHP(ctx, ver)
    }
}
```

## Related Code Files

- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/provisioner/detector.go` (Phase 01)
- `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/provisioner/installer.go` (Phase 02)

## Implementation Steps

1. Create `provisioner.go`:
   - Define `ProgressStatus`, `ProgressEvent`, `InstallSummary` types
   - Define `Provisioner` struct
   - `NewProvisioner()`: creates Detector + Installer internally
   - `DetectAll()`: forward to detector
   - `InstallSelected()`: apt update → loop names → map to install method → emit progress → collect results
2. No separate test file needed — logic is thin delegation. Integration tested via Phase 01+02 mocks.

## Todo

- [ ] Define progress event types
- [ ] Implement `Provisioner` struct + constructor
- [ ] Implement `DetectAll()` forwarding
- [ ] Implement `InstallSelected()` with progress callbacks
- [ ] Implement name-to-method mapping

## Success Criteria

- `apt-get update` called exactly once per `InstallSelected` invocation
- Progress events emitted before and after each package
- Failed package doesn't stop remaining installations
- Summary accurately reflects all results

## Risk Assessment

- **None**: Thin orchestrator, no new system calls
- **Low**: Name mapping must cover all valid PackageInfo.Name values

## Security

- Inherits from Installer (Phase 02) — no additional surface

## Next Steps

Phase 04 consumes ProgressEvent in TUI to show real-time install status.
