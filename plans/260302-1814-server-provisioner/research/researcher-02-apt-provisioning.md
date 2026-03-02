# Research: APT Provisioning & LEMP Automation (Go)

Date: 2026-03-02 | Sources: 5 web + official docs

---

## 1. Package Detection: Which Method is Most Reliable?

**Winner: `dpkg-query -W --showformat='${Status}'`**

| Method | Reliable? | Notes |
|--------|-----------|-------|
| `dpkg-query -W` | **YES** | Machine-readable, scriptable, queries dpkg DB directly |
| `dpkg -l` | Partial | Human-oriented output, column widths vary with terminal width, harder to parse |
| `which` / `command -v` | **NO** | Checks PATH only — package may be installed but binary not in PATH, or binary exists but package broken |

**Recommended check in Go (via exec):**

```go
// Returns true if package is installed and in "install ok installed" state
func isPackageInstalled(pkg string) bool {
    out, err := exec.Command("dpkg-query", "-W",
        "--showformat=${Status}", pkg).Output()
    if err != nil {
        return false
    }
    return strings.TrimSpace(string(out)) == "install ok installed"
}
```

Key: `dpkg-query` returns exit code 1 if package not found, so error check alone is sufficient for simple yes/no.

---

## 2. Non-Interactive apt-get Install

**Required env + flags:**

```go
env := append(os.Environ(),
    "DEBIAN_FRONTEND=noninteractive",
    "APT_LISTCHANGES_FRONTEND=none",
)

cmd := exec.Command("apt-get", "install", "-y",
    "-o", "Dpkg::Options::=--force-confdef",
    "-o", "Dpkg::Options::=--force-confold",
    "nginx", "mariadb-server", "redis-server",
)
cmd.Env = env
```

- `DEBIAN_FRONTEND=noninteractive` — suppresses all prompts
- `--force-confdef` — keep existing config on conflict (don't prompt)
- `--force-confold` — keep old config files when upgraded
- Always run `apt-get update` before install

**Full sequence:**
```bash
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  -o Dpkg::Options::=--force-confdef \
  -o Dpkg::Options::=--force-confold \
  nginx mariadb-server redis-server
```

---

## 3. MariaDB Secure Installation via SQL

**Skip `mysql_secure_installation` entirely.** Execute SQL directly:

```go
sql := `
ALTER USER 'root'@'localhost' IDENTIFIED VIA mysql_native_password USING PASSWORD('YOUR_PASSWORD');
DELETE FROM mysql.user WHERE User='';
DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');
DROP DATABASE IF EXISTS test;
DELETE FROM mysql.db WHERE Db='test' OR Db='test\_%';
FLUSH PRIVILEGES;
`
cmd := exec.Command("mysql", "--user=root")
cmd.Stdin = strings.NewReader(sql)
```

**Ubuntu 22/24 + MariaDB 10.6+ note:** Default root uses `unix_socket` auth, so initial connection works without password via `sudo mysql`. After setting password, subsequent connections need `-p`.

**Idempotent SQL (safe to re-run):**

```sql
-- Check if already secured before running
SELECT plugin FROM mysql.user WHERE User='root' AND Host='localhost';
-- If plugin = 'mysql_native_password', skip ALTER USER

-- These are safe to re-run:
DELETE FROM mysql.user WHERE User='';          -- no-op if already done
DROP DATABASE IF EXISTS test;                  -- IF EXISTS = idempotent
DELETE FROM mysql.db WHERE Db='test' OR Db='test\_%';  -- no-op if empty
FLUSH PRIVILEGES;                              -- always safe
```

**Go idempotent check pattern:**
```go
func isMariaDBSecured() bool {
    out, err := exec.Command("mysql", "--user=root", "-e",
        "SELECT COUNT(*) FROM mysql.user WHERE User='' OR (User='root' AND Host NOT IN ('localhost','127.0.0.1','::1'));",
    ).Output()
    if err != nil { return false }
    return strings.Contains(string(out), "0")
}
```

---

## 4. Idempotent Installation Patterns

```go
// Pattern: Check before install
func ensurePackage(pkg string) error {
    if isPackageInstalled(pkg) {
        log.Printf("[SKIP] %s already installed", pkg)
        return nil
    }
    return runAptInstall(pkg)
}

// Pattern: Check before service enable
func ensureServiceEnabled(service string) error {
    out, _ := exec.Command("systemctl", "is-enabled", service).Output()
    if strings.TrimSpace(string(out)) == "enabled" {
        return nil
    }
    return exec.Command("systemctl", "enable", "--now", service).Run()
}
```

**Rule:** Always check state before mutating. `dpkg-query` + `systemctl is-active/is-enabled` cover 90% of cases.

---

## 5. Error Handling for apt Operations

### Lock File Handling

apt has a built-in timeout option (preferred over manual polling):

```go
cmd := exec.Command("apt-get", "install", "-y",
    "-o", "DPkg::Lock::Timeout=120",  // wait up to 120s for lock
    "nginx",
)
```

Manual polling fallback:

```go
func waitForAptLock(timeout time.Duration) error {
    locks := []string{
        "/var/lib/dpkg/lock-frontend",
        "/var/lib/dpkg/lock",
        "/var/cache/apt/archives/lock",
    }
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        locked := false
        for _, lock := range locks {
            out, _ := exec.Command("fuser", lock).Output()
            if len(strings.TrimSpace(string(out))) > 0 {
                locked = true
                break
            }
        }
        if !locked { return nil }
        time.Sleep(5 * time.Second)
    }
    return fmt.Errorf("apt lock not released within %v", timeout)
}
```

### apt-get Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error (check stderr for details) |
| 2 | apt usage error |
| 100 | Package not found |

### Network Failure Retry

```go
func aptInstallWithRetry(packages []string, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := runAptInstall(packages...)
        if err == nil { return nil }
        // Retry only on network-related errors
        if isNetworkError(err) {
            log.Printf("Network error, retry %d/%d", i+1, maxRetries)
            time.Sleep(time.Duration(i+1) * 10 * time.Second) // linear backoff
            continue
        }
        return err // non-network error, fail fast
    }
    return fmt.Errorf("apt-get failed after %d retries", maxRetries)
}

func isNetworkError(err error) bool {
    stderr := extractStderr(err)
    return strings.Contains(stderr, "Unable to connect") ||
           strings.Contains(stderr, "Failed to fetch") ||
           strings.Contains(stderr, "Temporary failure resolving")
}
```

### Dependency Conflict Recovery

```go
// If apt-get install fails with dependency conflict:
exec.Command("apt-get", "install", "-f", "-y").Run()    // fix broken deps
exec.Command("dpkg", "--configure", "-a").Run()          // reconfigure pending
```

---

## Summary: Recommended Go Implementation Order

1. `waitForAptLock(120s)` or use `-o DPkg::Lock::Timeout=120`
2. `apt-get update -qq`
3. For each package: `isPackageInstalled()` → skip or install
4. `DEBIAN_FRONTEND=noninteractive apt-get install -y --force-conf*`
5. `systemctl enable --now nginx/mariadb/redis`
6. `isMariaDBSecured()` → skip or run SQL hardening via `mysql --user=root`

---

## Unresolved Questions

1. MariaDB password auth vs unix_socket — if the Go provisioner runs as root, unix_socket auth works without password; is a root DB password actually needed for this use case?
2. Redis requires no hardening SQL — but does the project need `requirepass` set in `redis.conf` via Go file manipulation?
3. Ubuntu 24.04 ships `mariadb-server` 10.11 — any SQL syntax differences vs 10.6 to verify?

---

## Sources

- [dpkg-query man page - Ubuntu](https://manpages.ubuntu.com/manpages/focal/man1/dpkg-query.1.html)
- [Automating mysql_secure_installation - bertvv](https://bertvv.github.io/notes-to-self/2015/11/16/automating-mysql_secure_installation/)
- [MariaDB Authentication Plugin - Unix Socket](https://mariadb.com/kb/en/authentication-plugin-unix-socket/)
- [Authentication in MariaDB 10.4+](https://mariadb.org/authentication-in-mariadb-10-4/)
- [apt-get lock conflict handling - Medium](https://medium.com/@sroy.sanchita/apt-get-avoiding-lock-conflicts-1d22ae651ea6)
- [Could not get lock /var/lib/dpkg/lock - phoenixnap](https://phoenixnap.com/kb/fix-could-not-get-lock-error-ubuntu)
