package system

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

// testMockExecutor records commands for usermgmt tests.
type testMockExecutor struct {
	commands []string
	failOn   map[string]error
}

func newTestMockExecutor() *testMockExecutor {
	return &testMockExecutor{failOn: make(map[string]error)}
}

func (m *testMockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)
	if err, ok := m.failOn[name]; ok {
		return "", err
	}
	return "", nil
}

func (m *testMockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

func TestUserCreate_CallsUseradd(t *testing.T) {
	exec := newTestMockExecutor()
	mgr := NewUserManager(exec)

	if err := mgr.Create("testuser", "/home/testuser"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify useradd was called with correct args
	if len(exec.commands) == 0 {
		t.Fatal("expected useradd command")
	}
	cmd := exec.commands[0]
	if !strings.Contains(cmd, "useradd") {
		t.Errorf("expected useradd, got: %s", cmd)
	}
	if !strings.Contains(cmd, "-m") {
		t.Error("expected -m flag for home dir creation")
	}
	if !strings.Contains(cmd, "/home/testuser") {
		t.Error("expected home dir in command")
	}
	if !strings.Contains(cmd, "testuser") {
		t.Error("expected username in command")
	}
}

func TestUserCreate_FailsOnError(t *testing.T) {
	exec := newTestMockExecutor()
	exec.failOn["useradd"] = fmt.Errorf("permission denied")
	mgr := NewUserManager(exec)

	err := mgr.Create("testuser", "/home/testuser")
	if err == nil {
		t.Fatal("expected error when useradd fails")
	}
}

func TestUserDelete_CallsUserdel(t *testing.T) {
	exec := newTestMockExecutor()
	mgr := NewUserManager(exec)

	// Delete won't call userdel unless user exists (via os/user.Lookup)
	// Since test users don't exist in OS, Delete returns nil (idempotent)
	err := mgr.Delete("nonexistent_juiscript_test")
	if err != nil {
		t.Fatalf("Delete of non-existent user should succeed: %v", err)
	}

	// No commands should be recorded (user doesn't exist, so no userdel)
	if len(exec.commands) > 0 {
		t.Errorf("expected no commands for non-existent user, got: %v", exec.commands)
	}
}

func TestLookupUID_CurrentUser(t *testing.T) {
	exec := newTestMockExecutor()
	mgr := NewUserManager(exec)

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		t.Skip("USER env not set")
	}

	uid, gid, err := mgr.LookupUID(currentUser)
	if err != nil {
		t.Fatalf("LookupUID for current user: %v", err)
	}
	if uid <= 0 {
		t.Errorf("expected positive UID, got %d", uid)
	}
	if gid < 0 {
		t.Errorf("expected non-negative GID, got %d", gid)
	}
}

func TestLookupUID_NonExistentUser(t *testing.T) {
	exec := newTestMockExecutor()
	mgr := NewUserManager(exec)

	_, _, err := mgr.LookupUID("nonexistent_user_juiscript_test_xyz")
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestExists_CurrentUser(t *testing.T) {
	exec := newTestMockExecutor()
	mgr := NewUserManager(exec)

	currentUser := os.Getenv("USER")
	if currentUser == "" {
		t.Skip("USER env not set")
	}

	if !mgr.Exists(currentUser) {
		t.Errorf("current user %q should exist", currentUser)
	}
	if mgr.Exists("nonexistent_user_juiscript_test_xyz") {
		t.Error("nonexistent user should not exist")
	}
}
