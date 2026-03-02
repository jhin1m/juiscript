# Phase 06: SSL Management

## Context
Let's Encrypt SSL certificate management via certbot. Automates obtaining, renewing, and revoking certificates. Updates Nginx vhosts to serve HTTPS.

## Overview
- **Effort**: 3h
- **Priority**: P2
- **Status**: done
- **Depends on**: Phase 01, Phase 03

## Key Insights
- `certbot --non-interactive` for automation; `--nginx` plugin handles Nginx config modification.
- Alternative: `certbot certonly --webroot -w {webroot} -d {domain}` for manual Nginx SSL config control.
- Webroot method preferred (more predictable than `--nginx` plugin which modifies configs).
- Certs at `/etc/letsencrypt/live/{domain}/fullchain.pem` and `privkey.pem`.
- Auto-renewal via systemd timer (certbot installs this by default).
- HTTP-to-HTTPS redirect after cert obtained.

## Requirements
1. Obtain SSL cert for a domain (webroot method)
2. Revoke SSL cert
3. Check cert status/expiry
4. Update Nginx vhost for SSL (add ssl server block, redirect HTTP)
5. Force-renew a cert
6. TUI screen for SSL status and actions

## Architecture

### SSL Package (`internal/ssl/`)
```go
type CertInfo struct {
    Domain    string
    Expiry    time.Time
    Issuer    string
    Valid     bool
    DaysLeft  int
}

type Manager struct {
    executor system.Executor
    nginx    *nginx.Manager
    tpl      *template.Engine
}

func (m *Manager) Obtain(domain, webRoot, email string) error
func (m *Manager) Revoke(domain string) error
func (m *Manager) Renew(domain string) error
func (m *Manager) Status(domain string) (*CertInfo, error)
func (m *Manager) ListCerts() ([]CertInfo, error)
```

### Obtain Flow
1. Ensure Nginx vhost exists and is serving HTTP (for ACME challenge)
2. Run `certbot certonly --non-interactive --agree-tos --webroot -w {webroot} -d {domain} --email {email}`
3. On success: update Nginx vhost to include SSL server block (listen 443, certs paths)
4. Add HTTP-to-HTTPS redirect to port 80 block
5. Test Nginx config, reload
6. Update site metadata: `ssl_enabled = true`

### SSL Nginx Snippet Template
```nginx
listen 443 ssl http2;
ssl_certificate /etc/letsencrypt/live/{{ .Domain }}/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/{{ .Domain }}/privkey.pem;
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:...;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 1d;
ssl_stapling on;
ssl_stapling_verify on;
```

## Related Files
```
internal/ssl/manager.go
internal/ssl/manager_test.go
internal/tui/screens/ssl.go
templates/nginx-ssl-block.conf.tmpl
templates/nginx-ssl-redirect.conf.tmpl
```

## Implementation Steps

1. **Manager struct**: Inject executor, nginx manager, template engine
2. **Obtain()**: Run certbot webroot, update vhost, reload Nginx
3. **Revoke()**: `certbot revoke --cert-path /etc/letsencrypt/live/{domain}/cert.pem --non-interactive`
4. **Renew()**: `certbot renew --cert-name {domain} --force-renewal`
5. **Status()**: Parse cert with `openssl x509 -in {cert} -noout -dates -issuer`
6. **ListCerts()**: `certbot certificates` parsed for all managed domains
7. **SSL Nginx templates**: SSL block snippet, HTTP redirect snippet
8. **Vhost update**: Regenerate full vhost with SSL sections included
9. **TUI SSL screen**: List certs with expiry, status (valid/expiring/expired), obtain/revoke actions

## Todo
- [x] SSL Manager with certbot wrapper
- [x] Obtain with webroot method
- [x] Revoke and renew
- [x] Cert status parsing
- [x] SSL Nginx templates
- [x] Vhost update for SSL
- [x] TUI SSL screen
- [x] Tests

## Success Criteria
- Obtain cert for a domain, Nginx serves HTTPS afterward
- HTTP requests redirect to HTTPS
- Cert status shows expiry date accurately
- Revoke removes cert and disables SSL in vhost

## Risk Assessment
| Risk | Impact | Mitigation |
|------|--------|------------|
| Rate limiting by Let's Encrypt | Medium | Use staging for testing; warn user about limits |
| DNS not pointing to server | High | Pre-check: resolve domain, compare with server IP |
| certbot not installed | Low | Check on startup, prompt to install |

## Security Considerations
- TLS 1.2+ only (no SSLv3, TLS 1.0/1.1)
- OCSP stapling enabled
- HSTS header added after SSL confirmed working
- Cert private keys readable only by root (Let's Encrypt default)

## Next Steps
Phase 07 (Service Control) provides the service reload mechanisms used here.
