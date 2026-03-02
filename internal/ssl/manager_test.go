package ssl

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jhin1m/juiscript/internal/nginx"
	"github.com/jhin1m/juiscript/internal/template"
)

// mockExecutor simulates system command execution for testing.
type mockExecutor struct {
	commands []string
	outputs  map[string]string // command prefix -> stdout
	failOn   map[string]error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		outputs: make(map[string]string),
		failOn:  make(map[string]error),
	}
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.commands = append(m.commands, cmd)

	if err, ok := m.failOn[name]; ok {
		return "", err
	}

	// Check for prefix-matched outputs
	for prefix, output := range m.outputs {
		if strings.HasPrefix(cmd, prefix) {
			return output, nil
		}
	}
	return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
	return m.Run(context.Background(), name, args...)
}

// mockFileManager simulates filesystem operations.
type mockFileManager struct {
	written map[string][]byte
	exists  map[string]bool
}

func newMockFileManager() *mockFileManager {
	return &mockFileManager{
		written: make(map[string][]byte),
		exists:  make(map[string]bool),
	}
}

func (f *mockFileManager) WriteAtomic(path string, data []byte, _ os.FileMode) error {
	f.written[path] = data
	f.exists[path] = true
	return nil
}

func (f *mockFileManager) Symlink(_, _ string) error   { return nil }
func (f *mockFileManager) RemoveSymlink(_ string) error { return nil }
func (f *mockFileManager) Remove(path string) error     { delete(f.exists, path); return nil }
func (f *mockFileManager) Exists(path string) bool      { return f.exists[path] }
func (f *mockFileManager) ReadFile(path string) ([]byte, error) {
	if data, ok := f.written[path]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("file not found: %s", path)
}

// setupMockNginxManager creates a real nginx.Manager using mock deps for SSL testing.
func setupMockNginxManager(t *testing.T, exec *mockExecutor, files *mockFileManager) *nginx.Manager {
	t.Helper()
	tpl, err := template.New()
	if err != nil {
		t.Fatalf("failed to create template engine: %v", err)
	}
	return nginx.NewManager(exec, files, tpl,
		"/etc/nginx/sites-available", "/etc/nginx/sites-enabled")
}

// --- Tests for Obtain ---

func TestObtain_Success(t *testing.T) {
	exec := newMockExecutor()
	files := newMockFileManager()

	// Create a real nginx.Manager with mocks for EnableSSL to work
	nginxMgr := setupMockNginxManager(t, exec, files)

	// Pre-create a vhost config so EnableSSL can read it
	vhostPath := "/etc/nginx/sites-available/example.com.conf"
	files.written[vhostPath] = []byte("server {\n    listen 80;\n    listen [::]:80;\n    server_name example.com;\n}")
	files.exists[vhostPath] = true

	mgr := NewManager(exec, nginxMgr, files)

	if err := mgr.Obtain("example.com", "/var/www/html", "admin@example.com"); err != nil {
		t.Fatalf("Obtain failed: %v", err)
	}

	// Verify certbot was called with correct args
	found := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "certbot certonly") &&
			strings.Contains(cmd, "--webroot") &&
			strings.Contains(cmd, "-d example.com") &&
			strings.Contains(cmd, "--email admin@example.com") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected certbot certonly command with webroot method")
	}

	// Verify SSL was enabled in vhost
	result := string(files.written[vhostPath])
	if !strings.Contains(result, "listen 443 ssl http2") {
		t.Error("expected SSL to be enabled in vhost after obtain")
	}
}

func TestObtain_CertbotFails(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["certbot"] = fmt.Errorf("rate limited")
	files := newMockFileManager()
	mgr := NewManager(exec, nil, files)

	err := mgr.Obtain("example.com", "/var/www/html", "admin@example.com")
	if err == nil {
		t.Fatal("expected error when certbot fails")
	}
	if !strings.Contains(err.Error(), "certbot obtain") {
		t.Errorf("expected 'certbot obtain' in error, got: %v", err)
	}
}

func TestObtain_InvalidDomain(t *testing.T) {
	mgr := NewManager(newMockExecutor(), nil, newMockFileManager())

	err := mgr.Obtain("../etc/passwd", "/var/www", "test@test.com")
	if err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

// --- Tests for Revoke ---

func TestRevoke_Success(t *testing.T) {
	exec := newMockExecutor()
	files := newMockFileManager()
	nginxMgr := setupMockNginxManager(t, exec, files)

	// Pre-create vhost with SSL markers so DisableSSL works
	vhostPath := "/etc/nginx/sites-available/example.com.conf"
	files.written[vhostPath] = []byte("# BEGIN SSL\nlisten 443;\n# END SSL\nserver { listen 80; }")
	files.exists[vhostPath] = true

	mgr := NewManager(exec, nginxMgr, files)

	if err := mgr.Revoke("example.com"); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	// Verify certbot revoke and delete were called
	foundRevoke := false
	foundDelete := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "certbot revoke") &&
			strings.Contains(cmd, "--cert-path") {
			foundRevoke = true
		}
		if strings.Contains(cmd, "certbot delete") &&
			strings.Contains(cmd, "--cert-name example.com") {
			foundDelete = true
		}
	}
	if !foundRevoke {
		t.Error("expected certbot revoke command")
	}
	if !foundDelete {
		t.Error("expected certbot delete command")
	}
}

func TestRevoke_CertbotFails(t *testing.T) {
	exec := newMockExecutor()
	exec.failOn["certbot"] = fmt.Errorf("cert not found")
	mgr := NewManager(exec, nil, newMockFileManager())

	err := mgr.Revoke("example.com")
	if err == nil {
		t.Fatal("expected error when certbot revoke fails")
	}
}

// --- Tests for Renew ---

func TestRenew_Success(t *testing.T) {
	exec := newMockExecutor()
	mgr := NewManager(exec, nil, newMockFileManager())

	if err := mgr.Renew("example.com"); err != nil {
		t.Fatalf("Renew failed: %v", err)
	}

	found := false
	for _, cmd := range exec.commands {
		if strings.Contains(cmd, "certbot renew") &&
			strings.Contains(cmd, "--cert-name example.com") &&
			strings.Contains(cmd, "--force-renewal") {
			found = true
		}
	}
	if !found {
		t.Error("expected certbot renew command with force-renewal")
	}
}

func TestRenew_InvalidDomain(t *testing.T) {
	mgr := NewManager(newMockExecutor(), nil, newMockFileManager())

	if err := mgr.Renew(""); err == nil {
		t.Fatal("expected error for empty domain")
	}
}

// --- Tests for Status ---

func TestStatus_Success(t *testing.T) {
	exec := newMockExecutor()
	files := newMockFileManager()

	// Simulate cert file exists
	certPath := "/etc/letsencrypt/live/example.com/fullchain.pem"
	files.exists[certPath] = true

	// Mock openssl output with future expiry
	futureDate := time.Now().Add(60 * 24 * time.Hour).UTC()
	opensslOutput := fmt.Sprintf(
		"notBefore=Jan  1 00:00:00 2026 GMT\nnotAfter=%s\nissuer=CN = R3, O = Let's Encrypt, C = US",
		futureDate.Format("Jan  2 15:04:05 2006 GMT"),
	)
	exec.outputs["openssl"] = opensslOutput

	mgr := NewManager(exec, nil, files)
	info, err := mgr.Status("example.com")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if info.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", info.Domain)
	}
	if !info.Valid {
		t.Error("expected cert to be valid")
	}
	if info.DaysLeft <= 0 {
		t.Errorf("expected positive days left, got %d", info.DaysLeft)
	}
	if !strings.Contains(info.Issuer, "Let's Encrypt") {
		t.Errorf("expected Let's Encrypt issuer, got %s", info.Issuer)
	}
}

func TestStatus_CertNotFound(t *testing.T) {
	files := newMockFileManager()
	// Don't set exists for the cert path
	mgr := NewManager(newMockExecutor(), nil, files)

	_, err := mgr.Status("missing.com")
	if err == nil {
		t.Fatal("expected error when cert file not found")
	}
	if !strings.Contains(err.Error(), "certificate not found") {
		t.Errorf("expected 'certificate not found', got: %v", err)
	}
}

// --- Tests for ListCerts ---

func TestListCerts_Success(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["certbot certificates"] = `Found the following certs:
  Certificate Name: example.com
    Domains: example.com
    Expiry Date: 2026-05-30 00:00:00+00:00 (VALID: 89 days)
    Certificate Path: /etc/letsencrypt/live/example.com/fullchain.pem
  Certificate Name: blog.com
    Domains: blog.com
    Expiry Date: 2026-04-15 00:00:00+00:00 (VALID: 44 days)
    Certificate Path: /etc/letsencrypt/live/blog.com/fullchain.pem`

	mgr := NewManager(exec, nil, newMockFileManager())
	certs, err := mgr.ListCerts()
	if err != nil {
		t.Fatalf("ListCerts failed: %v", err)
	}

	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}

	if certs[0].Domain != "example.com" {
		t.Errorf("expected first cert domain example.com, got %s", certs[0].Domain)
	}
	if certs[1].Domain != "blog.com" {
		t.Errorf("expected second cert domain blog.com, got %s", certs[1].Domain)
	}
	if !certs[0].Valid {
		t.Error("expected first cert to be valid")
	}
}

func TestListCerts_Empty(t *testing.T) {
	exec := newMockExecutor()
	exec.outputs["certbot certificates"] = "No certs found."

	mgr := NewManager(exec, nil, newMockFileManager())
	certs, err := mgr.ListCerts()
	if err != nil {
		t.Fatalf("ListCerts failed: %v", err)
	}
	if len(certs) != 0 {
		t.Errorf("expected 0 certs, got %d", len(certs))
	}
}

// --- Tests for parseCertOutput ---

func TestParseCertOutput_ValidCert(t *testing.T) {
	output := "notBefore=Jan  1 00:00:00 2026 GMT\nnotAfter=Dec 31 23:59:59 2030 GMT\nissuer=CN = R3, O = Let's Encrypt, C = US"

	info, err := parseCertOutput("test.com", output)
	if err != nil {
		t.Fatalf("parseCertOutput failed: %v", err)
	}

	if info.Domain != "test.com" {
		t.Errorf("expected domain test.com, got %s", info.Domain)
	}
	if !info.Valid {
		t.Error("expected cert to be valid")
	}
	if info.DaysLeft <= 0 {
		t.Error("expected positive days left for future cert")
	}
}

func TestParseCertOutput_ExpiredCert(t *testing.T) {
	output := "notBefore=Jan  1 00:00:00 2020 GMT\nnotAfter=Jan  1 00:00:00 2021 GMT\nissuer=CN = Expired CA"

	info, err := parseCertOutput("old.com", output)
	if err != nil {
		t.Fatalf("parseCertOutput failed: %v", err)
	}

	if info.Valid {
		t.Error("expected expired cert to be invalid")
	}
	if info.DaysLeft >= 0 {
		t.Errorf("expected negative days left for expired cert, got %d", info.DaysLeft)
	}
}

func TestParseCertOutput_NoExpiry(t *testing.T) {
	output := "issuer=CN = R3"

	_, err := parseCertOutput("bad.com", output)
	if err == nil {
		t.Fatal("expected error when expiry not found")
	}
}

// --- Tests for validateDomain ---

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email   string
		wantErr bool
	}{
		{"admin@example.com", false},
		{"user+tag@test.com", false},
		{"", true},
		{"nope", true},
		{"bad@;rm -rf /", true},
		{"$(cmd)@evil.com", true},
	}
	for _, tt := range tests {
		err := validateEmail(tt.email)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateEmail(%q): got err=%v, wantErr=%v", tt.email, err, tt.wantErr)
		}
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		domain  string
		wantErr bool
	}{
		{"example.com", false},
		{"sub.example.com", false},
		{"my-site.com", false},
		{"", true},
		{"../etc/passwd", true},
		{"foo/bar", true},
		{"..", true},
		{"example.com; rm -rf /", true},
		{"$(whoami).evil.com", true},
		{"test|cat /etc/passwd", true},
	}

	for _, tt := range tests {
		err := validateDomain(tt.domain)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateDomain(%q): got err=%v, wantErr=%v", tt.domain, err, tt.wantErr)
		}
	}
}
