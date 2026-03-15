# Phase 2: Site Manager Tests

## File: `internal/site/manager_test.go` (new)
Current coverage: 36.5% -> Target: >70%

## Test List

### Create -- Happy Path
| Test | Description | Key Assertions |
|------|-------------|---------------|
| `TestCreate_Laravel_Success` | Full create flow for Laravel site | User created, dirs exist (via `t.TempDir`), FPM pool written via `files.written`, nginx vhost created, php-fpm reload called, metadata saved |
| `TestCreate_WordPress_Success` | Same for WordPress | WebRoot uses `public_html` path, `www.` alias in vhost |

### Create -- Validation Errors
| Test | Description |
|------|-------------|
| `TestCreate_InvalidDomain` | Empty, no TLD, starts with hyphen -> error, no side effects |
| `TestCreate_InvalidProjectType` | `"flask"` -> error |
| `TestCreate_InvalidPHPVersion` | `"abc"`, `""` -> error |
| `TestCreate_UserAlreadyExists` | Pre-populate `mockUserManager.users` -> `"site user already exists"` |

### Create -- Rollback Scenarios
Each test injects a failure at one step and verifies prior steps are rolled back.

| Test | Failure Point | Rollback Verified |
|------|--------------|-------------------|
| `TestCreate_RollbackOnUserCreateFail` | `mockUserManager.failOn["create"]` | No dirs, no files, no nginx |
| `TestCreate_RollbackOnDirCreateFail` | Use a non-writable `SitesRoot` path | User deleted |
| `TestCreate_RollbackOnFPMPoolFail` | `mockFileManager.failOn["write"]` | User deleted |
| `TestCreate_RollbackOnNginxFail` | `mockExecutor.failOn["nginx"]` (nginx -t fails) | User deleted, FPM pool removed |
| `TestCreate_RollbackOnPHPReloadFail` | `mockExecutor.failOn["systemctl"]` | User deleted, FPM pool removed, nginx vhost deleted |

### Delete
| Test | Description |
|------|-------------|
| `TestDelete_Success` | Pre-save metadata to temp dir, call Delete, verify nginx.Delete called, FPM pool removed, user deleted, metadata gone |
| `TestDelete_SiteNotFound` | No metadata file -> error |

### Enable / Disable
| Test | Description |
|------|-------------|
| `TestEnable_Success` | Pre-save metadata, call Enable, verify `nginx.Enable` called, metadata updated `Enabled=true` |
| `TestDisable_Success` | Same pattern, verify `Enabled=false` |

### buildVhostConfig
| Test | Description |
|------|-------------|
| `TestBuildVhostConfig` | Create Site struct, call `buildVhostConfig`, verify all VhostConfig fields mapped correctly (Domain, WebRoot, PHPSocket, AccessLog, ErrorLog, ProjectType, MaxBodySize) |

## Implementation Notes

### Metadata Path Issue
`Manager.Create` calls `SaveMetadata(config.SitesPath(), site)`. `config.SitesPath()` returns `/etc/juiscript/sites/`. For tests, two options:
1. **Preferred**: Check if `config.SitesPath()` is overridable. If it reads from a global var or func, override in test.
2. **Fallback**: Skip metadata save verification in Create tests; it's already tested in `metadata_test.go`.

Need to inspect `config.SitesPath()` implementation before coding.

### createDirs Uses Real FS
`createDirs` calls `os.MkdirAll` directly (not via FileManager interface). Tests must use `t.TempDir()` as `SitesRoot` so dirs actually get created in temp space.

### nginx.Manager Is Constructed Internally
`site.NewManager` constructs `nginx.NewManager` internally -- we can't inject a mock nginx.Manager. But the nginx.Manager itself uses our mocked Executor and FileManager, so failures propagate correctly.

## Estimated Time: 1 hour
