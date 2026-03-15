# Phase 4: System Package Tests

## Target Coverage: 26.5% -> >60%

## File: `internal/system/fileops_test.go` (additions)

### New Tests
| Test | Description | Pattern |
|------|-------------|---------|
| `TestRemove` | Create file in `t.TempDir()`, call `Remove`, verify gone | Real FS |
| `TestRemoveDir` | Create dir with files, call `Remove`, verify all gone | Real FS |
| `TestReadFile` | Write file, read back, compare content | Real FS |
| `TestReadFile_NotExist` | Read non-existent path | Expect error |
| `TestWriteAtomic_Permissions` | Write with 0600, verify `os.Stat().Mode()` | Real FS |
| `TestSymlink_OverwriteExisting` | Create symlink, create again to different target | Should replace cleanly |

## File: `internal/system/executor_test.go` (new)

Test the real `execImpl` with safe commands that work on macOS.

| Test | Description | Assertions |
|------|-------------|-----------|
| `TestRun_Echo` | `Run(ctx, "echo", "hello")` | Output = `"hello\n"`, no error |
| `TestRun_FailingCommand` | `Run(ctx, "false")` | Error returned, non-zero exit |
| `TestRun_Timeout` | Set 1ms timeout, run `sleep 10` | Context deadline exceeded error |
| `TestRunWithInput_Cat` | `RunWithInput(ctx, "hello", "cat")` | Output = `"hello"` |
| `TestRun_DefaultTimeout` | No deadline set on context, run `echo test` | Should apply 30s default, succeed |

### Implementation Notes
- Use `slog.New(slog.NewTextHandler(io.Discard, nil))` for test logger
- All commands (`echo`, `false`, `cat`, `sleep`) available on macOS
- Keep tests fast -- timeout test uses 1ms deadline

```go
func testExecutor() Executor {
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    return NewExecutor(logger)
}

func TestRun_Echo(t *testing.T) {
    exec := testExecutor()
    out, err := exec.Run(context.Background(), "echo", "hello")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if strings.TrimSpace(out) != "hello" {
        t.Errorf("got %q, want %q", out, "hello")
    }
}
```

## File: `internal/system/usermgmt_test.go` (new)

Test `userMgmt` via mock Executor -- cannot call real `useradd`/`userdel` on macOS.

| Test | Description | Assertions |
|------|-------------|-----------|
| `TestUserCreate_CallsUseradd` | Mock executor, call Create | Verify `useradd -m -d /home/test -s /bin/bash testuser` recorded |
| `TestUserCreate_AlreadyExists` | Pre-exist user in OS lookup | Error "user already exists" |
| `TestUserDelete_CallsUserdel` | Mock executor, call Delete | Verify `userdel -r testuser` recorded |
| `TestUserDelete_NotExists` | User doesn't exist | No error, no commands (idempotent) |

### Challenge: `Exists` and `LookupUID` Use `user.Lookup`
These call `os/user.Lookup` directly -- not mockable without refactoring. Options:
1. **Skip testing Exists/LookupUID** -- they're trivial wrappers (2 lines each)
2. **Test with current user** -- `user.Lookup(os.Getenv("USER"))` should work on macOS

**Recommendation**: Test `Exists` with current OS user. Skip `LookupUID` mock test since it just parses `user.Lookup` output.

```go
func TestExists_CurrentUser(t *testing.T) {
    exec := newMockExecutor() // won't be called
    mgr := NewUserManager(exec)

    currentUser := os.Getenv("USER")
    if currentUser == "" {
        t.Skip("USER env not set")
    }
    if !mgr.Exists(currentUser) {
        t.Errorf("current user %q should exist", currentUser)
    }
    if mgr.Exists("nonexistent_user_juiscript_test") {
        t.Error("nonexistent user should not exist")
    }
}
```

### Mock Executor for usermgmt tests
Need a test-local mock since `usermgmt.go` is in the `system` package (same package as test):

```go
type testMockExecutor struct {
    commands []string
    failOn   map[string]error
}

func newTestMockExecutor() *testMockExecutor {
    return &testMockExecutor{failOn: make(map[string]error)}
}

func (m *testMockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
    cmd := name + " " + strings.Join(args, " ")
    m.commands = append(m.commands, cmd)
    if err, ok := m.failOn[name]; ok {
        return "", err
    }
    return "", nil
}

func (m *testMockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
    return m.Run(context.Background(), name, args...)
}
```

## Estimated Time: 40 minutes
