package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jhin1m/juiscript/internal/php"
	"github.com/jhin1m/juiscript/internal/system"
)

// InstallStatus describes the outcome of a package install attempt.
type InstallStatus string

const (
	StatusInstalled InstallStatus = "installed"
	StatusSkipped   InstallStatus = "skipped" // already installed
	StatusFailed    InstallStatus = "failed"
)

// InstallResult captures the outcome of a single package install.
type InstallResult struct {
	Package string
	Status  InstallStatus
	Message string
}

// installTimeout is the max time for a single apt-get install.
const installTimeout = 5 * time.Minute

// Installer handles apt-get operations, systemctl enable/start, and MariaDB hardening.
// Each method is idempotent: skips if already installed.
type Installer struct {
	executor system.Executor
	phpMgr   *php.Manager // nil-safe: checked before use
}

// NewInstaller creates an Installer with the given executor and optional PHP manager.
func NewInstaller(exec system.Executor, phpMgr *php.Manager) *Installer {
	return &Installer{executor: exec, phpMgr: phpMgr}
}

// AptUpdate runs apt-get update with noninteractive frontend.
func (i *Installer) AptUpdate(ctx context.Context) error {
	_, err := i.executor.Run(ctx,
		"env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "update", "-y",
	)
	if err != nil {
		return fmt.Errorf("apt-get update: %w", err)
	}
	return nil
}

// InstallNginx installs Nginx, enables and starts the service.
func (i *Installer) InstallNginx(ctx context.Context) (*InstallResult, error) {
	return i.installSimplePackage(ctx, "nginx", "nginx")
}

// InstallRedis installs Redis, enables and starts the service.
func (i *Installer) InstallRedis(ctx context.Context) (*InstallResult, error) {
	return i.installSimplePackage(ctx, "redis-server", "redis-server")
}

// InstallMariaDB installs MariaDB, enables/starts service, and runs SQL hardening.
func (i *Installer) InstallMariaDB(ctx context.Context) (*InstallResult, error) {
	pkg := "mariadb-server"

	// Idempotency: skip if already installed
	installed, _ := isPackageInstalled(ctx, i.executor, pkg)
	if installed {
		return &InstallResult{Package: pkg, Status: StatusSkipped, Message: "already installed"}, nil
	}

	if err := i.aptInstall(ctx, pkg); err != nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: err.Error()}, err
	}

	if err := i.enableAndStart(ctx, "mariadb"); err != nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: err.Error()}, err
	}

	// Harden MariaDB: remove test DB, anonymous users, remote root
	if err := i.hardenMariaDB(ctx); err != nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: "installed but hardening failed: " + err.Error()}, err
	}

	return &InstallResult{Package: pkg, Status: StatusInstalled, Message: "installed and hardened"}, nil
}

// InstallPHP delegates to php.Manager.InstallVersion for DRY.
func (i *Installer) InstallPHP(ctx context.Context, version string) (*InstallResult, error) {
	pkg := "php" + version

	if i.phpMgr == nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: "PHP manager not available"}, fmt.Errorf("php manager is nil")
	}

	if err := i.phpMgr.InstallVersion(ctx, version); err != nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: err.Error()}, err
	}

	return &InstallResult{Package: pkg, Status: StatusInstalled, Message: "installed via php.Manager"}, nil
}

// installSimplePackage handles the common pattern: check → install → enable+start.
func (i *Installer) installSimplePackage(ctx context.Context, pkg, serviceName string) (*InstallResult, error) {
	installed, _ := isPackageInstalled(ctx, i.executor, pkg)
	if installed {
		return &InstallResult{Package: pkg, Status: StatusSkipped, Message: "already installed"}, nil
	}

	if err := i.aptInstall(ctx, pkg); err != nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: err.Error()}, err
	}

	if err := i.enableAndStart(ctx, serviceName); err != nil {
		return &InstallResult{Package: pkg, Status: StatusFailed, Message: err.Error()}, err
	}

	return &InstallResult{Package: pkg, Status: StatusInstalled, Message: "installed and started"}, nil
}

// aptInstall runs apt-get install with noninteractive flags and lock timeout.
func (i *Installer) aptInstall(ctx context.Context, pkg string) error {
	installCtx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	_, err := i.executor.Run(installCtx,
		"env", "DEBIAN_FRONTEND=noninteractive",
		"apt-get", "install", "-y",
		"-o", "Dpkg::Options::=--force-confdef",
		"-o", "Dpkg::Options::=--force-confold",
		"-o", "DPkg::Lock::Timeout=120",
		pkg,
	)
	if err != nil {
		return fmt.Errorf("apt-get install %s: %w", pkg, err)
	}
	return nil
}

// enableAndStart runs systemctl enable + start for a service.
func (i *Installer) enableAndStart(ctx context.Context, service string) error {
	_, err := i.executor.Run(ctx, "systemctl", "enable", service)
	if err != nil {
		return fmt.Errorf("systemctl enable %s: %w", service, err)
	}

	_, err = i.executor.Run(ctx, "systemctl", "start", service)
	if err != nil {
		return fmt.Errorf("systemctl start %s: %w", service, err)
	}
	return nil
}

// hardenMariaDB removes test DB, anonymous users, and remote root access via SQL.
// Keeps unix_socket auth for root (matches existing DB manager pattern).
func (i *Installer) hardenMariaDB(ctx context.Context) error {
	hardeningSQL := `DELETE FROM mysql.user WHERE User='';
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');
DROP DATABASE IF EXISTS test;
DELETE FROM mysql.db WHERE Db='test' OR Db='test\_%';
FLUSH PRIVILEGES;`

	_, err := i.executor.RunWithInput(ctx, hardeningSQL, "mysql", "--user=root")
	if err != nil {
		return fmt.Errorf("mariadb hardening: %w", err)
	}
	return nil
}

// isPackageInstalled checks if a package is installed via dpkg-query.
// Shared logic used by both Detector and Installer.
func isPackageInstalled(ctx context.Context, exec system.Executor, pkg string) (bool, string) {
	output, err := exec.Run(ctx,
		"dpkg-query", "-W", "--showformat=${Status}\n${Version}", pkg,
	)
	if err != nil {
		return false, ""
	}

	lines := strings.SplitN(strings.TrimSpace(output), "\n", 2)
	if len(lines) < 1 {
		return false, ""
	}

	installed := strings.Contains(lines[0], "install ok installed")
	version := ""
	if installed && len(lines) == 2 {
		version = strings.TrimSpace(lines[1])
	}

	return installed, version
}
