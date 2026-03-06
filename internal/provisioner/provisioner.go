package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jhin1m/juiscript/internal/php"
	"github.com/jhin1m/juiscript/internal/system"
)

// ProgressStatus describes the current state of a package install step.
type ProgressStatus string

const (
	ProgressStarting ProgressStatus = "starting"
	ProgressDone     ProgressStatus = "done"
	ProgressError    ProgressStatus = "error"
)

// ProgressEvent is emitted during InstallSelected for TUI progress display.
type ProgressEvent struct {
	PackageName string
	Status      ProgressStatus
	Message     string
}

// InstallSummary collects the results of a batch install operation.
type InstallSummary struct {
	Results   []InstallResult
	TotalTime time.Duration
}

// Provisioner coordinates package detection and installation.
// Provides progress callbacks for TUI integration.
type Provisioner struct {
	detector  *Detector
	installer *Installer
}

// NewProvisioner creates a Provisioner with internal Detector and Installer.
func NewProvisioner(exec system.Executor, phpMgr *php.Manager) *Provisioner {
	return &Provisioner{
		detector:  NewDetector(exec),
		installer: NewInstaller(exec, phpMgr),
	}
}

// DetectAll forwards to Detector.DetectAll.
func (p *Provisioner) DetectAll(ctx context.Context) ([]PackageInfo, error) {
	return p.detector.DetectAll(ctx)
}

// InstallSelected installs the given packages sequentially.
// Calls AptUpdate once before the batch, then installs each package.
// Emits progress events via progressFn (may be nil).
// Continue-on-failure: a failed package doesn't stop remaining installs.
func (p *Provisioner) InstallSelected(ctx context.Context, names []string, progressFn func(ProgressEvent)) (*InstallSummary, error) {
	start := time.Now()
	summary := &InstallSummary{}

	emit := func(e ProgressEvent) {
		if progressFn != nil {
			progressFn(e)
		}
	}

	// Emit progress so TUI knows we're working (apt-get update can take 10-60s)
	emit(ProgressEvent{PackageName: "system", Status: ProgressStarting, Message: "Updating package lists..."})

	// Single apt-get update before batch
	if err := p.installer.AptUpdate(ctx); err != nil {
		emit(ProgressEvent{PackageName: "system", Status: ProgressError, Message: err.Error()})
		return nil, err
	}

	emit(ProgressEvent{PackageName: "system", Status: ProgressDone, Message: "Package lists updated"})

	for _, name := range names {
		// Bail early if context cancelled (e.g. user aborted)
		if err := ctx.Err(); err != nil {
			summary.TotalTime = time.Since(start)
			return summary, err
		}

		emit(ProgressEvent{PackageName: name, Status: ProgressStarting})

		result, err := p.installByName(ctx, name)
		if err != nil {
			emit(ProgressEvent{PackageName: name, Status: ProgressError, Message: err.Error()})
			// Continue-on-failure: collect result and move to next
			if result != nil {
				summary.Results = append(summary.Results, *result)
			}
			continue
		}

		emit(ProgressEvent{PackageName: name, Status: ProgressDone, Message: result.Message})
		summary.Results = append(summary.Results, *result)
	}

	summary.TotalTime = time.Since(start)
	return summary, nil
}

// installByName maps a package name to the correct install method.
// Names match PackageInfo.Name values: "nginx", "mariadb", "redis", or "php8.3" format.
func (p *Provisioner) installByName(ctx context.Context, name string) (*InstallResult, error) {
	switch name {
	case "nginx":
		return p.installer.InstallNginx(ctx)
	case "mariadb":
		return p.installer.InstallMariaDB(ctx)
	case "redis":
		return p.installer.InstallRedis(ctx)
	default:
		// PHP versions: "php8.3" → InstallPHP(ctx, "8.3")
		if strings.HasPrefix(name, "php") {
			ver := strings.TrimPrefix(name, "php")
			if !isVersionDir(ver) {
				return &InstallResult{Package: name, Status: StatusFailed, Message: "invalid PHP version"}, fmt.Errorf("invalid PHP version: %s", ver)
			}
			return p.installer.InstallPHP(ctx, ver)
		}
		return &InstallResult{Package: name, Status: StatusFailed, Message: "unknown package"}, fmt.Errorf("unknown package: %s", name)
	}
}
