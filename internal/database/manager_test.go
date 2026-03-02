package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

// mockExecutor simulates system command execution for testing.
type mockExecutor struct {
	commands []string
	output   string // next output to return
	failOn   map[string]error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{failOn: make(map[string]error)}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)
	if err, ok := m.failOn[name]; ok {
		return "", err
	}
	out := m.output
	m.output = "" // consume once
	return out, nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

func (m *mockExecutor) lastCommand() string {
	if len(m.commands) == 0 {
		return ""
	}
	return m.commands[len(m.commands)-1]
}

// --- Validation Tests ---

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "mydb", false},
		{"valid with underscore", "my_database", false},
		{"valid with numbers", "db123", false},
		{"empty", "", true},
		{"starts with number", "1db", true},
		{"has dash", "my-db", true},
		{"has uppercase", "MyDB", true},
		{"has space", "my db", true},
		{"sql injection attempt", "db; DROP TABLE", true},
		{"backtick injection", "db`evil`", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestIsSystemDB(t *testing.T) {
	systemNames := []string{"information_schema", "mysql", "performance_schema", "sys"}
	for _, name := range systemNames {
		if !isSystemDB(name) {
			t.Errorf("isSystemDB(%q) = false, want true", name)
		}
	}
	if isSystemDB("myapp") {
		t.Error("isSystemDB(myapp) = true, want false")
	}
}

// --- CreateDB Tests ---

func TestCreateDB(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)
	ctx := context.Background()

	err := mgr.CreateDB(ctx, "testdb")
	if err != nil {
		t.Fatalf("CreateDB() error = %v", err)
	}

	last := exec.lastCommand()
	if !strings.Contains(last, "CREATE DATABASE") {
		t.Errorf("expected CREATE DATABASE, got: %s", last)
	}
	if !strings.Contains(last, "utf8mb4") {
		t.Errorf("expected utf8mb4 charset, got: %s", last)
	}
}

func TestCreateDB_InvalidName(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.CreateDB(context.Background(), "invalid-name!")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if len(exec.commands) > 0 {
		t.Error("should not execute SQL for invalid name")
	}
}

// --- DropDB Tests ---

func TestDropDB(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.DropDB(context.Background(), "testdb")
	if err != nil {
		t.Fatalf("DropDB() error = %v", err)
	}
	if !strings.Contains(exec.lastCommand(), "DROP DATABASE") {
		t.Errorf("expected DROP DATABASE command")
	}
}

func TestDropDB_SystemDB(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.DropDB(context.Background(), "mysql")
	if err == nil {
		t.Fatal("expected error when dropping system DB")
	}
	if len(exec.commands) > 0 {
		t.Error("should not execute SQL for system DB")
	}
}

// --- ListDBs Tests ---

func TestListDBs(t *testing.T) {
	exec := newMockExecutor()
	exec.output = "testdb\t1.50\t5\nmyapp\t0.25\t3\nmysql\t0.80\t31\n"
	mgr := NewManager(exec)

	dbs, err := mgr.ListDBs(context.Background())
	if err != nil {
		t.Fatalf("ListDBs() error = %v", err)
	}

	// Should filter out system DBs (mysql)
	if len(dbs) != 2 {
		t.Fatalf("expected 2 DBs, got %d", len(dbs))
	}
	if dbs[0].Name != "testdb" {
		t.Errorf("expected first DB name 'testdb', got %q", dbs[0].Name)
	}
}

// --- User Tests ---

func TestCreateUser(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	password, err := mgr.CreateUser(context.Background(), "site_user", "mydb")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if len(password) != 24 {
		t.Errorf("expected 24 char password, got %d", len(password))
	}

	last := exec.lastCommand()
	if !strings.Contains(last, "CREATE USER") {
		t.Errorf("expected CREATE USER, got: %s", last)
	}
	if !strings.Contains(last, "GRANT ALL") {
		t.Errorf("expected GRANT ALL, got: %s", last)
	}
}

func TestCreateUser_InvalidUsername(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	_, err := mgr.CreateUser(context.Background(), "INVALID", "mydb")
	if err == nil {
		t.Fatal("expected error for invalid username")
	}
}

func TestDropUser(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.DropUser(context.Background(), "site_user")
	if err != nil {
		t.Fatalf("DropUser() error = %v", err)
	}
	last := exec.lastCommand()
	if !strings.Contains(last, "DROP USER") {
		t.Errorf("expected DROP USER, got: %s", last)
	}
}

func TestResetPassword(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	password, err := mgr.ResetPassword(context.Background(), "site_user")
	if err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if len(password) != 24 {
		t.Errorf("expected 24 char password, got %d", len(password))
	}
	last := exec.lastCommand()
	if !strings.Contains(last, "ALTER USER") {
		t.Errorf("expected ALTER USER, got: %s", last)
	}
}

// --- Import/Export Tests ---

func TestImport_PlainSQL(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	// Create a temp file to satisfy os.Stat check
	tmpFile := t.TempDir() + "/dump.sql"
	createTestFile(t, tmpFile)

	err := mgr.Import(context.Background(), "testdb", tmpFile)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	last := exec.lastCommand()
	if !strings.Contains(last, "mysql -u root testdb") {
		t.Errorf("expected mysql import command, got: %s", last)
	}
}

func TestImport_GzippedSQL(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	tmpFile := t.TempDir() + "/dump.sql.gz"
	createTestFile(t, tmpFile)

	err := mgr.Import(context.Background(), "testdb", tmpFile)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	last := exec.lastCommand()
	if !strings.Contains(last, "gunzip") {
		t.Errorf("expected gunzip in command, got: %s", last)
	}
}

func TestExport(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Export(context.Background(), "testdb", "/tmp/backup.sql.gz")
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	last := exec.lastCommand()
	if !strings.Contains(last, "mysqldump") {
		t.Errorf("expected mysqldump command, got: %s", last)
	}
	if !strings.Contains(last, "--single-transaction") {
		t.Errorf("expected --single-transaction flag, got: %s", last)
	}
	if !strings.Contains(last, "gzip") {
		t.Errorf("expected gzip in command, got: %s", last)
	}
}

func TestExport_PlainSQL(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Export(context.Background(), "testdb", "/tmp/backup.sql")
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	last := exec.lastCommand()
	if strings.Contains(last, "gzip") {
		t.Errorf("should not use gzip for .sql output, got: %s", last)
	}
}

// --- Password Generator Tests ---

func TestGeneratePassword(t *testing.T) {
	pw, err := GeneratePassword(24)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	if len(pw) != 24 {
		t.Errorf("expected 24 chars, got %d", len(pw))
	}

	// Verify uniqueness (two passwords should differ)
	pw2, _ := GeneratePassword(24)
	if pw == pw2 {
		t.Error("two generated passwords should not be identical")
	}
}

func TestGeneratePassword_MinLength(t *testing.T) {
	pw, err := GeneratePassword(3)
	if err != nil {
		t.Fatalf("GeneratePassword() error = %v", err)
	}
	// Should enforce minimum of 8
	if len(pw) < 8 {
		t.Errorf("expected minimum 8 chars, got %d", len(pw))
	}
}

// --- Path Validation Tests ---

func TestImport_ShellInjection(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Import(context.Background(), "testdb", "/tmp/dump.sql; rm -rf /")
	if err == nil {
		t.Fatal("expected error for path with shell metacharacters")
	}
	if len(exec.commands) > 0 {
		t.Error("should not execute any command for unsafe path")
	}
}

func TestExport_ShellInjection(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec)

	err := mgr.Export(context.Background(), "testdb", "/tmp/$(whoami).sql")
	if err == nil {
		t.Fatal("expected error for path with shell metacharacters")
	}
	if len(exec.commands) > 0 {
		t.Error("should not execute any command for unsafe path")
	}
}

// --- Error Handling ---

func TestCreateDB_ExecutorFails(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["mysql"] = fmt.Errorf("connection refused")
	mgr := NewManager(exec)

	err := mgr.CreateDB(context.Background(), "testdb")
	if err == nil {
		t.Fatal("expected error when executor fails")
	}
}

// createTestFile creates a dummy file for import tests.
func createTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("-- test sql"), 0644); err != nil {
		t.Fatal(err)
	}
}
