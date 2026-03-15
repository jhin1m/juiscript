# Phase 3: Backup Manager Tests

## File: `internal/backup/manager_test.go` (add to existing)
Current coverage: 19.6% -> Target: >70%

## Existing Tests (keep as-is)
- `TestBackupFilename` -- filename generation
- `TestParseBackupFilename` -- table-driven parse
- `TestValidateDomain` -- domain validation
- `TestFormatSize` -- size formatting
- `TestCleanup_KeepLastValidation` -- keepLast < 1
- `TestCronScheduleValidation` -- regex
- `TestIsInsideBackupDir` -- path traversal

## New Tests

### List
| Test | Setup | Assertions |
|------|-------|-----------|
| `TestList_WithBackups` | Create 3 `.tar.gz` files in temp dir with valid names (different timestamps) | Returns 3 items sorted newest-first, correct paths/domains/sizes |
| `TestList_EmptyDir` | Empty temp dir | Returns nil, no error |
| `TestList_NonExistentDir` | Config points to non-existent path | Returns nil, no error (os.IsNotExist handled) |
| `TestList_FiltersByDomain` | Create files for 2 domains | Only returns files for requested domain |

### Delete
| Test | Setup | Assertions |
|------|-------|-----------|
| `TestDelete_Success` | Create real file in temp backup dir | File removed, no error |
| `TestDelete_OutsideBackupDir` | Pass `/etc/passwd` path | Error returned, file untouched |
| `TestDelete_NonExistent` | Pass valid path inside backup dir that doesn't exist | Error "backup not found" |

### Cleanup
| Test | Setup | Assertions |
|------|-------|-----------|
| `TestCleanup_KeepsCorrectNumber` | Create 5 backup files for same domain | After `Cleanup(domain, 2)`, only 2 newest remain |
| `TestCleanup_LessThanKeepLast` | Create 2 files, keepLast=5 | All 2 files remain, no error |

### SetupCron
| Test | Setup | Assertions |
|------|-------|-----------|
| `TestSetupCron_Success` | Mock FileManager | `WriteAtomic` called with correct cron file path and content containing schedule + domain |
| `TestSetupCron_InvalidSchedule` | Pass `"not a schedule"` | Error returned, no write |
| `TestSetupCron_InvalidDomain` | Pass `"../evil"` | Error returned |
| `TestSetupCron_EmptySchedule` | Pass `""` | Error returned |

### RemoveCron
| Test | Setup | Assertions |
|------|-------|-----------|
| `TestRemoveCron_Success` | `mockFileManager.exists[cronFile] = true` | `Remove` called on cron file |
| `TestRemoveCron_NotExists` | File doesn't exist | No error (idempotent) |

### Metadata Roundtrip
| Test | Setup | Assertions |
|------|-------|-----------|
| `TestMetadata_Roundtrip` | Write metadata TOML to temp file, read back | All fields match (Domain, Type, ProjectType, PHPVersion, DBName, DBUser, SiteUser, CreatedAt) |

### Create & Restore (Deferred)
`Create()` and `Restore()` depend on `site.LoadMetadata(config.SitesPath(), domain)` -- a package-level function with a hardcoded config path. Options:
1. Pre-create metadata in the expected path if `config.SitesPath()` is overridable
2. Skip if too coupled -- these methods have the highest system dependency

**Recommendation**: Test `Create` and `Restore` only if `config.SitesPath()` can be overridden. Otherwise defer to integration tests.

## Implementation Pattern

```go
func testConfig(backupDir string) *config.Config {
    return &config.Config{
        Backup: config.BackupConfig{Dir: backupDir},
    }
}

func createTestBackupFiles(t *testing.T, dir, domain string, count int) []string {
    t.Helper()
    var paths []string
    for i := 0; i < count; i++ {
        ts := time.Now().Add(time.Duration(-i) * time.Hour)
        name := backupFilename(domain, ts)
        path := filepath.Join(dir, name)
        os.WriteFile(path, []byte("fake-archive"), 0600)
        paths = append(paths, path)
    }
    return paths
}
```

## Estimated Time: 1 hour
