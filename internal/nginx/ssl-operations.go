package nginx

import (
	"fmt"
	"strings"
)

// SSLConfig holds certificate paths and settings for enabling SSL on a vhost.
type SSLConfig struct {
	CertPath string // Path to fullchain.pem
	KeyPath  string // Path to privkey.pem
	WebRoot  string // Web root for ACME challenge (default: /var/www/html)
}

// SSL marker comments for clean injection and removal.
const (
	sslRedirectBegin = "# BEGIN SSL REDIRECT"
	sslRedirectEnd   = "# END SSL REDIRECT"
	sslBlockBegin    = "# BEGIN SSL"
	sslBlockEnd      = "# END SSL"
)

// EnableSSL adds SSL directives and an HTTP redirect to an existing vhost config.
// Uses marker comments so SSL sections can be cleanly identified and removed.
func (m *Manager) EnableSSL(domain string, cfg SSLConfig) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	// Read the existing vhost config
	availablePath := m.availablePath(domain)
	content, err := m.files.ReadFile(availablePath)
	if err != nil {
		return fmt.Errorf("read vhost config: %w", err)
	}

	original := string(content)

	// Already has SSL? Remove old SSL sections first to avoid duplication
	if strings.Contains(original, sslBlockBegin) {
		original = removeSSLSections(original)
	}

	// Inject SSL directives into the server block (after "listen [::]:80;")
	enhanced, err := injectSSLDirectives(original, cfg)
	if err != nil {
		return fmt.Errorf("inject ssl: %w", err)
	}

	// Prepend the HTTP-to-HTTPS redirect block
	webRoot := cfg.WebRoot
	if webRoot == "" {
		webRoot = "/var/www/html"
	}
	redirect := buildRedirectBlock(domain, webRoot)
	final := redirect + "\n" + enhanced

	// Write atomically, test, reload (same pattern as Create)
	if err := m.files.WriteAtomic(availablePath, []byte(final), 0644); err != nil {
		return fmt.Errorf("write ssl vhost: %w", err)
	}

	if err := m.Test(); err != nil {
		// Rollback: restore original config
		m.files.WriteAtomic(availablePath, content, 0644)
		return fmt.Errorf("nginx test failed after ssl enable (rolled back): %w", err)
	}

	return m.Reload()
}

// DisableSSL removes SSL directives and the redirect block from a vhost config.
func (m *Manager) DisableSSL(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	availablePath := m.availablePath(domain)
	content, err := m.files.ReadFile(availablePath)
	if err != nil {
		return fmt.Errorf("read vhost config: %w", err)
	}

	cleaned := removeSSLSections(string(content))

	if err := m.files.WriteAtomic(availablePath, []byte(cleaned), 0644); err != nil {
		return fmt.Errorf("write vhost without ssl: %w", err)
	}

	if err := m.Test(); err != nil {
		// Rollback: restore original
		m.files.WriteAtomic(availablePath, content, 0644)
		return fmt.Errorf("nginx test failed after ssl disable (rolled back): %w", err)
	}

	return m.Reload()
}

// injectSSLDirectives adds SSL listen and certificate lines after the "listen [::]:80;" line.
// Returns error if the insertion point is not found in the config.
func injectSSLDirectives(config string, cfg SSLConfig) (string, error) {
	sslSnippet := fmt.Sprintf(`    %s
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    ssl_certificate %s;
    ssl_certificate_key %s;

    # Modern TLS settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # OCSP stapling for faster TLS handshake
    ssl_stapling on;
    ssl_stapling_verify on;

    # HSTS (uncomment after confirming SSL works)
    # add_header Strict-Transport-Security "max-age=63072000" always;
    %s`,
		sslBlockBegin, cfg.CertPath, cfg.KeyPath, sslBlockEnd)

	// Insert after the "listen [::]:80;" line
	lines := strings.Split(config, "\n")
	var result []string
	inserted := false

	for _, line := range lines {
		result = append(result, line)
		// Insert SSL block after the IPv6 listen line
		if !inserted && strings.Contains(strings.TrimSpace(line), "listen [::]:80;") {
			result = append(result, "")
			result = append(result, sslSnippet)
			inserted = true
		}
	}

	if !inserted {
		return "", fmt.Errorf("could not find 'listen [::]:80;' in vhost config to inject SSL directives")
	}

	return strings.Join(result, "\n"), nil
}

// buildRedirectBlock generates the HTTP-to-HTTPS redirect server block.
func buildRedirectBlock(domain, webRoot string) string {
	return fmt.Sprintf(`%s
server {
    listen 80;
    listen [::]:80;
    server_name %s;

    # Allow ACME challenge for cert renewal
    location /.well-known/acme-challenge/ {
        root %s;
    }

    # Redirect all HTTP to HTTPS
    location / {
        return 301 https://$host$request_uri;
    }
}
%s`, sslRedirectBegin, domain, webRoot, sslRedirectEnd)
}

// removeSSLSections strips SSL marker sections from config.
func removeSSLSections(config string) string {
	// Remove redirect block (between BEGIN/END SSL REDIRECT markers)
	config = removeBetweenMarkers(config, sslRedirectBegin, sslRedirectEnd)

	// Remove SSL directives block (between BEGIN/END SSL markers)
	config = removeBetweenMarkers(config, sslBlockBegin, sslBlockEnd)

	// Clean up extra blank lines left behind
	for strings.Contains(config, "\n\n\n") {
		config = strings.ReplaceAll(config, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(config) + "\n"
}

// removeBetweenMarkers removes text between begin and end markers (inclusive).
func removeBetweenMarkers(text, begin, end string) string {
	startIdx := strings.Index(text, begin)
	if startIdx < 0 {
		return text
	}

	endIdx := strings.Index(text, end)
	if endIdx < 0 {
		return text
	}

	// Include the end marker and any trailing newline
	endIdx += len(end)
	if endIdx < len(text) && text[endIdx] == '\n' {
		endIdx++
	}

	return text[:startIdx] + text[endIdx:]
}
