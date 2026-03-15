---
title: "Test Coverage Boost: backup, system, site"
description: "Increase test coverage from ~20-36% to >70% for 3 critical packages via mock-based orchestration tests"
status: completed
priority: P2
effort: 3h
branch: main
tags: [testing, coverage, mock, quality]
created: 2026-03-10
---

# Test Coverage Boost Plan

## Goal
Raise coverage for `site`, `backup`, and `system` packages from 19-36% to >70% using mock-based unit tests. No root access required.

## Current State

| Package | Before | After | Status |
|---------|--------|-------|--------|
| `internal/site` | 36.5% | 82.9% | DONE |
| `internal/backup` | 19.6% | 78.3% | DONE |
| `internal/system` | 26.5% | 77.1% | DONE |

## Existing Mock Patterns (Reuse)
Every test package (nginx, database, php, ssl, service, supervisor, cache, firewall) duplicates the same `mockExecutor` and `mockFileManager`. Pattern:
- `mockExecutor`: records `commands []string`, has `failOn map[string]error`, optional `output string`
- `mockFileManager`: tracks `written map[string][]byte`, `symlinks`, `exists`, `failOn`

No shared mock file exists -- each package defines its own. **Phase 1 creates package-local mocks for site, backup, system** following this pattern.

## Phases

| Phase | File | Tests | Status | Completed |
|-------|------|-------|--------|-----------|
| 1 | Mock infrastructure | Mocks for site & backup packages | DONE | 2026-03-10 |
| 2 | `internal/site/manager_test.go` | ~13 tests | DONE | 2026-03-10 |
| 3 | `internal/backup/manager_test.go` (additions) | ~12 tests | DONE | 2026-03-10 |
| 4 | `internal/system/` new test files | ~8 tests | DONE | 2026-03-10 |

## Key Constraints
- All tests must run on macOS (no Ubuntu-only commands in assertions)
- Use `t.TempDir()` for file operations
- Table-driven tests per Go convention
- Follow existing mock patterns in codebase
- No third-party mock libs (consistent with project)

## Risk & Mitigation

| Risk | Mitigation |
|------|-----------|
| `site.Manager.Create` calls `config.SitesPath()` (package-level) | Use `t.Setenv` or temp dir for metadata path |
| `backup.Create/Restore` call `site.LoadMetadata` | Pre-create temp metadata files in test setup |
| `system.UserManager` uses `user.Lookup` (OS call) | Test via mock Executor only; skip Exists/LookupUID (OS-bound) |
| `os.MkdirAll` in `createDirs` uses real FS | Use `t.TempDir()` as sitesRoot |

## Bonus Fixes
- FPM template bug: Added missing fields to php-fpm pool template (MaxRequests, MemoryLimit, UploadMaxSize, Timezone)
- Config refactor: `config.SitesPath()` now testable as receiver method; `EnsureDirs` fixed to use receiver method

## Completion Summary (2026-03-10)
- Coverage targets exceeded: site +46.4%, backup +58.7%, system +50.6%
- All 4 phases completed and reviewed
- Report: `plans/reports/code-reviewer-260310-1526-step2-test-coverage.md`
