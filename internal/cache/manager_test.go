package cache

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jhin1m/juiscript/internal/config"
)

// mockExecutor simulates Redis/systemctl commands for testing.
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

func defaultCfg() *config.Config {
	return &config.Config{
		Redis: config.RedisConfig{MaxDatabases: 16},
	}
}

// --- Status tests ---

func TestStatus_RedisRunning(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["redis-cli PING"] = "PONG"
	exec.outputs["redis-cli INFO server memory"] = "redis_version:7.2.4\r\nused_memory_human:1.5M\r\n"

	mgr := NewManager(exec, defaultCfg())
	status, err := mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !status.RedisRunning {
		t.Error("expected RedisRunning=true")
	}
	if status.RedisVersion != "7.2.4" {
		t.Errorf("got version %q, want 7.2.4", status.RedisVersion)
	}
	if status.RedisMemory != "1.5M" {
		t.Errorf("got memory %q, want 1.5M", status.RedisMemory)
	}
}

func TestStatus_RedisDown(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["redis-cli PING"] = fmt.Errorf("connection refused")

	mgr := NewManager(exec, defaultCfg())
	status, err := mgr.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.RedisRunning {
		t.Error("expected RedisRunning=false when PING fails")
	}
}

// --- FlushDB tests ---

func TestFlushDB_Valid(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.FlushDB(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exec.hasCommand("redis-cli -n 3 FLUSHDB") {
		t.Error("expected FLUSHDB command for db 3")
	}
}

func TestFlushDB_InvalidDB(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.FlushDB(context.Background(), 99)
	if err == nil {
		t.Fatal("expected validation error for db=99")
	}
	if !strings.Contains(err.Error(), "must be 0-15") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFlushDB_NegativeDB(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.FlushDB(context.Background(), -1)
	if err == nil {
		t.Fatal("expected validation error for db=-1")
	}
}

// --- FlushAll tests ---

func TestFlushAll(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.FlushAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exec.hasCommand("redis-cli FLUSHALL") {
		t.Error("expected FLUSHALL command")
	}
}

// --- ResetOpcache tests ---

func TestResetOpcache(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.ResetOpcache(context.Background(), "8.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exec.hasCommand("systemctl restart php8.3-fpm") {
		t.Error("expected systemctl restart php8.3-fpm")
	}
}

func TestResetOpcache_EmptyVersion(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.ResetOpcache(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty PHP version")
	}
}

func TestResetOpcache_InvalidVersion(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.ResetOpcache(context.Background(), "../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for invalid PHP version")
	}
	if len(exec.commands) > 0 {
		t.Error("should not execute any command with invalid version")
	}
}

func TestResetOpcache_RestartFails(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["systemctl restart php8.3-fpm"] = fmt.Errorf("service not found")
	mgr := NewManager(exec, defaultCfg())

	err := mgr.ResetOpcache(context.Background(), "8.3")
	if err == nil {
		t.Fatal("expected error when restart fails")
	}
	if !strings.Contains(err.Error(), "restart php8.3-fpm") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- EnableRedis tests ---

func TestEnableRedis_AlreadyRunning(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["systemctl is-active redis-server"] = "active"
	exec.outputs["redis-cli PING"] = "PONG"
	mgr := NewManager(exec, defaultCfg())

	err := mgr.EnableRedis(context.Background(), "example.com", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnableRedis_StartsRedis(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["systemctl is-active redis-server"] = fmt.Errorf("inactive")
	exec.outputs["redis-cli PING"] = "PONG"
	mgr := NewManager(exec, defaultCfg())

	err := mgr.EnableRedis(context.Background(), "example.com", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exec.hasCommand("systemctl start redis-server") {
		t.Error("expected systemctl start redis-server")
	}
}

func TestEnableRedis_InvalidDB(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, defaultCfg())

	err := mgr.EnableRedis(context.Background(), "example.com", 20)
	if err == nil {
		t.Fatal("expected validation error for db=20")
	}
}

// --- validateDB tests ---

func TestValidateDB_Bounds(t *testing.T) {
	tests := []struct {
		name    string
		db      int
		max     int
		wantErr bool
	}{
		{"valid min", 0, 16, false},
		{"valid max", 15, 16, false},
		{"valid mid", 8, 16, false},
		{"too high", 16, 16, true},
		{"negative", -1, 16, true},
		{"custom max valid", 3, 4, false},
		{"custom max invalid", 4, 4, true},
		{"zero max uses default", 15, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Redis: config.RedisConfig{MaxDatabases: tt.max}}
			mgr := NewManager(newMockExecutor(), cfg)
			err := mgr.validateDB(tt.db)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDB(%d) error = %v, wantErr = %v", tt.db, err, tt.wantErr)
			}
		})
	}
}

// --- parseRedisField tests ---

func TestParseRedisField(t *testing.T) {
	info := "redis_version:7.2.4\r\nused_memory_human:1.5M\r\nuptime_in_seconds:12345\r\n"
	if v := parseRedisField(info, "redis_version"); v != "7.2.4" {
		t.Errorf("got %q, want 7.2.4", v)
	}
	if v := parseRedisField(info, "used_memory_human"); v != "1.5M" {
		t.Errorf("got %q, want 1.5M", v)
	}
	if v := parseRedisField(info, "nonexistent"); v != "" {
		t.Errorf("got %q, want empty", v)
	}
}
