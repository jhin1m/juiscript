package ssl

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/jhin1m/juiscript/internal/nginx"
	"github.com/jhin1m/juiscript/internal/system"
)

// CertInfo holds SSL certificate metadata for display.
type CertInfo struct {
	Domain   string
	Expiry   time.Time
	Issuer   string
	Valid    bool
	DaysLeft int
}

// letsencryptBase is the default certbot certificate directory.
const letsencryptBase = "/etc/letsencrypt/live"

// Manager handles Let's Encrypt SSL certificate operations via certbot.
type Manager struct {
	executor system.Executor
	nginx    *nginx.Manager
	files    system.FileManager
}

// NewManager creates an SSL manager.
func NewManager(exec system.Executor, nginxMgr *nginx.Manager, files system.FileManager) *Manager {
	return &Manager{
		executor: exec,
		nginx:    nginxMgr,
		files:    files,
	}
}

// Obtain requests a new SSL certificate using certbot webroot method.
// Steps: run certbot -> update nginx vhost with SSL -> reload nginx.
func (m *Manager) Obtain(domain, webRoot, email string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	if err := validateEmail(email); err != nil {
		return err
	}

	// Run certbot with webroot method for ACME challenge validation
	args := []string{
		"certonly",
		"--non-interactive",
		"--agree-tos",
		"--webroot",
		"-w", webRoot,
		"-d", domain,
		"--email", email,
	}

	if _, err := m.executor.Run(context.Background(), "certbot", args...); err != nil {
		return fmt.Errorf("certbot obtain: %w", err)
	}

	// Update Nginx vhost to enable SSL
	sslCfg := nginx.SSLConfig{
		CertPath: filepath.Join(letsencryptBase, domain, "fullchain.pem"),
		KeyPath:  filepath.Join(letsencryptBase, domain, "privkey.pem"),
		WebRoot:  webRoot,
	}

	if err := m.nginx.EnableSSL(domain, sslCfg); err != nil {
		return fmt.Errorf("enable ssl in vhost: %w", err)
	}

	return nil
}

// Revoke revokes an existing certificate and disables SSL in the vhost.
func (m *Manager) Revoke(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	certPath := filepath.Join(letsencryptBase, domain, "cert.pem")
	args := []string{
		"revoke",
		"--cert-path", certPath,
		"--non-interactive",
	}

	if _, err := m.executor.Run(context.Background(), "certbot", args...); err != nil {
		return fmt.Errorf("certbot revoke: %w", err)
	}

	// Clean up certbot files for this domain
	deleteArgs := []string{"delete", "--cert-name", domain, "--non-interactive"}
	if _, err := m.executor.Run(context.Background(), "certbot", deleteArgs...); err != nil {
		return fmt.Errorf("certbot delete: %w", err)
	}

	// Disable SSL in Nginx vhost
	if err := m.nginx.DisableSSL(domain); err != nil {
		return fmt.Errorf("disable ssl in vhost: %w", err)
	}

	return nil
}

// Renew forces renewal of a certificate.
func (m *Manager) Renew(domain string) error {
	if err := validateDomain(domain); err != nil {
		return err
	}

	args := []string{
		"renew",
		"--cert-name", domain,
		"--force-renewal",
	}

	if _, err := m.executor.Run(context.Background(), "certbot", args...); err != nil {
		return fmt.Errorf("certbot renew: %w", err)
	}

	return nil
}

// Status returns certificate information for a specific domain.
// Parses cert using openssl to get expiry, issuer, and validity.
func (m *Manager) Status(domain string) (*CertInfo, error) {
	if err := validateDomain(domain); err != nil {
		return nil, err
	}

	certPath := filepath.Join(letsencryptBase, domain, "fullchain.pem")

	// Check if cert file exists
	if !m.files.Exists(certPath) {
		return nil, fmt.Errorf("certificate not found for domain: %s", domain)
	}

	// Parse cert with openssl
	output, err := m.executor.Run(context.Background(),
		"openssl", "x509", "-in", certPath, "-noout", "-dates", "-issuer")
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	return parseCertOutput(domain, output)
}

// ListCerts returns info for all certificates managed by certbot.
func (m *Manager) ListCerts() ([]CertInfo, error) {
	output, err := m.executor.Run(context.Background(), "certbot", "certificates")
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}

	return parseCertbotCertificates(output)
}

// parseCertOutput extracts cert metadata from openssl x509 output.
// Expected output format:
//
//	notBefore=Mar  1 00:00:00 2026 GMT
//	notAfter=May 30 00:00:00 2026 GMT
//	issuer=CN = R3, O = Let's Encrypt, C = US
func parseCertOutput(domain, output string) (*CertInfo, error) {
	info := &CertInfo{Domain: domain}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "notAfter="):
			dateStr := strings.TrimPrefix(line, "notAfter=")
			expiry, err := time.Parse("Jan  2 15:04:05 2006 GMT", dateStr)
			if err != nil {
				// Try alternate format (single digit day without extra space)
				expiry, err = time.Parse("Jan 2 15:04:05 2006 GMT", dateStr)
				if err != nil {
					return nil, fmt.Errorf("parse expiry date %q: %w", dateStr, err)
				}
			}
			info.Expiry = expiry
			info.DaysLeft = int(time.Until(expiry).Hours() / 24)
			info.Valid = time.Until(expiry) > 0

		case strings.HasPrefix(line, "issuer="):
			info.Issuer = strings.TrimPrefix(line, "issuer=")
		}
	}

	if info.Expiry.IsZero() {
		return nil, fmt.Errorf("could not parse certificate expiry for %s", domain)
	}

	return info, nil
}

// parseCertbotCertificates parses `certbot certificates` output.
// Format:
//
//	Certificate Name: example.com
//	  Domains: example.com
//	  Expiry Date: 2026-05-30 00:00:00+00:00 (VALID: 89 days)
//	  Certificate Path: /etc/letsencrypt/live/example.com/fullchain.pem
func parseCertbotCertificates(output string) ([]CertInfo, error) {
	var certs []CertInfo
	var current *CertInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case strings.HasPrefix(line, "Certificate Name:"):
			if current != nil {
				certs = append(certs, *current)
			}
			current = &CertInfo{
				Domain: strings.TrimSpace(strings.TrimPrefix(line, "Certificate Name:")),
			}

		case current != nil && strings.HasPrefix(line, "Expiry Date:"):
			// Format: "2026-05-30 00:00:00+00:00 (VALID: 89 days)"
			dateAndStatus := strings.TrimPrefix(line, "Expiry Date:")
			dateAndStatus = strings.TrimSpace(dateAndStatus)

			// Extract just the date part (before the parenthesis)
			if idx := strings.Index(dateAndStatus, " ("); idx > 0 {
				dateStr := dateAndStatus[:idx]
				expiry, err := time.Parse("2006-01-02 15:04:05-07:00", dateStr)
				if err != nil {
					expiry, err = time.Parse("2006-01-02 15:04:05+00:00", dateStr)
					if err != nil {
						continue // skip unparseable dates
					}
				}
				current.Expiry = expiry
				current.DaysLeft = int(time.Until(expiry).Hours() / 24)
				current.Valid = strings.Contains(dateAndStatus, "VALID")
			}
		}
	}

	// Don't forget the last cert
	if current != nil {
		certs = append(certs, *current)
	}

	return certs, nil
}

// validateDomain guards against path traversal and command injection.
// Only allows letters, digits, dots, and hyphens (valid DNS characters).
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("invalid domain: empty")
	}
	if strings.Contains(domain, "..") {
		return fmt.Errorf("invalid domain: %q (contains consecutive dots)", domain)
	}
	for _, r := range domain {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '-' {
			return fmt.Errorf("invalid domain: %q (contains %q)", domain, r)
		}
	}
	return nil
}

// validateEmail performs basic email validation to prevent injection.
func validateEmail(email string) error {
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email: %q", email)
	}
	for _, r := range email {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) &&
			r != '@' && r != '.' && r != '-' && r != '_' && r != '+' {
			return fmt.Errorf("invalid email: %q (contains %q)", email, r)
		}
	}
	return nil
}
