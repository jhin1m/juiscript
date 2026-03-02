# Brainstorm: Server Provisioner tích hợp

**Date:** 2026-03-02
**Status:** Agreed

## Problem

JuiScript giả định LEMP stack đã cài sẵn. Trên VPS trắng (fresh Ubuntu), tool gần như vô dụng vì không có Nginx/MariaDB/Redis.

## Evaluated Approaches

| Approach | Pros | Cons |
|----------|------|------|
| **A. Provisioner tích hợp (Go)** | Single binary, UX tốt, TUI checklist | Effort trung bình |
| B. Shell bootstrap script | Nhanh, đơn giản | Tách rời khỏi tool, UX kém |
| C. First-run auto-detect full | UX tốt nhất | Phức tạp nhất, overkill |

**Chosen: A** - Provisioner tích hợp trong Go binary.

## Agreed Solution

### Scope
- 4 packages cơ bản: Nginx, PHP (multi-version), MariaDB (OS repo default), Redis
- Certbot, Supervisor, Composer, Node.js → bổ sung sau

### UX Flow
1. Mỗi lần chạy → auto-detect missing packages (`dpkg-query`)
2. Thiếu ≥1 → banner gợi ý "Press 's' for Setup"
3. Setup screen: TUI checklist với trạng thái installed/missing
4. User tick chọn → confirm → cài tự động với progress
5. Done → quay về Dashboard

### Architecture

```
internal/provisioner/
├── detector.go      # Detect installed packages
├── installer.go     # apt-get install per service
└── provisioner.go   # Orchestrator

internal/tui/screens/
└── setup.go         # TUI checklist screen
```

### Key Decisions
- MariaDB: dùng OS default repo, không cho chọn version
- Detection: `dpkg-query -W <package>` (reliable hơn `which`)
- `mysql_secure_installation`: tự động hóa bằng SQL queries
- `apt-get update`: chỉ 1 lần trước khi cài batch
- Error recovery: fail 1 → tiếp tục cài cái khác → báo summary
- Idempotent: skip nếu đã installed

### Estimate
~740 LOC total (detector ~80, installer ~150, provisioner ~60, TUI ~200, integration ~50, tests ~200)

## Next Steps
- Tạo implementation plan chi tiết nếu cần
- PHP install đã có sẵn → tái sử dụng `php.Manager.InstallVersion()`
