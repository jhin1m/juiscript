package provisioner

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockExecutor simulates dpkg-query commands for testing.
type mockExecutor struct {
	commands []string
	failOn   map[string]error
	outputs  map[string]string
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		failOn:  make(map[string]error),
		outputs: make(map[string]string),
	}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)

	if output, ok := m.outputs[cmd]; ok {
		if err, ok := m.failOn[cmd]; ok {
			return output, err
		}
		return output, nil
	}
	if err, ok := m.failOn[cmd]; ok {
		return "", err
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

// dpkgCmd builds the expected dpkg-query command string for matching mock outputs.
func dpkgCmd(pkg string) string {
	return "dpkg-query -W --showformat=${Status}\n${Version} " + pkg
}

// --- isInstalled tests ---

func TestIsInstalled_InstalledPackage(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs[dpkgCmd("nginx")] = "install ok installed\n1.24.0-2ubuntu1"

	d := NewDetector(exec)
	installed, version := d.isInstalled(context.Background(), "nginx")

	if !installed {
		t.Error("expected nginx to be installed")
	}
	if version != "1.24.0-2ubuntu1" {
		t.Errorf("expected version 1.24.0-2ubuntu1, got %q", version)
	}
}

func TestIsInstalled_NotInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("nginx")] = fmt.Errorf("exit code 1: no packages found")

	d := NewDetector(exec)
	installed, version := d.isInstalled(context.Background(), "nginx")

	if installed {
		t.Error("expected nginx to NOT be installed")
	}
	if version != "" {
		t.Errorf("expected empty version, got %q", version)
	}
}

func TestIsInstalled_BadStatus(t *testing.T) {
	exec := newMockExecutor()
	// Package exists in dpkg DB but is deinstalled/removed
	exec.outputs[dpkgCmd("redis-server")] = "deinstall ok config-files\n7.0.15"

	d := NewDetector(exec)
	installed, version := d.isInstalled(context.Background(), "redis-server")

	if installed {
		t.Error("expected redis-server to NOT be installed (deinstalled state)")
	}
	if version != "" {
		t.Errorf("expected empty version for non-installed, got %q", version)
	}
}

// --- DetectAll tests (table-driven) ---
// Note: DetectAll also calls detectPHPVersions which reads /etc/php/.
// On dev machines (macOS), /etc/php/ won't exist, so PHP returns single placeholder.

func TestDetectAll_AllInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs[dpkgCmd("nginx")] = "install ok installed\n1.24.0"
	exec.outputs[dpkgCmd("mariadb-server")] = "install ok installed\n10.11.6"
	exec.outputs[dpkgCmd("redis-server")] = "install ok installed\n7.0.15"

	d := NewDetector(exec)
	results, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	// Should have 3 static + 1 PHP placeholder (no /etc/php on macOS)
	if len(results) < 4 {
		t.Fatalf("expected at least 4 results, got %d", len(results))
	}

	// Verify static packages
	for _, name := range []string{"nginx", "mariadb", "redis"} {
		found := false
		for _, r := range results {
			if r.Name == name && r.Installed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %s to be installed", name)
		}
	}
}

func TestDetectAll_NoneInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("nginx")] = fmt.Errorf("not found")
	exec.failOn[dpkgCmd("mariadb-server")] = fmt.Errorf("not found")
	exec.failOn[dpkgCmd("redis-server")] = fmt.Errorf("not found")

	d := NewDetector(exec)
	results, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	for _, r := range results {
		if r.Installed {
			t.Errorf("expected %s to NOT be installed", r.Name)
		}
	}
}

func TestDetectAll_PartialInstall(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs[dpkgCmd("nginx")] = "install ok installed\n1.24.0"
	exec.failOn[dpkgCmd("mariadb-server")] = fmt.Errorf("not found")
	exec.outputs[dpkgCmd("redis-server")] = "install ok installed\n7.0.15"

	d := NewDetector(exec)
	results, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	expectations := map[string]bool{
		"nginx":   true,
		"mariadb": false,
		"redis":   true,
	}

	for _, r := range results {
		if expected, ok := expectations[r.Name]; ok {
			if r.Installed != expected {
				t.Errorf("%s: expected Installed=%v, got %v", r.Name, expected, r.Installed)
			}
		}
	}
}

// --- isVersionDir tests ---

func TestIsVersionDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"8.3", true},
		{"7.4", true},
		{"8.0", true},
		{"", false},
		{"8", false},
		{"abc", false},
		{"8.3.1", false},
		{".3", false},
		{"8.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVersionDir(tt.name); got != tt.want {
				t.Errorf("isVersionDir(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// --- PackageInfo field tests ---

func TestDetectAll_CorrectDisplayNames(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs[dpkgCmd("nginx")] = "install ok installed\n1.24.0"
	exec.outputs[dpkgCmd("mariadb-server")] = "install ok installed\n10.11.6"
	exec.outputs[dpkgCmd("redis-server")] = "install ok installed\n7.0.15"

	d := NewDetector(exec)
	results, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	expectedDisplayNames := map[string]string{
		"nginx":   "Nginx",
		"mariadb": "MariaDB",
		"redis":   "Redis",
	}

	for _, r := range results {
		if expected, ok := expectedDisplayNames[r.Name]; ok {
			if r.DisplayName != expected {
				t.Errorf("%s: expected DisplayName=%q, got %q", r.Name, expected, r.DisplayName)
			}
		}
	}
}

func TestDetectAll_PHPPlaceholderWhenNotInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("nginx")] = fmt.Errorf("not found")
	exec.failOn[dpkgCmd("mariadb-server")] = fmt.Errorf("not found")
	exec.failOn[dpkgCmd("redis-server")] = fmt.Errorf("not found")

	d := NewDetector(exec)
	results, err := d.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll failed: %v", err)
	}

	// Find PHP entry
	var phpEntry *PackageInfo
	for i := range results {
		if results[i].Name == "php" {
			phpEntry = &results[i]
			break
		}
	}

	if phpEntry == nil {
		t.Fatal("expected PHP entry in results")
	}
	if phpEntry.Installed {
		t.Error("expected PHP to NOT be installed")
	}
	if phpEntry.DisplayName != "PHP" {
		t.Errorf("expected DisplayName=PHP, got %q", phpEntry.DisplayName)
	}
	if phpEntry.Package != "" {
		t.Errorf("expected empty Package for PHP placeholder, got %q", phpEntry.Package)
	}
}
