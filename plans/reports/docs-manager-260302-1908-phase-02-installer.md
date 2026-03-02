# Documentation Update: Phase 02 - Package Installation

## Summary
Updated `codebase-summary.md` and `system-architecture.md` to document Phase 02 Installer implementation for server provisioning.

## Files Changed

### codebase-summary.md
1. **Project Structure** (Line ~48-51)
   - Added installer.go entry (196 lines)
   - Added installer_test.go entry (12 tests)

2. **Phase Completion Status** (Line ~510-513)
   - Updated Phase 02 description with Installer details
   - Added test coverage metrics
   - Noted shared isPackageInstalled function

3. **New Section: internal/provisioner/installer.go** (Lines ~851-894)
   - InstallStatus enum & InstallResult struct documentation
   - Installer struct: executor + phpMgr
   - 5 key methods: AptUpdate, InstallNginx, InstallRedis, InstallMariaDB, InstallPHP
   - Helper methods: installSimplePackage, aptInstall, enableAndStart, hardenMariaDB
   - Idempotency pattern explanation
   - Shared function: isPackageInstalled

4. **New Section: internal/provisioner/installer_test.go** (Lines ~895-925)
   - 12 comprehensive test cases documented
   - Test categories: AptUpdate (2), InstallNginx (3), InstallRedis (1), InstallMariaDB (3), InstallPHP (1)
   - Mock executor pattern details
   - SQL command verification for MariaDB hardening

### system-architecture.md
1. **Component Architecture** (Line ~113-145)
   - Updated provisioner package documentation
   - Added Phase 02 Installer section
   - Listed Installer methods with signatures
   - Documented idempotency + hardening features

2. **New Section: Package Installation Implementation** (Lines ~583-707)
   - Installation strategy flowchart
   - Idempotency pattern (check → skip if present → install → enable/start)
   - apt-get configuration details (DEBIAN_FRONTEND, force-conf flags, dpkg lock timeout)
   - MariaDB hardening SQL with security rationale
   - Shared isPackageInstalled function (DRY pattern)
   - PHP delegation to php.Manager (architectural decision)
   - Service management (enable + start via systemctl)
   - Test coverage matrix (12 tests organized by feature)

## Key Documentation Highlights

### Installer Design
- **Idempotent**: All operations check if already installed, skip if present
- **Atomic**: Service enable/start as unit
- **Secure**: MariaDB hardening removes test DB, anonymous users, remote root
- **Composable**: PHP installation delegates to php.Manager for code reuse
- **Robust**: 5-minute timeout per package, comprehensive error wrapping

### Code Refactoring
- Moved isPackageInstalled from detector.go to installer.go
- detector.go refactored to call shared isPackageInstalled
- Benefits: DRY principle, single source of truth for package detection

### Test Coverage
- 12 unit tests covering happy path, failure scenarios, edge cases
- Mock executor pattern for testing without root
- SQL verification for MariaDB hardening

## Architecture Decisions Documented
1. Idempotency: Check before install (skip if present)
2. Shared function: isPackageInstalled used by both Detector + Installer
3. PHP delegation: InstallPHP delegates to php.Manager (avoids duplication)
4. MariaDB hardening: Done after install via SQL (matches existing pattern)
5. Service management: systemctl enable + start (ensures boot startup)

## Phase Progression
- **Phase 01**: Detector ✓
- **Phase 02**: Installer ✓ (NEW)
- **Phase 03**: Site Management (TUI integration pending)

