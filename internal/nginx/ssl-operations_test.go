package nginx

import (
	"strings"
	"testing"
)

func TestEnableSSL(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	// Pre-create a vhost config
	vhostContent := `server {
    listen 80;
    listen [::]:80;
    server_name ssl.test.com;
    root /var/www/html;
}`
	availPath := "/etc/nginx/sites-available/ssl.test.com.conf"
	files.written[availPath] = []byte(vhostContent)
	files.exists[availPath] = true

	cfg := SSLConfig{
		CertPath: "/etc/letsencrypt/live/ssl.test.com/fullchain.pem",
		KeyPath:  "/etc/letsencrypt/live/ssl.test.com/privkey.pem",
	}

	if err := mgr.EnableSSL("ssl.test.com", cfg); err != nil {
		t.Fatalf("EnableSSL failed: %v", err)
	}

	// Verify SSL directives were injected
	result := string(files.written[availPath])

	if !strings.Contains(result, "listen 443 ssl http2") {
		t.Error("expected SSL listen directive")
	}
	if !strings.Contains(result, cfg.CertPath) {
		t.Error("expected SSL certificate path")
	}
	if !strings.Contains(result, cfg.KeyPath) {
		t.Error("expected SSL key path")
	}
	if !strings.Contains(result, "ssl_protocols TLSv1.2 TLSv1.3") {
		t.Error("expected TLS protocol settings")
	}
	if !strings.Contains(result, "ssl_stapling on") {
		t.Error("expected OCSP stapling")
	}
	if !strings.Contains(result, "Strict-Transport-Security") {
		t.Error("expected HSTS header (commented out)")
	}

	// Verify redirect block was added
	if !strings.Contains(result, sslRedirectBegin) {
		t.Error("expected SSL redirect block")
	}
	if !strings.Contains(result, "return 301 https://") {
		t.Error("expected HTTP-to-HTTPS redirect")
	}
}

func TestEnableSSL_RollbackOnTestFailure(t *testing.T) {
	mgr, exec, files := setupTestManager(t)

	original := `server {
    listen 80;
    listen [::]:80;
    server_name fail.test.com;
}`
	availPath := "/etc/nginx/sites-available/fail.test.com.conf"
	files.written[availPath] = []byte(original)
	files.exists[availPath] = true

	// Make nginx -t fail
	exec.failOn["nginx"] = errTest

	cfg := SSLConfig{
		CertPath: "/etc/letsencrypt/live/fail.test.com/fullchain.pem",
		KeyPath:  "/etc/letsencrypt/live/fail.test.com/privkey.pem",
	}

	err := mgr.EnableSSL("fail.test.com", cfg)
	if err == nil {
		t.Fatal("expected error when nginx test fails")
	}

	// Verify rollback: original content restored
	restored := string(files.written[availPath])
	if strings.Contains(restored, "listen 443 ssl") {
		t.Error("expected SSL to be rolled back")
	}
}

func TestDisableSSL(t *testing.T) {
	mgr, _, files := setupTestManager(t)

	// Config with SSL enabled (has markers)
	sslConfig := sslRedirectBegin + `
server {
    listen 80;
    server_name disable.test.com;
    location / { return 301 https://$host$request_uri; }
}
` + sslRedirectEnd + `
server {
    listen 80;
    listen [::]:80;

    ` + sslBlockBegin + `
    listen 443 ssl http2;
    ssl_certificate /etc/letsencrypt/live/disable.test.com/fullchain.pem;
    ` + sslBlockEnd + `
    server_name disable.test.com;
    root /var/www/html;
}`

	availPath := "/etc/nginx/sites-available/disable.test.com.conf"
	files.written[availPath] = []byte(sslConfig)
	files.exists[availPath] = true

	if err := mgr.DisableSSL("disable.test.com"); err != nil {
		t.Fatalf("DisableSSL failed: %v", err)
	}

	result := string(files.written[availPath])

	// Verify SSL sections removed
	if strings.Contains(result, sslBlockBegin) {
		t.Error("expected SSL block to be removed")
	}
	if strings.Contains(result, sslRedirectBegin) {
		t.Error("expected redirect block to be removed")
	}
	if strings.Contains(result, "listen 443 ssl") {
		t.Error("expected SSL listen to be removed")
	}

	// Original content should remain
	if !strings.Contains(result, "server_name disable.test.com") {
		t.Error("expected original server_name to remain")
	}
}

func TestEnableSSL_InvalidDomain(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	err := mgr.EnableSSL("../bad", SSLConfig{})
	if err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

func TestDisableSSL_InvalidDomain(t *testing.T) {
	mgr, _, _ := setupTestManager(t)

	err := mgr.DisableSSL("")
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
}

func TestRemoveBetweenMarkers(t *testing.T) {
	input := "before\n# BEGIN\nstuff\nmore stuff\n# END\nafter"
	result := removeBetweenMarkers(input, "# BEGIN", "# END")

	if strings.Contains(result, "stuff") {
		t.Error("expected content between markers to be removed")
	}
	if !strings.Contains(result, "before") || !strings.Contains(result, "after") {
		t.Error("expected surrounding content to remain")
	}
}

func TestRemoveBetweenMarkers_NoMarkers(t *testing.T) {
	input := "just normal text"
	result := removeBetweenMarkers(input, "# BEGIN", "# END")

	if result != input {
		t.Error("expected unchanged text when no markers present")
	}
}

func TestInjectSSLDirectives(t *testing.T) {
	config := `server {
    listen 80;
    listen [::]:80;
    server_name inject.test.com;
}`

	cfg := SSLConfig{
		CertPath: "/etc/letsencrypt/live/inject.test.com/fullchain.pem",
		KeyPath:  "/etc/letsencrypt/live/inject.test.com/privkey.pem",
	}

	result, err := injectSSLDirectives(config, cfg)
	if err != nil {
		t.Fatalf("injectSSLDirectives failed: %v", err)
	}

	if !strings.Contains(result, "listen 443 ssl http2") {
		t.Error("expected SSL listen directive injected")
	}
	if !strings.Contains(result, cfg.CertPath) {
		t.Error("expected cert path in injected config")
	}

	// Verify marker comments present
	if !strings.Contains(result, sslBlockBegin) || !strings.Contains(result, sslBlockEnd) {
		t.Error("expected SSL block markers")
	}

	// Verify original content preserved
	if !strings.Contains(result, "listen 80;") {
		t.Error("expected original listen directive preserved")
	}
}

func TestInjectSSLDirectives_NoInsertionPoint(t *testing.T) {
	config := "server { server_name test.com; }" // no listen [::]:80;
	_, err := injectSSLDirectives(config, SSLConfig{CertPath: "/a", KeyPath: "/b"})
	if err == nil {
		t.Fatal("expected error when insertion point not found")
	}
}

func TestBuildRedirectBlock(t *testing.T) {
	result := buildRedirectBlock("redirect.test.com", "/var/www/html")

	if !strings.Contains(result, sslRedirectBegin) {
		t.Error("expected redirect begin marker")
	}
	if !strings.Contains(result, "server_name redirect.test.com") {
		t.Error("expected domain in redirect block")
	}
	if !strings.Contains(result, "return 301 https://") {
		t.Error("expected HTTPS redirect")
	}
	if !strings.Contains(result, ".well-known/acme-challenge") {
		t.Error("expected ACME challenge location")
	}
}

// errTest is a reusable test error.
var errTest = errorf("test error")

func errorf(msg string) error {
	return &testError{msg}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
