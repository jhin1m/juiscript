# Code Review: Phase 03 - Nginx/Vhost Management

## Scope
- Files reviewed: 9 files (5 Go, 3 templates, 1 test)
  - `internal/nginx/manager.go` (NEW, 248 lines)
  - `internal/nginx/manager_test.go` (NEW, 441 lines)
  - `internal/site/manager.go` (MODIFIED, 294 lines)
  - `internal/tui/screens/nginx.go` (NEW, 143 lines)
  - `internal/tui/app.go` (MODIFIED, 267 lines)
  - `internal/template/templates/nginx-laravel.conf.tmpl` (NEW)
  - `internal/template/templates/nginx-wordpress.conf.tmpl` (NEW)
  - `internal/template/templates/nginx-ssl.conf.tmpl` (NEW)
  - `internal/template/engine_test.go` (MODIFIED)
- Build: PASS (`go build ./...`)
- Vet: PASS (`go vet ./...`)
- Tests: ALL PASS (nginx: 9/9, site: 14/14, template: 4/4)

## Overall Assessment

Solid, well-structured implementation. Rollback logic is correct and tested. Dependency injection via interfaces is consistent with Phase 01/02 patterns. The duplicated `ProjectType` to resolve import cycle is the only architectural concern worth discussing.

---

## Critical Issues

None.

---

## High Priority Findings

### 1. Path Traversal: `nginx.Manager` public methods accept raw domain strings without validation

`Enable(domain)`, `Disable(domain)`, `Delete(domain)` pass the domain directly to `filepath.Join`:

```go
// internal/nginx/manager.go:215-221
func (m *Manager) availablePath(domain string) string {
    return filepath.Join(m.sitesAvailable, domain+".conf")
}
```

`filepath.Join` cleans `../` sequences, so `../../../etc/nginx/nginx` as domain would escape `sitesAvailable`. The `site.Manager` calls `ValidateDomain` before calling `nginx.Manager.Create`, but `nginx.Manager.Enable/Disable/Delete` are public and can be called directly without validation (e.g., from TUI action handlers when implemented).

**Fix**: Add domain validation in `nginx.Manager` public methods, or at minimum add a private `validateDomain(domain string) error` guard:

```go
func (m *Manager) Delete(domain string) error {
    if strings.Contains(domain, "/") || strings.Contains(domain, "..") {
        return fmt.Errorf("invalid domain: %s", domain)
    }
    // ...
}
```

### 2. `ExtraConfig` is a code injection vector with no sanitization

`ExtraConfig string // raw Nginx directives injected into server block` is rendered verbatim into templates. Any caller can inject arbitrary nginx directives (e.g., `alias /etc/`, `include /proc/self/fd/0`). Currently `buildVhostConfig` in `site/manager.go` leaves it empty (safe), but the field is exported and the test explicitly passes raw directives.

This is intentional by design (per field doc), but should be documented as a trust boundary. Since this is an internal admin tool (root-level CLI), the risk is acceptable — but **callers must be trusted**.

No code fix needed if this is root-only tooling, but add a comment:

```go
ExtraConfig string // raw Nginx directives — caller is responsible for sanitization; trust boundary
```

---

## Medium Priority Improvements

### 3. Duplicated `ProjectType` — import cycle workaround is a DRY violation

`nginx.ProjectType` and `site.ProjectType` are identical string types with identical constants:

```go
// nginx/manager.go:15-20
type ProjectType string
const (
    ProjectLaravel   ProjectType = "laravel"
    ProjectWordPress ProjectType = "wordpress"
)
```

The workaround is noted in comments and the review prompt acknowledges it. The proper fix is to extract `ProjectType` into a shared `internal/types` package (or `internal/config`), but this is a larger refactor. For now, the explicit type conversion at line 213 of `site/manager.go` is acceptable:

```go
ProjectType: nginx.ProjectType(s.ProjectType),
```

**Recommendation**: Track as tech debt. Extract to `internal/types` in a future cleanup phase.

### 4. `nginx.Manager.List()` uses `os.ReadDir` directly (not via `FileManager`)

```go
// internal/nginx/manager.go:174
entries, err := os.ReadDir(m.sitesAvailable)
```

All other FS ops go through `system.FileManager` for testability, but `List` bypasses this. The `TestList` test works around it by creating real temp dirs. This inconsistency means `List` can't be fully mocked.

**Fix**: Add `ReadDir(path string) ([]os.DirEntry, error)` to `system.FileManager` interface, or at minimum document the inconsistency.

### 5. Rollback error results silently discarded in `nginx.Manager.Create`

```go
// internal/nginx/manager.go:96, 103-104
m.files.Remove(availablePath) // rollback: remove config
// ...
m.disable(cfg.Domain)
m.files.Remove(availablePath)
```

Rollback errors are swallowed. If rollback itself fails, the system is left in a partially broken state with no indication. For a CLI admin tool this matters — the operator needs to know.

**Fix**: Log or wrap rollback errors (even if returning original error):

```go
if rbErr := m.disable(cfg.Domain); rbErr != nil {
    return fmt.Errorf("nginx config test failed (rollback also failed: %v): %w", rbErr, err)
}
```

### 6. `site.Manager.Delete()` ignores nginx and PHP-FPM errors

```go
// internal/site/manager.go:149, 155
_ = m.nginx.Delete(domain)
// ...
_ = m.reloadPHPFPM(site.PHPVersion)
```

Silent error discard on delete. If nginx delete fails, the site still gets removed from metadata, leaving orphaned nginx configs. At minimum log these errors; ideally surface them as warnings.

---

## Low Priority Suggestions

### 7. TUI nginx screen actions are all TODO stubs

All three message handlers in `app.go` are `TODO`:

```go
case screens.ToggleVhostMsg:
    // TODO: Call nginx manager to toggle
    return a, nil
```

`app.go` holds no reference to `site.Manager` or `nginx.Manager`, so these can never be implemented without wiring the managers in. This is expected scaffolding for Phase 03, but the `NginxScreen` will display nothing useful until `SetVhosts` is called with real data. Acceptable as intentional placeholder — mark in plan.

### 8. WordPress template adds `www.` alias unconditionally

```
server_name {{ .Domain }} www.{{ .Domain }};
```

This is fine for typical cases but may break sites that don't want the `www` alias (e.g., API subdomains on WordPress). `VhostConfig` has no `AddWWW bool` field. YAGNI applies here — current behavior is correct for the stated scope.

### 9. SSL snippet template is a standalone fragment, not a full template

`nginx-ssl.conf.tmpl` contains partial nginx directives (no `server {}` block) and references `.SSLCertPath`/`.SSLKeyPath` from `VhostConfig`, but the main vhost templates have no `{{ if .SSLEnabled }}{{ template "nginx-ssl.conf.tmpl" . }}{{ end }}` include. It's a placeholder for Phase 06, which is correct per the plan. The field names are consistent with `VhostConfig`.

---

## Positive Observations

- **Rollback correctness**: The three-step rollback (disable symlink → remove config) in `Create` and `Enable` is properly ordered and tested. `TestCreate_RollbackOnNginxTestFailure` specifically verifies both config file and symlink are removed.
- **Interface-driven design**: `system.Executor` and `system.FileManager` injected into `nginx.Manager` enable full mock-based unit tests without requiring a real nginx install.
- **Template security headers**: Both templates include `X-Frame-Options`, `X-Content-Type-Options`, `X-XSS-Protection`, `Referrer-Policy` — better than the plan's spec which omitted them.
- **IPv6 listen**: Both templates include `listen [::]:80` — good practice.
- **Test coverage**: 9 tests covering Create (Laravel, WordPress, default sizes, rollback, extra config, unsupported type), Delete, Enable/Disable with rollback, List, and error parsing. Coverage is meaningful, not just smoke tests.
- **`nginx -t` before reload**: No reload without config test — critical operational safety is properly implemented.
- **`fastcgi_index index.php`**: Added in actual templates vs. plan spec — correct improvement.
- **Font/SVG MIME types in WordPress template**: `svg|woff|woff2|ttf|eot` added vs. plan spec — correct.
- **Type conversion idiom**: `nginx.ProjectType(s.ProjectType)` is clean for the cycle-breaking workaround.

---

## Recommended Actions

1. **(High)** Add domain input validation in `nginx.Manager.Enable/Disable/Delete` — path traversal guard.
2. **(Medium)** Surface rollback errors in `nginx.Manager.Create` — don't silently discard.
3. **(Medium)** Surface nginx/PHP-FPM errors in `site.Manager.Delete` — at least log them.
4. **(Medium)** Add `ReadDir` to `system.FileManager` interface for consistent abstraction, or document the `os.ReadDir` direct usage.
5. **(Low)** Track `ProjectType` duplication as tech debt — extract to `internal/types` in future cleanup.
6. **(Low)** Add trust boundary comment on `ExtraConfig` field.

---

## Metrics

- Type Coverage: 100% (all exported types have godoc)
- Test Coverage: 9/9 nginx tests pass; meaningful behavioral coverage
- Linting Issues: 0 (`go vet` clean)
- Build: PASS

---

## Unresolved Questions

- Q1: Will `nginx.Manager` public methods (`Enable/Disable/Delete`) be callable directly from TUI without going through `site.Manager`? If yes, domain validation in `nginx.Manager` is required (High priority). If all calls flow through `site.Manager`, the existing validation in `site.Manager.Create` is sufficient for Create but not for Enable/Disable/Delete.
- Q2: Is `ExtraConfig` ever expected to be set by end users (TUI input) vs. programmatically? If user-facing, sanitization/allowlist becomes Critical.
