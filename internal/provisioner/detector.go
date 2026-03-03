package provisioner

import (
	"context"
	"os"
	"strings"

	"github.com/jhin1m/juiscript/internal/system"
)

// PackageInfo describes a LEMP package's detection result.
type PackageInfo struct {
	Name        string // internal key: "nginx", "mariadb", "redis", "php"
	DisplayName string // TUI label: "Nginx", "MariaDB", "Redis", "PHP 8.3"
	Package     string // apt package name: "nginx", "mariadb-server", "redis-server"
	Installed   bool
	Version     string // e.g. "1.24.0-2ubuntu1"
}

// staticPackages defines the non-PHP LEMP packages to detect.
var staticPackages = []struct {
	name        string
	displayName string
	pkg         string
}{
	{"nginx", "Nginx", "nginx"},
	{"mariadb", "MariaDB", "mariadb-server"},
	{"redis", "Redis", "redis-server"},
}

// defaultPHPVersions are shown when no PHP is installed on the system.
var defaultPHPVersions = []string{"7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}

// Detector checks which LEMP packages are installed on the system.
// Uses dpkg-query for package detection and /etc/php/ scan for PHP versions.
type Detector struct {
	executor system.Executor
}

// NewDetector creates a Detector with the given command executor.
func NewDetector(exec system.Executor) *Detector {
	return &Detector{executor: exec}
}

// DetectAll returns status for all LEMP packages: Nginx, MariaDB, Redis, and PHP versions.
func (d *Detector) DetectAll(ctx context.Context) ([]PackageInfo, error) {
	var results []PackageInfo

	// Check static packages (nginx, mariadb, redis)
	for _, sp := range staticPackages {
		installed, version := isPackageInstalled(ctx, d.executor, sp.pkg)
		results = append(results, PackageInfo{
			Name:        sp.name,
			DisplayName: sp.displayName,
			Package:     sp.pkg,
			Installed:   installed,
			Version:     version,
		})
	}

	// Always show all default PHP versions, plus any extra detected ones.
	// This ensures versions remain selectable after installing one.
	phpVersions := d.mergedPHPVersions()

	for _, ver := range phpVersions {
		pkg := "php" + ver + "-fpm"
		installed, version := isPackageInstalled(ctx, d.executor, pkg)
		results = append(results, PackageInfo{
			Name:        "php",
			DisplayName: "PHP " + ver,
			Package:     pkg,
			Installed:   installed,
			Version:     version,
		})
	}

	return results, nil
}

// detectPHPVersions scans /etc/php/ for directories matching version format (X.Y).
// Reuses the same validation pattern as php.isVersionDir and service.detectPHPVersions.
func (d *Detector) detectPHPVersions() []string {
	entries, err := os.ReadDir("/etc/php")
	if err != nil {
		return nil
	}

	var versions []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if isVersionDir(entry.Name()) {
			versions = append(versions, entry.Name())
		}
	}
	return versions
}

// mergedPHPVersions returns all default PHP versions plus any extra detected versions.
// Ensures users can always install new versions even after installing one.
func (d *Detector) mergedPHPVersions() []string {
	detected := d.detectPHPVersions()

	seen := make(map[string]bool)
	var merged []string
	for _, v := range defaultPHPVersions {
		seen[v] = true
		merged = append(merged, v)
	}
	// Add any extra detected versions not in defaults
	for _, v := range detected {
		if !seen[v] {
			merged = append(merged, v)
		}
	}
	return merged
}

// isVersionDir checks if a name matches PHP version format (e.g. "8.3", "7.4").
func isVersionDir(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
