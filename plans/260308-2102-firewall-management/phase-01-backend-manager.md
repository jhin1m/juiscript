# Phase 1: Backend Manager

## Context

- Parent: [plan.md](plan.md)
- Dependencies: `internal/system` (Executor)
- Docs: [code-standards.md](../../docs/code-standards.md), [system-architecture.md](../../docs/system-architecture.md)

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-08 |
| Priority | P1 |
| Effort | 2h |
| Status | pending |

Create `internal/firewall/manager.go` with UFW + Fail2ban operations. Follows `internal/service/manager.go` pattern: struct with Executor dependency, input validation, output parsing.

## Key Insights

- UFW `status numbered` gives parseable output: `[ 1] 22/tcp ALLOW IN Anywhere`
- Fail2ban `fail2ban-client status <jail>` gives jail info including banned IP list
- `fail2ban-client status` (no jail) lists all active jails
- Both tools require root (already enforced by juiscript)
- Input validation critical: port range 1-65535, IP via `net.ParseIP`, protocol tcp/udp/both

## Requirements

1. UFW rule listing with parsed structured data
2. UFW enable/disable/allow/deny/delete operations
3. Fail2ban status, ban/unban operations
4. Strong input validation to prevent command injection
5. Unit tests with mock executor (no root needed)

## Architecture

```
internal/firewall/
  manager.go         # Manager struct, UFW + F2b operations
  manager_test.go    # Table-driven tests with mock executor
```

## Data Types

```go
package firewall

// UFWRule represents a single UFW firewall rule.
type UFWRule struct {
    Num    int    // Rule number from `ufw status numbered`
    To     string // e.g. "22/tcp", "80/tcp", "443"
    Action string // "ALLOW IN", "DENY IN", etc.
    From   string // "Anywhere", specific IP, etc.
}

// UFWStatus holds UFW state and rules.
type UFWStatus struct {
    Active bool
    Rules  []UFWRule
}

// F2bJailStatus holds fail2ban jail info.
type F2bJailStatus struct {
    Name      string
    BannedIPs []string
    BanCount  int
}
```

## Related Code Files

- `internal/service/manager.go` - Manager pattern to follow
- `internal/system/executor.go` - Executor interface
- `internal/service/manager_test.go` - Mock executor test pattern

## Implementation Steps

### Step 1: Manager struct

```go
package firewall

import (
    "context"
    "fmt"
    "net"
    "strconv"
    "strings"

    "github.com/jhin1m/juiscript/internal/system"
)

type Manager struct {
    executor system.Executor
}

func NewManager(exec system.Executor) *Manager {
    return &Manager{executor: exec}
}
```

### Step 2: Input validation

```go
// validatePort checks port is 1-65535.
func validatePort(port int) error {
    if port < 1 || port > 65535 {
        return fmt.Errorf("port must be 1-65535, got %d", port)
    }
    return nil
}

// validateIP checks IP address format (IPv4 or IPv6).
func validateIP(ip string) error {
    if net.ParseIP(ip) == nil {
        return fmt.Errorf("invalid IP address: %q", ip)
    }
    return nil
}

// validateProtocol checks protocol is tcp, udp, or both.
func validateProtocol(proto string) error {
    switch proto {
    case "tcp", "udp", "both", "":
        return nil
    default:
        return fmt.Errorf("protocol must be tcp, udp, or both; got %q", proto)
    }
}
```

### Step 3: UFW operations

```go
// Status returns current UFW status and rules.
func (m *Manager) Status(ctx context.Context) (*UFWStatus, error) {
    out, err := m.executor.Run(ctx, "ufw", "status", "numbered")
    if err != nil {
        return nil, fmt.Errorf("ufw status: %w", err)
    }
    return parseUFWStatus(out), nil
}

// Enable enables UFW firewall.
func (m *Manager) Enable(ctx context.Context) error {
    _, err := m.executor.RunWithInput(ctx, "y\n", "ufw", "enable")
    if err != nil {
        return fmt.Errorf("ufw enable: %w", err)
    }
    return nil
}

// Disable disables UFW firewall.
func (m *Manager) Disable(ctx context.Context) error {
    _, err := m.executor.Run(ctx, "ufw", "disable")
    if err != nil {
        return fmt.Errorf("ufw disable: %w", err)
    }
    return nil
}

// AllowPort adds a UFW allow rule for a port.
func (m *Manager) AllowPort(ctx context.Context, port int, proto string) error {
    if err := validatePort(port); err != nil {
        return err
    }
    if err := validateProtocol(proto); err != nil {
        return err
    }
    target := portTarget(port, proto)
    _, err := m.executor.Run(ctx, "ufw", "allow", target)
    if err != nil {
        return fmt.Errorf("ufw allow %s: %w", target, err)
    }
    return nil
}

// DenyPort adds a UFW deny rule for a port.
func (m *Manager) DenyPort(ctx context.Context, port int, proto string) error {
    if err := validatePort(port); err != nil {
        return err
    }
    if err := validateProtocol(proto); err != nil {
        return err
    }
    target := portTarget(port, proto)
    _, err := m.executor.Run(ctx, "ufw", "deny", target)
    if err != nil {
        return fmt.Errorf("ufw deny %s: %w", target, err)
    }
    return nil
}

// DeleteRule removes a UFW rule by number.
func (m *Manager) DeleteRule(ctx context.Context, ruleNum int) error {
    if ruleNum < 1 {
        return fmt.Errorf("rule number must be >= 1, got %d", ruleNum)
    }
    num := strconv.Itoa(ruleNum)
    _, err := m.executor.RunWithInput(ctx, "y\n", "ufw", "delete", num)
    if err != nil {
        return fmt.Errorf("ufw delete rule %d: %w", ruleNum, err)
    }
    return nil
}

// portTarget builds the port/proto string for UFW commands.
func portTarget(port int, proto string) string {
    s := strconv.Itoa(port)
    if proto != "" && proto != "both" {
        s += "/" + proto
    }
    return s
}
```

### Step 4: UFW output parsing

```go
// parseUFWStatus parses `ufw status numbered` output.
// Example output:
//   Status: active
//
//        To                         Action      From
//        --                         ------      ----
//   [ 1] 22/tcp                     ALLOW IN    Anywhere
//   [ 2] 80/tcp                     ALLOW IN    Anywhere
func parseUFWStatus(raw string) *UFWStatus {
    status := &UFWStatus{}
    lines := strings.Split(raw, "\n")

    for _, line := range lines {
        trimmed := strings.TrimSpace(line)

        // Check active/inactive status
        if strings.HasPrefix(trimmed, "Status:") {
            status.Active = strings.Contains(trimmed, "active")
            continue
        }

        // Parse numbered rules: [ N] ...
        if !strings.HasPrefix(trimmed, "[") {
            continue
        }

        rule := parseUFWRule(trimmed)
        if rule != nil {
            status.Rules = append(status.Rules, *rule)
        }
    }
    return status
}

// parseUFWRule parses a single line like "[ 1] 22/tcp  ALLOW IN  Anywhere"
func parseUFWRule(line string) *UFWRule {
    // Extract rule number from brackets
    closeBracket := strings.Index(line, "]")
    if closeBracket < 0 {
        return nil
    }
    numStr := strings.TrimSpace(line[1:closeBracket])
    num, err := strconv.Atoi(numStr)
    if err != nil {
        return nil
    }

    // Rest after "] "
    rest := strings.TrimSpace(line[closeBracket+1:])
    // Split by whitespace groups
    fields := strings.Fields(rest)
    if len(fields) < 3 {
        return nil
    }

    return &UFWRule{
        Num:    num,
        To:     fields[0],
        Action: strings.Join(fields[1:len(fields)-1], " "),
        From:   fields[len(fields)-1],
    }
}
```

### Step 5: Fail2ban operations

```go
// F2bStatus returns status of all fail2ban jails.
func (m *Manager) F2bStatus(ctx context.Context) ([]F2bJailStatus, error) {
    out, err := m.executor.Run(ctx, "fail2ban-client", "status")
    if err != nil {
        return nil, fmt.Errorf("fail2ban status: %w", err)
    }

    jails := parseF2bJailList(out)
    var results []F2bJailStatus
    for _, jail := range jails {
        js, err := m.jailStatus(ctx, jail)
        if err != nil {
            results = append(results, F2bJailStatus{Name: jail})
            continue
        }
        results = append(results, *js)
    }
    return results, nil
}

// jailStatus gets detailed status for a single jail.
func (m *Manager) jailStatus(ctx context.Context, jail string) (*F2bJailStatus, error) {
    out, err := m.executor.Run(ctx, "fail2ban-client", "status", jail)
    if err != nil {
        return nil, fmt.Errorf("fail2ban jail %s: %w", jail, err)
    }
    return parseF2bJailStatus(jail, out), nil
}

// BanIP bans an IP in a fail2ban jail.
func (m *Manager) BanIP(ctx context.Context, ip, jail string) error {
    if err := validateIP(ip); err != nil {
        return err
    }
    if jail == "" {
        jail = "sshd"
    }
    _, err := m.executor.Run(ctx, "fail2ban-client", "set", jail, "banip", ip)
    if err != nil {
        return fmt.Errorf("ban %s in %s: %w", ip, jail, err)
    }
    return nil
}

// UnbanIP unbans an IP from a fail2ban jail.
func (m *Manager) UnbanIP(ctx context.Context, ip, jail string) error {
    if err := validateIP(ip); err != nil {
        return err
    }
    if jail == "" {
        jail = "sshd"
    }
    _, err := m.executor.Run(ctx, "fail2ban-client", "set", jail, "unbanip", ip)
    if err != nil {
        return fmt.Errorf("unban %s from %s: %w", ip, jail, err)
    }
    return nil
}
```

### Step 6: Fail2ban output parsing

```go
// parseF2bJailList parses `fail2ban-client status` to extract jail names.
// Example:
//   |- Number of jail:      2
//   `- Jail list:   sshd, nginx-http-auth
func parseF2bJailList(raw string) []string {
    for _, line := range strings.Split(raw, "\n") {
        if strings.Contains(line, "Jail list:") {
            parts := strings.SplitN(line, ":", 2)
            if len(parts) < 2 {
                return nil
            }
            jailStr := strings.TrimSpace(parts[1])
            if jailStr == "" {
                return nil
            }
            var jails []string
            for _, j := range strings.Split(jailStr, ",") {
                j = strings.TrimSpace(j)
                if j != "" {
                    jails = append(jails, j)
                }
            }
            return jails
        }
    }
    return nil
}

// parseF2bJailStatus parses `fail2ban-client status <jail>` output.
// Example:
//   |- Currently banned: 3
//   `- Banned IP list:   1.2.3.4 5.6.7.8 9.10.11.12
func parseF2bJailStatus(jail, raw string) *F2bJailStatus {
    js := &F2bJailStatus{Name: jail}
    for _, line := range strings.Split(raw, "\n") {
        trimmed := strings.TrimSpace(line)
        if strings.Contains(trimmed, "Currently banned:") {
            parts := strings.SplitN(trimmed, ":", 2)
            if len(parts) == 2 {
                js.BanCount, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
            }
        }
        if strings.Contains(trimmed, "Banned IP list:") {
            parts := strings.SplitN(trimmed, ":", 2)
            if len(parts) == 2 {
                ipStr := strings.TrimSpace(parts[1])
                if ipStr != "" {
                    js.BannedIPs = strings.Fields(ipStr)
                }
            }
        }
    }
    return js
}
```

### Step 7: Unit tests (manager_test.go)

Follow `service/manager_test.go` pattern with mock executor:

```go
func TestValidatePort(t *testing.T) {
    tests := []struct {
        name    string
        port    int
        wantErr bool
    }{
        {"valid min", 1, false},
        {"valid max", 65535, false},
        {"valid http", 80, false},
        {"zero", 0, true},
        {"negative", -1, true},
        {"too high", 65536, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePort(tt.port)
            if (err != nil) != tt.wantErr {
                t.Errorf("validatePort(%d) error = %v, wantErr = %v", tt.port, err, tt.wantErr)
            }
        })
    }
}

func TestValidateIP(t *testing.T) {
    tests := []struct {
        ip      string
        wantErr bool
    }{
        {"192.168.1.1", false},
        {"10.0.0.1", false},
        {"::1", false},
        {"2001:db8::1", false},
        {"not-an-ip", true},
        {"999.999.999.999", true},
        {"", true},
    }
    // ... table-driven
}

func TestParseUFWStatus(t *testing.T) {
    raw := `Status: active

     To                         Action      From
     --                         ------      ----
[ 1] 22/tcp                     ALLOW IN    Anywhere
[ 2] 80/tcp                     ALLOW IN    Anywhere
[ 3] 443/tcp                    ALLOW IN    Anywhere`

    status := parseUFWStatus(raw)
    if !status.Active { t.Error("expected active") }
    if len(status.Rules) != 3 { t.Errorf("got %d rules, want 3", len(status.Rules)) }
    if status.Rules[0].To != "22/tcp" { t.Errorf("rule 1 To = %q", status.Rules[0].To) }
}

func TestParseF2bJailList(t *testing.T) { /* ... */ }
func TestParseF2bJailStatus(t *testing.T) { /* ... */ }
func TestAllowPort(t *testing.T) { /* mock executor, verify command args */ }
func TestBanIP(t *testing.T) { /* mock executor, verify command args */ }
```

## Todo

- [ ] Create `internal/firewall/manager.go` with Manager struct
- [ ] Implement UFW data types (UFWRule, UFWStatus, F2bJailStatus)
- [ ] Implement input validation (port, IP, protocol)
- [ ] Implement UFW operations (Status, Enable, Disable, AllowPort, DenyPort, DeleteRule)
- [ ] Implement UFW output parser (parseUFWStatus, parseUFWRule)
- [ ] Implement Fail2ban operations (F2bStatus, BanIP, UnbanIP)
- [ ] Implement Fail2ban output parsers (parseF2bJailList, parseF2bJailStatus)
- [ ] Create `internal/firewall/manager_test.go` with table-driven tests
- [ ] Ensure 70%+ test coverage

## Success Criteria

- All UFW operations callable with mock executor
- All Fail2ban operations callable with mock executor
- Port validation rejects 0, negative, >65535
- IP validation rejects malformed addresses, accepts IPv4 + IPv6
- Parsers handle real UFW/Fail2ban output formats correctly
- Tests pass without root

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| UFW output format varies across versions | Parser breaks | Test with multiple format samples |
| Fail2ban not installed | Commands fail | Graceful error, TUI shows "not installed" |
| Command injection via port/IP | Security breach | Strict validation before any exec call |

## Security Considerations

- Never pass user input directly to shell; use Executor.Run with args array
- Validate all port numbers (1-65535 integer, not string concatenation)
- Validate all IPs via net.ParseIP (rejects injection attempts)
- Protocol limited to enum: tcp, udp, both
- Rule number validated as positive integer
- No string interpolation in commands

## Next Steps

After completing: proceed to Phase 2 (CLI commands)
