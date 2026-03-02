package service

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockExecutor simulates systemctl commands for testing.
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

func (m *mockExecutor) hasCommand(substr string) bool {
	for _, cmd := range m.commands {
		if strings.Contains(cmd, substr) {
			return true
		}
	}
	return false
}

// --- isAllowed tests ---

func TestIsAllowed(t *testing.T) {
	allowed := []ServiceName{ServiceNginx, ServiceMariaDB, ServiceRedis, "php8.3-fpm", "php7.4-fpm"}
	for _, name := range allowed {
		if !isAllowed(name) {
			t.Errorf("expected %q to be allowed", name)
		}
	}

	blocked := []ServiceName{"ssh", "cron", "../etc", "rm -rf /", "php8.3fpm", "php-fpm", "phpXXXXX-fpm", "php.evil-fpm"}
	for _, name := range blocked {
		if isAllowed(name) {
			t.Errorf("expected %q to be blocked", name)
		}
	}
}

// --- Start/Stop/Restart/Reload tests ---

func TestStart_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Start(context.Background(), ServiceNginx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !exec.hasCommand("systemctl start nginx") {
		t.Error("expected systemctl start nginx")
	}
}

func TestStop_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Stop(context.Background(), ServiceMariaDB)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if !exec.hasCommand("systemctl stop mariadb") {
		t.Error("expected systemctl stop mariadb")
	}
}

func TestRestart_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Restart(context.Background(), ServiceRedis)
	if err != nil {
		t.Fatalf("Restart failed: %v", err)
	}

	if !exec.hasCommand("systemctl restart redis-server") {
		t.Error("expected systemctl restart redis-server")
	}
}

func TestReload_PHPFPMService(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Reload(context.Background(), PHPFPMService("8.3"))
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if !exec.hasCommand("systemctl reload php8.3-fpm") {
		t.Error("expected systemctl reload php8.3-fpm")
	}
}

func TestAction_BlockedService(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	// Attempting to start an unapproved service should fail
	err := mgr.Start(context.Background(), "sshd")
	if err == nil {
		t.Fatal("expected error for blocked service")
	}
	if !strings.Contains(err.Error(), "not in the allowed list") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAction_SystemctlFailure(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["systemctl start nginx"] = fmt.Errorf("exit code 1")
	mgr := NewManager(exec)

	err := mgr.Start(context.Background(), ServiceNginx)
	if err == nil {
		t.Fatal("expected error when systemctl fails")
	}
}

// --- IsActive tests ---

func TestIsActive_True(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["systemctl is-active nginx"] = "active\n"
	mgr := NewManager(exec)

	if !mgr.IsActive(context.Background(), ServiceNginx) {
		t.Error("expected nginx to be active")
	}
}

func TestIsActive_False(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["systemctl is-active nginx"] = "inactive\n"
	mgr := NewManager(exec)

	if mgr.IsActive(context.Background(), ServiceNginx) {
		t.Error("expected nginx to be inactive")
	}
}

func TestIsActive_BlockedService(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	if mgr.IsActive(context.Background(), "cron") {
		t.Error("blocked service should return false")
	}
}

// --- Status tests ---

func TestStatus_ParsesProperties(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["systemctl show nginx --property=ActiveState,SubState,MainPID,MemoryCurrent"] =
		"ActiveState=active\nSubState=running\nMainPID=1234\nMemoryCurrent=52428800\n"
	mgr := NewManager(exec)

	st, err := mgr.Status(context.Background(), ServiceNginx)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if !st.Active {
		t.Error("expected Active=true")
	}
	if st.State != "active" {
		t.Errorf("expected State=active, got %s", st.State)
	}
	if st.SubState != "running" {
		t.Errorf("expected SubState=running, got %s", st.SubState)
	}
	if st.PID != 1234 {
		t.Errorf("expected PID=1234, got %d", st.PID)
	}
	// 52428800 bytes = 50 MB
	if st.MemoryMB < 49.9 || st.MemoryMB > 50.1 {
		t.Errorf("expected MemoryMB ~50, got %.1f", st.MemoryMB)
	}
}

func TestStatus_InactiveService(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["systemctl show mariadb --property=ActiveState,SubState,MainPID,MemoryCurrent"] =
		"ActiveState=inactive\nSubState=dead\nMainPID=0\nMemoryCurrent=0\n"
	mgr := NewManager(exec)

	st, err := mgr.Status(context.Background(), ServiceMariaDB)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if st.Active {
		t.Error("expected Active=false")
	}
	if st.PID != 0 {
		t.Errorf("expected PID=0, got %d", st.PID)
	}
}

func TestStatus_BlockedService(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	_, err := mgr.Status(context.Background(), "ssh")
	if err == nil {
		t.Fatal("expected error for blocked service")
	}
}

// --- parseStatus tests ---

func TestParseStatus_HandlesBadInput(t *testing.T) {
	st := parseStatus(ServiceNginx, "garbage=data\nno-equals-sign\n=empty-key\n")

	if st.Name != ServiceNginx {
		t.Error("Name should be preserved")
	}
	if st.Active {
		t.Error("should default to inactive")
	}
}

// --- PHPFPMService tests ---

func TestPHPFPMService(t *testing.T) {
	if got := PHPFPMService("8.3"); got != "php8.3-fpm" {
		t.Errorf("PHPFPMService(8.3) = %q, want php8.3-fpm", got)
	}
	if got := PHPFPMService("7.4"); got != "php7.4-fpm" {
		t.Errorf("PHPFPMService(7.4) = %q, want php7.4-fpm", got)
	}
}

// --- isNumeric tests ---

func TestIsNumeric(t *testing.T) {
	if !isNumeric("123") {
		t.Error("123 should be numeric")
	}
	if isNumeric("") {
		t.Error("empty string should not be numeric")
	}
	if isNumeric("abc") {
		t.Error("abc should not be numeric")
	}
	if isNumeric("12.3") {
		t.Error("12.3 should not be numeric (contains dot)")
	}
}
