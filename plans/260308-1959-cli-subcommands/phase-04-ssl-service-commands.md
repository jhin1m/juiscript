# Phase 4: SSL + Service Commands

## File: `cmd/juiscript/cmd-ssl.go`

### Command Tree

```
juiscript ssl list
juiscript ssl obtain --domain example.com --email admin@example.com [--webroot /path]
juiscript ssl revoke --domain example.com
juiscript ssl renew --domain example.com
```

### Subcommand Details

**ssl list**
- Calls `mgrs.SSL.ListCerts()`
- Output table: DOMAIN | EXPIRY | DAYS LEFT | VALID

**ssl obtain --domain X --email X [--webroot /path]**
- Calls `mgrs.SSL.Obtain(domain, webRoot, email)`
- `--webroot` flag: default derives from site config (`cfg.General.SitesRoot + "/" + domain`)
  - If site exists: use `site.Get(domain).WebRoot`
  - If `--webroot` provided: use that value
- Print: "SSL certificate obtained for: example.com"

**ssl revoke --domain X**
- Calls `mgrs.SSL.Revoke(domain)`
- Print: "SSL certificate revoked for: example.com"

**ssl renew --domain X**
- Calls `mgrs.SSL.Renew(domain)`
- Print: "SSL certificate renewed for: example.com"

### API Notes

- `Obtain(domain, webRoot, email)` -- 3 args, webRoot is second
- WebRoot derivation logic: try `mgrs.Site.Get(domain)` first to get actual webroot, fallback to `--webroot` flag, error if neither available

---

## File: `cmd/juiscript/cmd-service.go`

### Command Tree

```
juiscript service list
juiscript service status --name nginx
juiscript service start --name nginx
juiscript service stop --name mariadb
juiscript service restart --name redis-server
juiscript service reload --name nginx
```

### Subcommand Details

**service list**
- Calls `mgrs.Service.ListAll(ctx)`
- Output table: SERVICE | STATE | SUB-STATE | PID | MEMORY

**service status --name X**
- Calls `mgrs.Service.Status(ctx, service.ServiceName(name))`
- Output key-value: Name, Active, State, SubState, PID, Memory

**service start/stop/restart/reload --name X**
- Convert `--name` string to `service.ServiceName`
- Call corresponding method
- Print: "Service started: nginx"

### API Notes

- ServiceName is a string type. CLI accepts raw string, cast to `service.ServiceName(name)`.
- Allowed services enforced by manager's whitelist (nginx, mariadb, redis-server, phpX.Y-fpm).
- Manager returns clear error for disallowed service names.

## Acceptance Criteria

- [ ] All 4 SSL subcommands functional
- [ ] SSL obtain derives webroot from site metadata or flag
- [ ] All 6 service subcommands functional
- [ ] ServiceName whitelist enforced by manager (not CLI)
- [ ] `go build` succeeds
