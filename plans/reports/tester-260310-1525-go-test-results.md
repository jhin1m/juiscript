# Go Test Suite Report
**Date:** 2026-03-10 15:25
**Command:** `go test ./... -count=1 -timeout 120s`

---

## Test Results Overview

**Total Packages:** 17
**Packages with Tests:** 14
**Packages Passed:** 13
**Packages Failed:** 1
**Packages with No Tests:** 3 (cmd/juiscript, internal/tui, internal/tui/screens, internal/tui/theme)

**Overall Status:** FAILED

---

## Package Results

| Package | Status | Duration | Notes |
|---------|--------|----------|-------|
| backup | PASS | 0.17s | - |
| cache | PASS | 0.23s | - |
| config | PASS | 0.33s | - |
| database | PASS | 0.44s | - |
| firewall | PASS | 0.54s | - |
| nginx | PASS | 0.66s | - |
| php | PASS | 0.79s | - |
| **provisioner** | **FAIL** | **0.92s** | **1 test failure** |
| service | PASS | 1.01s | - |
| site | PASS | 1.06s | - |
| ssl | PASS | 1.04s | - |
| supervisor | PASS | 1.05s | - |
| system | PASS | 1.07s | - |
| template | PASS | 1.04s | - |
| tui/components | PASS | 0.95s | - |

---

## Failed Test Details

### Package: `internal/provisioner`

**Test:** `TestDetectAll_PHPPlaceholderWhenNotInstalled`
**Location:** `detector_test.go:271`, `detector_test.go:274`

**Failure Details:**
```
detector_test.go:271: expected DisplayName=PHP, got "PHP 7.4"
detector_test.go:274: expected empty Package for PHP placeholder, got "php7.4-fpm"
```

**Root Cause Analysis:**

The test expects:
- When NO PHP versions are installed, the detector should show a placeholder "PHP" entry
- DisplayName should be exactly "PHP" (not "PHP 7.4")
- Package field should be empty (not "php7.4-fpm")

Current implementation:
- The `DetectAll()` function iterates through `defaultPHPVersions = []string{"7.4", "8.0", "8.1", "8.2", "8.3", "8.4"}`
- For each version, it creates a PackageInfo with `DisplayName: "PHP " + ver` and `Package: "php" + ver + "-fpm"`
- There is no placeholder entry showing just "PHP" when none are installed

**Expected Behavior vs Actual:**
- Expected: Single "PHP" placeholder when no PHP installed, allowing user to select versions
- Actual: Multiple PHP version entries (PHP 7.4, PHP 8.0, etc.) are always shown regardless of install status

**Issue Scope:**
- Affects `Detector.DetectAll()` method in `detector.go`
- Test expectations need alignment with current implementation OR implementation needs modification to provide a placeholder entry
- This represents a mismatch between test design and feature implementation

---

## Compilation Status
✓ All packages compile successfully. No compilation errors.

---

## Test Execution Summary
- **Total Duration:** ~14.5 seconds
- **Packages Executed:** 14
- **Tests Passed:** All tests except 1 in provisioner
- **No flaky test indicators:** Test failed consistently on rerun

---

## Unresolved Questions

1. Should DetectAll() include a placeholder "PHP" entry (as test expects), or should it always show specific versions (as current implementation does)?
2. What is the intended UX for the case when no PHP is installed—should user see a single "PHP" option or all default versions?
3. Is the test outdated relative to the implementation, or does the implementation need revision?
