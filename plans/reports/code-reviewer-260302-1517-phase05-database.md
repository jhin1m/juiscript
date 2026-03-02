# Code Review: Phase 05 Database Management

## Scope
- Files reviewed: `internal/database/manager.go`, `internal/database/database-operations.go`, `internal/database/user-operations.go`, `internal/database/import-export.go`, `internal/database/manager_test.go`, `internal/tui/screens/database.go`, `internal/tui/app.go` (database integration section)
- Lines of code: ~460
- Review focus: Security, performance, architecture alignment, YAGNI/KISS/DRY
- Build: PASS | Tests: 15/15 PASS | Coverage: 81.6%

## Overall Assessment

Solid implementation. Core security concerns (SQL injection, system DB protection, crypto/rand passwords) are correctly handled. Architecture matches existing patterns. Two critical security issues in `import-export.go` require immediate fix. The TUI integration is intentionally incomplete (TODO stubs in app.go) which is acceptable for this phase.

---

## Critical Issues

### 1. Shell Injection in `import-export.go` - filePath not sanitized

**File:** `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/database/import-export.go`

```go
// Line 34 - filePath is unvalidated user input injected into a shell command
cmd = fmt.Sprintf("gunzip -c %s | mysql -u root %s", filePath, dbName)
// Line 36
cmd = fmt.Sprintf("mysql -u root %s < %s", dbName, filePath)
// Line 60-61 - same pattern in Export
cmd = fmt.Sprintf("mysqldump -u root --single-transaction --quick %s | gzip > %s", dbName, outputPath)
cmd = fmt.Sprintf("mysqldump -u root --single-transaction --quick %s > %s", dbName, outputPath)
```

`dbName` is validated by regex but `filePath`/`outputPath` is not. A path like `/tmp/dump.sql; rm -rf /` or with embedded backticks would execute arbitrary shell commands via `bash -c`. The `os.Stat` check only verifies existence, not safety.

**Fix required:** Quote paths with `%q`-style shell escaping or use `shellescape`-equivalent. At minimum, reject paths containing shell metacharacters (`; | & $ \`` etc.) before building the command string. A simple approach: validate `filePath` contains no chars outside `[a-zA-Z0-9._/\-]`.

### 2. `outputPath` in Export allows arbitrary write location

`Export` accepts any `outputPath` without restriction. A caller could pass `/etc/cron.d/evil` and the mysqldump output would be written there as root. This is a privilege escalation vector.

**Fix required:** Validate that `outputPath` is within an expected directory (e.g., home directory or configured backup path), or at minimum reject paths under `/etc`, `/usr`, `/bin`, etc.

---

## High Priority

### 3. Password exposed in SQL command logged via executor

**File:** `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/database/user-operations.go` lines 24-29, 70-72

The `exec()` method calls `m.executor.Run(ctx, "mysql", ..., sql)` where `sql` contains the plaintext password in `IDENTIFIED BY '{password}'`. The `execImpl` in `system/executor.go` logs all args:

```go
// executor.go line 67-72
e.logger.Info("exec",
    "cmd", name,
    "args", args,  // <-- args includes the SQL with password in plaintext
    ...
)
```

Passwords will appear in logs in plaintext. This is an information disclosure vulnerability.

**Fix required:** Either pass the password via stdin using `RunWithInput` (MariaDB supports `--password` from stdin), or strip sensitive SQL from log args. The `RunWithInput` path already exists in the `Executor` interface but is unused here.

---

## Medium Priority

### 4. `ListDBs` parses tab-separated output with `strings.Fields` (whitespace split)

**File:** `/Users/jhin1m/Desktop/ducanh-project/juiscript/internal/database/database-operations.go` lines 44-66

`strings.Fields` splits on any whitespace. MariaDB output with `-N` flag is tab-separated. If a database name somehow contained spaces (impossible given current nameRegex, but worth noting), this would silently produce wrong data. Low actual risk given validation, but `strings.Split(line, "\t")` would be more precise and self-documenting.

### 5. `importTimeout` constant reused for Export with wrong name context

**File:** `import-export.go` line 54

```go
ctx, cancel := context.WithTimeout(ctx, importTimeout) // used in Export()
```

The constant is named `importTimeout` but reused for export. Minor readability issue - rename to `opTimeout` or `dbOpTimeout`.

### 6. `CreateUser` - no rollback on partial failure

**File:** `user-operations.go` lines 24-29

Three statements (`CREATE USER`, `GRANT ALL`, `FLUSH PRIVILEGES`) are sent as a single batch. If `GRANT ALL` fails after `CREATE USER` succeeds, the user exists without privileges and without the caller knowing. MariaDB multi-statement execution does not guarantee atomic rollback.

**Acceptable risk** given this is a management tool running as root, but worth documenting in a comment. The existing `IF NOT EXISTS` on `CREATE USER` helps with idempotency but not partial failure cleanup.

### 7. `mockExecutor` in test uses `failOn` keyed by command name, not full command

**File:** `manager_test.go` lines 25-27

```go
if err, ok := m.failOn[name]; ok { // "name" is just "mysql", "bash", etc.
```

Cannot distinguish between different calls to the same binary (e.g., fail `mysql` for `CreateDB` but not `ListDBs`). The other packages' mock executors (e.g., php package pattern in code-standards) use the full command string as key. This limits test precision - though current tests are adequate for this phase.

---

## Low Priority

### 8. `View()` string concatenation with `+=` in loop

**File:** `internal/tui/screens/database.go` lines 103-119

```go
var rows string
for idx, db := range d.dbs {
    rows += row + "\n"  // O(n²) string concat
}
```

Use `strings.Builder`. Negligible for typical DB counts (<100), but inconsistent with Go idioms.

### 9. `GoBackMsg` defined in `database.go` but used generically

`GoBackMsg` is in the `screens` package `database.go` file but is a generic navigation message likely shared with other screens. If it's already defined elsewhere, this is a duplicate. If not, it should be in a shared `messages.go` file - consistent with how other screens handle navigation.

---

## Positive Observations

- `validateName` regex (`^[a-z][a-z0-9_]{0,63}$`) is strict and correct - backtick injection test case explicitly covered.
- System DB blocklist is applied at both `DropDB` and `ListDBs` levels - defense in depth.
- `crypto/rand` with `math/big.Int` for password generation is correct - no modulo bias.
- `GeneratePassword` enforces minimum length of 8 silently - acceptable.
- `-N` flag on `mysql` CLI suppresses column headers - correct for output parsing.
- `--single-transaction --quick` on `mysqldump` is the right combination for live exports.
- `os.Stat` check before import prevents misleading error messages.
- 81.6% test coverage exceeds the 70% project standard.
- All methods accept `context.Context` - consistent with codebase patterns.
- Mock executor implements full `Executor` interface correctly.

---

## Task Completeness (vs Plan TODO)

| Task | Status |
|------|--------|
| Manager with socket auth | DONE |
| CreateDB / DropDB | DONE |
| CreateUser / DropUser / ResetPassword | DONE |
| ListDBs with size info | DONE |
| Import (plain + gzip) | DONE |
| Export with gzip | DONE |
| Password generator | DONE |
| TUI database screen | DONE (display only) |
| Unit tests with mock executor | DONE |
| Wire DB manager into app.go handlers | NOT DONE (TODO stubs remain) |

The wiring of `CreateDBMsg`, `DropDBMsg`, `ImportDBMsg`, `ExportDBMsg` handlers in `app.go` is all stubs. This is the expected state per phase scope (TUI screen exists, actions will be wired in a later phase or follow-up), but worth confirming this is intentional.

---

## Recommended Actions

1. **[Critical]** Sanitize `filePath`/`outputPath` in `Import`/`Export` before shell interpolation - reject shell metacharacters.
2. **[Critical]** Restrict `outputPath` in `Export` to safe directories.
3. **[High]** Use `RunWithInput` to pass passwords to MariaDB via stdin instead of embedding in SQL string that gets logged.
4. **[Medium]** Document partial-failure behavior of `CreateUser` in a code comment.
5. **[Low]** Replace `strings.Fields` with `strings.Split(line, "\t")` in `ListDBs`.
6. **[Low]** Rename `importTimeout` to `dbOpTimeout` or similar.

---

## Metrics
- Build: PASS (0 errors, 0 warnings)
- `go vet`: PASS
- Test coverage: 81.6% (exceeds 70% standard)
- Tests: 15/15 PASS
- Linting issues: 0 via `go vet`

---

## Unresolved Questions
1. Are the `app.go` TODO stubs (`CreateDBMsg`, `DropDBMsg`, etc.) intentionally deferred to a later phase, or expected to be completed in Phase 05?
2. Where will DB credentials (generated passwords) be persisted? Plan mentions "site TOML with 0600 perms" but no persistence code exists yet - is this Phase 06 scope?
