# Go Server Management Research: Linux Services

## 1. exec.Command Patterns - Safe Shell Execution

**Key Principle**: Separate commands from arguments; never construct shell strings from user input.

- Use `exec.Command(cmd, arg1, arg2...)` with distinct argument slices—Go handles quoting/escaping
- Avoid shell invocation (`sh -c`) with untrusted input; built-in protections fail with shell
- Use `exec.CommandContext()` for timeouts/cancellation
- Input validation critical before passing to exec
- Handle I/O pipes properly: close stdin pipe to signal EOF, prevent command hangs
- Never use String() method as shell input

**Pitfalls**: Forgetting to close StdinPipe, ignoring stdout/stderr capture, not validating domain names/file paths in command args.

## 2. Service Management (systemctl)

Common patterns for Nginx, PHP-FPM, MariaDB:
```go
exec.Command("systemctl", "start", "nginx")
exec.Command("systemctl", "restart", "php-fpm")
exec.Command("systemctl", "enable", "mariadb.service")
exec.Command("systemctl", "status", "redis")
```

Capture exit codes and stderr for error handling. Check status before operations. Log all commands.

## 3. File Permissions Management

- `os.Chmod(path, 0600)` — set owner-only read/write
- `os.Chown(path, uid, gid)` — change owner (Linux only)
- `os.MkdirAll(dir, 0700)` — create dirs with restrictive perms, then chmod
- `fileInfo.Mode().Perm()` — read current permissions
- Principle: least privilege (0600 for files, 0700 for dirs unless shared)

**Root operations**: When running as root, explicitly chown files after creation. Use `os/user` package to resolve usernames to UIDs.

## 4. User Isolation Model

**Per-Website User Pattern**:
1. Create system user per domain: `useradd -s /usr/sbin/nologin website_user`
2. Create PHP-FPM pool listening on socket owned by that user
3. Nginx reverse-proxies to socket with correct permissions
4. Config files, PHP code owned by that user, Nginx reads them (0644 for config, 0755 for dirs)
5. Database users per website, restricted permissions

**Implementation**: Use `syscall.SysProcAttr` for privilege separation, `syscall.Credential` for UIDs/GIDs, or user namespaces (CLONE_NEWUSER) for isolation. Third-party option: `go-landlock` for filesystem sandboxing.

## 5. Template-Based Config Generation

**Go text/template usage**:
```go
type VhostConfig struct {
    Domain string
    User string
    Socket string
    Root string
}

tmpl := template.Must(template.ParseFiles("nginx.vhost.tmpl"))
var buf bytes.Buffer
tmpl.Execute(&buf, vhostConfig)
os.WriteFile("/etc/nginx/sites-enabled/"+domain+".conf", buf.Bytes(), 0644)
```

- Template files define server block (Nginx) or pool config (PHP-FPM)
- Execute to buffer, validate syntax with nginx -t
- Write only if validation passes
- Use atomic ops: write to temp file, rename, reload config

## 6. Backup Automation

**Database**: Use `mysqldump` via exec.Command:
```go
cmd := exec.Command("mysqldump", "-u", user, "-p"+pass, "--all-databases")
cmd.Stdout = gzip.NewWriter(file)
cmd.Run()
```

- Gzip compression (level 6-9), name with date: `db_$(date +%F).sql.gz`
- Restore: `zcat backup.sql.gz | mysql -u user -p`
- For files: `tar -czf backup.tar.gz /var/www/` via exec
- Atomic: backup to temp, verify, move to final location
- Use database transactions for hot backups (FLUSH TABLES WITH READ LOCK)

## 7. Let's Encrypt / Certbot Automation

**Pattern**: Call certbot via exec.Command:
```go
exec.Command("certbot", "certonly", "--non-interactive", "--webroot", "-w", "/var/www/html",
    "-d", domain, "--email", email, "--agree-tos")
```

- `--non-interactive` prevents prompts
- `--webroot` mode for running servers, `--standalone` for test
- Hook scripts: `--pre-hook`, `--post-hook` for service reload
- Renewal: `certbot renew` cron (daily check, automatic renewal if <30 days left)
- Alternative: Use Boulder ACME library (Go implementation) for direct protocol integration
- Monitor renewal via logs: `/var/log/letsencrypt/`

**Pitfall**: Missing reload hooks—nginx must restart after cert renewal. Test renewals with `--dry-run`.

## 8. Common Pitfalls Summary

1. **Command injection**: User input in command strings → use separate args
2. **Permission leaks**: Config files world-readable, sockets wrong owner → explicit chmod/chown
3. **Resource cleanup**: Unclosed pipes, running goroutines → always defer cleanup
4. **Service restart timing**: Not waiting for service ready → check status/health endpoint
5. **Backup integrity**: Not testing restore, no checksums → verify backups regularly
6. **Cert expiry**: Not monitoring renewal → log renewal, set alerts
7. **Privilege escalation**: Running whole app as root → use setcap, drop privileges early

---

## Sources

- [Go os/exec package](https://pkg.go.dev/os/exec)
- [Go command injection guide (Snyk)](https://snyk.io/blog/understanding-go-command-injection-vulnerabilities/)
- [Secure external command execution (LabEx)](https://labex.io/tutorials/go-how-to-securely-execute-external-commands-in-go-431338)
- [Go file permissions (LabEx)](https://labex.io/tutorials/go-how-to-handle-file-permission-in-go-419741)
- [User isolation & syscalls in Go](https://github.com/shoenig/go-landlock)
- [Database transactions in Go](https://go.dev/doc/database/execute-transactions)
- [Let's Encrypt ACME clients](https://letsencrypt.org/docs/client-options/)
- [Nginx + PHP-FPM config patterns](https://www.digitalocean.com/community/tutorials/php-fpm-nginx)
