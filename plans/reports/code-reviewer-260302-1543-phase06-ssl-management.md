# Code Review Report — Phase 06 SSL Management

## Scope

- Files reviewed: 6
  - `internal/ssl/manager.go`
  - `internal/ssl/manager_test.go`
  - `internal/nginx/ssl-operations.go`
  - `internal/nginx/ssl-operations_test.go`
  - `internal/tui/screens/ssl.go`
  - `internal/template/templates/nginx-ssl-redirect.conf.tmpl`
- Lines of code analyzed: ~650
- Review focus: Security, architecture alignment, YAGNI/KISS/DRY, error handling, test quality

---

## Overall Assessment

Solid implementation. Follows existing patterns well. Two medium-severity issues and several minor ones.

---

## Critical Issues

None.

---

## High Priority Findings

### H1 — `validateDomain` is insufficient for command injection

**File:** `internal/ssl/manager.go:258`, `internal/nginx/manager.go:228`

Current check only blocks `/` and `..`. A domain like `example.com; rm -rf /` or `$(whoami)` passes validation and gets passed directly to `certbot --cert-name`, `certbot -d`, `openssl -in` (via `filepath.Join`), and `nginx`'s `server_name` directive.

The existing pattern in the project is the same (nginx package has identical `validateDomain`), so this is a pre-existing issue, but the SSL package extends attack surface by introducing `--cert-name`, `--cert-path`, and `--email` args.

**Recommended fix — add allowlist:**
```go
func validateDomain(domain string) error {
    if domain == "" {
        return fmt.Errorf("invalid domain: empty")
    }
    // Allowlist: letters, digits, hyphens, dots only
    for _, r := range domain {
        if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '-' {
            return fmt.Errorf("invalid domain: %q (contains %q)", domain, r)
        }
    }
    return nil
}
```

**Note:** `email` parameter in `Obtain()` has no validation at all. A malformed email is passed directly to `certbot --email`. Add basic format check (must contain `@`).

---

## Medium Priority Improvements

### M1 — Template `nginx-ssl-redirect.conf.tmpl` is unused dead code

**File:** `internal/template/templates/nginx-ssl-redirect.conf.tmpl`

`buildRedirectBlock()` in `ssl-operations.go` generates the redirect block via `fmt.Sprintf` — the template is never rendered. The `template.Engine` embeds `templates/*.tmpl` at compile time (parsed unconditionally), so this file is compiled into the binary but its `{{ .WebRoot }}` field is never populated.

This violates YAGNI: the template exists with no caller. Either:
1. Use the template in `buildRedirectBlock` (pass `WebRoot` from `SSLConfig`/`Obtain`), or
2. Delete the template file.

The template approach is also more consistent with how other nginx configs are generated in this project.

### M2 — `injectSSLDirectives` silently no-ops if `listen [::]:80;` is missing

**File:** `internal/nginx/ssl-operations.go:122`

If the vhost lacks `listen [::]:80;` (e.g., IPv4-only config), SSL directives are never inserted, but the function returns no error and `EnableSSL` succeeds. The written config will have a redirect block but no `listen 443` — nginx test would likely catch it, but the silent skip is misleading.

```go
if !inserted {
    return config, fmt.Errorf("could not find insertion point 'listen [::]:80;' in vhost")
}
```

### M3 — HSTS injected unconditionally, OCSP stapling without `ssl_trusted_certificate`

**File:** `internal/nginx/ssl-operations.go:109-110`

`add_header Strict-Transport-Security "max-age=63072000" always;` is added immediately on SSL enable. HSTS is permanent — if the user later wants to revert or test, browsers will cache it for 2 years. This is aggressive for a management tool.

`ssl_stapling on; ssl_stapling_verify on;` without `ssl_trusted_certificate` will produce nginx warnings (`ssl_stapling` ignored; no resolver defined). Let's Encrypt's chain is included in `fullchain.pem`, so stapling will technically work, but `ssl_stapling_verify on` requires a trusted cert chain or resolver. Low risk but generates log noise.

**Contrast:** `nginx-ssl.conf.tmpl` already has HSTS commented out (`# add_header Strict-Transport-Security`) — the ssl-operations.go hardcodes it uncommented. Inconsistency.

### M4 — Rollback in `EnableSSL` ignores rollback error

**File:** `internal/nginx/ssl-operations.go:57`

```go
m.files.WriteAtomic(availablePath, content, 0644) // error silently discarded
```

Same pattern exists in `nginx/manager.go` (pre-existing), but worth flagging. If rollback write fails, the config is left in a broken state with no indication.

```go
if rbErr := m.files.WriteAtomic(availablePath, content, 0644); rbErr != nil {
    return fmt.Errorf("nginx test failed AND rollback failed (%v): %w", rbErr, err)
}
```

---

## Low Priority Suggestions

### L1 — `parseCertbotCertificates` silently skips unparseable dates

**File:** `internal/ssl/manager.go:238` — `continue` on parse error means a cert with an odd date format is silently dropped from the list. A logged warning or `Issuer` set to "unknown" would be more helpful.

### L2 — `DaysLeft` can be negative but `Valid` is set separately

`info.Valid = info.DaysLeft > 0` (line 188) — `DaysLeft` is `int(hours/24)`, which truncates. A cert expiring in 12 hours has `DaysLeft=0` but is still valid. Consider `info.Valid = time.Until(expiry) > 0` independently.

### L3 — `TestValidateDomain` missing shell-injection cases

Test covers path traversal (`..`, `/`) but not shell metacharacters (`$`, `;`, `|`, backticks). If H1 is fixed with an allowlist these become implicit, but the test should document intent.

### L4 — `GoBackMsg` defined in `dashboard.go`, re-used across all screens including ssl.go

Not a bug — already the established pattern. But if screens are ever extracted to a separate package, `GoBackMsg` would need its own shared location.

### L5 — `buildRedirectBlock` hardcodes `/var/www/html` as ACME challenge root

If the site's webroot differs (e.g., `/var/www/mysite`), certbot renewal challenges may fail. The `webRoot` is available in `Obtain()` but not threaded into `EnableSSL`. Low risk since certbot standalone renewal is the usual renewal path, but worth noting.

---

## Positive Observations

- Architecture is clean and consistent: `ssl.Manager` wraps `nginx.Manager` exactly like other packages wrap each other.
- `validateDomain` called at all entry points — defense in depth even if the check is weak.
- Rollback on nginx test failure is implemented and tested.
- `parseCertOutput` handles both date formats (`Jan  2` and `Jan 2`) — good defensive parsing.
- Test coverage: happy path, failure path, rollback, invalid input, and empty result cases all covered.
- Marker-based SSL injection/removal (`# BEGIN SSL` / `# END SSL`) is idempotent and clean.
- No root required in tests — all system calls mocked via interfaces.
- `nginx-ssl-redirect.conf.tmpl` template is well-formed and safe (no unescaped user data other than `.Domain` and `.WebRoot` which are Go-template auto-escaped... actually `text/template` does NOT HTML-escape, so this is fine for config files).

---

## Recommended Actions

1. **[High]** Replace `validateDomain` with allowlist regex/loop in both `ssl` and `nginx` packages. Add `validateEmail` in `ssl.Obtain`.
2. **[Medium]** Either use `nginx-ssl-redirect.conf.tmpl` in `buildRedirectBlock` or delete it. Current state is dead code.
3. **[Medium]** Return error from `injectSSLDirectives` when insertion point not found.
4. **[Medium]** Log or surface rollback failure in `EnableSSL`/`DisableSSL`.
5. **[Low]** Separate HSTS from OCSP stapling — let user opt-in or match the commented state in `nginx-ssl.conf.tmpl`.
6. **[Low]** Fix `Valid` computation to use `time.Until(expiry) > 0` instead of `DaysLeft > 0`.

---

## Metrics

- Type coverage: N/A (Go, statically typed)
- Test coverage: ~85% estimated (all major paths covered; missing: rollback write failure, `injectSSLDirectives` no-op path)
- Linting issues: 0 known structural issues; 1 discarded error (rollback in ssl-operations.go:57)

---

## Unresolved Questions

- Is `webRoot` intentionally hardcoded to `/var/www/html` in the redirect block, or should it come from site config?
- Should `nginx-ssl-redirect.conf.tmpl` be registered/used anywhere, or was it created for future use and can be deleted now?
- Is HSTS intentional and unconditional by design (matching the nginx-ssl.conf.tmpl which has it commented)?
