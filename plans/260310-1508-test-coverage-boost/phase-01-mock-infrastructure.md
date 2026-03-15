# Phase 1: Mock Infrastructure

## Objective
Create package-local mock implementations for `site` and `backup` test files. Follow the exact pattern from `nginx/manager_test.go` and `database/manager_test.go`.

## Why Not a Shared Mock Package?
Existing codebase duplicates mocks per package (12 packages do this). Creating a shared `internal/testutil/` would be a refactor outside this plan's scope. Keep consistent with codebase convention.

## Deliverables

### 1. `internal/site/manager_test.go` -- mock section (top of file)

```go
// mockExecutor -- same pattern as nginx/manager_test.go
type mockExecutor struct {
    commands []string
    failOn   map[string]error
    outputs  map[string]string // key: command name, value: output
}

func newMockExecutor() *mockExecutor {
    return &mockExecutor{
        failOn:  make(map[string]error),
        outputs: make(map[string]string),
    }
}

func (m *mockExecutor) Run(_ context.Context, name string, args ...string) (string, error) {
    cmd := name + " " + strings.Join(args, " ")
    m.commands = append(m.commands, cmd)
    if err, ok := m.failOn[name]; ok {
        return "", err
    }
    if out, ok := m.outputs[name]; ok {
        return out, nil
    }
    return "", nil
}

func (m *mockExecutor) RunWithInput(_ context.Context, _ string, name string, args ...string) (string, error) {
    return m.Run(context.Background(), name, args...)
}
```

```go
// mockFileManager -- same as nginx/manager_test.go
type mockFileManager struct {
    written  map[string][]byte
    symlinks map[string]string
    exists   map[string]bool
    failOn   map[string]error
}
// (same methods as nginx mock)
```

```go
// mockUserManager -- new, specific to site package
type mockUserManager struct {
    users    map[string]string // username -> homeDir
    failOn   map[string]error
    uid, gid int
}

func newMockUserManager() *mockUserManager {
    return &mockUserManager{
        users:  make(map[string]string),
        failOn: make(map[string]error),
        uid:    1001,
        gid:    1001,
    }
}

func (u *mockUserManager) Create(username, homeDir string) error {
    if err, ok := u.failOn["create"]; ok { return err }
    u.users[username] = homeDir
    return nil
}

func (u *mockUserManager) Delete(username string) error {
    if err, ok := u.failOn["delete"]; ok { return err }
    delete(u.users, username)
    return nil
}

func (u *mockUserManager) Exists(username string) bool {
    _, ok := u.users[username]
    return ok
}

func (u *mockUserManager) LookupUID(username string) (int, int, error) {
    if _, ok := u.users[username]; !ok {
        return 0, 0, fmt.Errorf("user not found: %s", username)
    }
    return u.uid, u.gid, nil
}
```

### 2. `internal/backup/manager_test.go` -- mock additions

Add `mockExecutor` and `mockFileManager` to existing test file (same pattern). Also add a `mockDatabaseManager` or use `nil` for `db` field where DB not needed.

```go
// For tests that don't touch DB (List, Delete, Cleanup, Cron):
mgr := &Manager{
    config:   testConfig(tmpDir),
    executor: newMockExecutor(),
    files:    newMockFileManager(),
    db:       nil, // not needed for these tests
}
```

### 3. Helper: `setupTestSiteManager`

```go
func setupTestSiteManager(t *testing.T) (*Manager, *mockExecutor, *mockFileManager, *mockUserManager) {
    t.Helper()
    tpl, err := template.New()
    if err != nil { t.Fatalf("template: %v", err) }

    exec := newMockExecutor()
    files := newMockFileManager()
    users := newMockUserManager()
    cfg := &config.Config{
        General: config.GeneralConfig{SitesRoot: t.TempDir()},
        Nginx:   config.NginxConfig{
            SitesAvailable: "/etc/nginx/sites-available",
            SitesEnabled:   "/etc/nginx/sites-enabled",
        },
    }

    mgr := NewManager(cfg, exec, files, users, tpl)
    return mgr, exec, files, users
}
```

## Estimated Time: 20 minutes
