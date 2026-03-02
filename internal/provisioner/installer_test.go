package provisioner

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// aptInstallCmd builds the expected apt-get install command string for mock matching.
func aptInstallCmd(pkg string) string {
	return "env DEBIAN_FRONTEND=noninteractive apt-get install -y " +
		"-o Dpkg::Options::=--force-confdef " +
		"-o Dpkg::Options::=--force-confold " +
		"-o DPkg::Lock::Timeout=120 " + pkg
}

// --- AptUpdate tests ---

func TestAptUpdate_Success(t *testing.T) {
	exec := newMockExecutor()
	inst := NewInstaller(exec, nil)

	err := inst.AptUpdate(context.Background())
	if err != nil {
		t.Fatalf("AptUpdate failed: %v", err)
	}

	if !exec.hasCommand("env DEBIAN_FRONTEND=noninteractive apt-get update -y") {
		t.Error("expected apt-get update command")
	}
}

func TestAptUpdate_Failure(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["env DEBIAN_FRONTEND=noninteractive apt-get update -y"] = fmt.Errorf("network error")
	inst := NewInstaller(exec, nil)

	err := inst.AptUpdate(context.Background())
	if err == nil {
		t.Fatal("expected error when apt-get update fails")
	}
}

// --- InstallNginx tests ---

func TestInstallNginx_Success(t *testing.T) {
	exec := newMockExecutor()
	// dpkg-query returns "not found" → package not installed
	exec.failOn[dpkgCmd("nginx")] = fmt.Errorf("not found")
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallNginx(context.Background())
	if err != nil {
		t.Fatalf("InstallNginx failed: %v", err)
	}

	if result.Status != StatusInstalled {
		t.Errorf("expected status installed, got %s", result.Status)
	}
	if !exec.hasCommand("systemctl enable nginx") {
		t.Error("expected systemctl enable nginx")
	}
	if !exec.hasCommand("systemctl start nginx") {
		t.Error("expected systemctl start nginx")
	}
}

func TestInstallNginx_AlreadyInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs[dpkgCmd("nginx")] = "install ok installed\n1.24.0"
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallNginx(context.Background())
	if err != nil {
		t.Fatalf("InstallNginx failed: %v", err)
	}

	if result.Status != StatusSkipped {
		t.Errorf("expected status skipped, got %s", result.Status)
	}
	// Should NOT run apt-get install when already installed
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "apt-get install") {
			t.Error("should not run apt-get install for already-installed package")
		}
	}
}

func TestInstallNginx_AptFailure(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("nginx")] = fmt.Errorf("not found")
	exec.failOn[aptInstallCmd("nginx")] = fmt.Errorf("apt-get failed")
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallNginx(context.Background())
	if err == nil {
		t.Fatal("expected error when apt-get install fails")
	}
	if result.Status != StatusFailed {
		t.Errorf("expected status failed, got %s", result.Status)
	}
}

// --- InstallRedis tests ---

func TestInstallRedis_Success(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("redis-server")] = fmt.Errorf("not found")
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallRedis(context.Background())
	if err != nil {
		t.Fatalf("InstallRedis failed: %v", err)
	}
	if result.Status != StatusInstalled {
		t.Errorf("expected installed, got %s", result.Status)
	}
}

// --- InstallMariaDB tests ---

func TestInstallMariaDB_Success(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("mariadb-server")] = fmt.Errorf("not found")
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallMariaDB(context.Background())
	if err != nil {
		t.Fatalf("InstallMariaDB failed: %v", err)
	}

	if result.Status != StatusInstalled {
		t.Errorf("expected installed, got %s", result.Status)
	}
	// Verify hardening SQL was executed via mysql
	if !exec.hasCommand("mysql --user=root") {
		t.Error("expected MariaDB hardening SQL execution")
	}
	// Verify hardening SQL content
	if !strings.Contains(exec.lastInput, "DELETE FROM mysql.user WHERE User=''") {
		t.Error("expected hardening SQL to remove anonymous users")
	}
	if !strings.Contains(exec.lastInput, "DROP DATABASE IF EXISTS test") {
		t.Error("expected hardening SQL to drop test database")
	}
	if !strings.Contains(exec.lastInput, "FLUSH PRIVILEGES") {
		t.Error("expected hardening SQL to flush privileges")
	}
}

func TestInstallMariaDB_AlreadyInstalled(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs[dpkgCmd("mariadb-server")] = "install ok installed\n10.11.6"
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallMariaDB(context.Background())
	if err != nil {
		t.Fatalf("InstallMariaDB failed: %v", err)
	}
	if result.Status != StatusSkipped {
		t.Errorf("expected skipped, got %s", result.Status)
	}
}

func TestInstallMariaDB_HardeningFailure(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn[dpkgCmd("mariadb-server")] = fmt.Errorf("not found")
	exec.failOn["mysql --user=root"] = fmt.Errorf("access denied")
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallMariaDB(context.Background())
	if err == nil {
		t.Fatal("expected error when hardening fails")
	}
	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
	if !strings.Contains(result.Message, "hardening failed") {
		t.Errorf("expected hardening failure message, got %q", result.Message)
	}
}

// --- InstallPHP tests ---

func TestInstallPHP_NilManager(t *testing.T) {
	exec := newMockExecutor()
	inst := NewInstaller(exec, nil)

	result, err := inst.InstallPHP(context.Background(), "8.3")
	if err == nil {
		t.Fatal("expected error when PHP manager is nil")
	}
	if result.Status != StatusFailed {
		t.Errorf("expected failed, got %s", result.Status)
	}
}
