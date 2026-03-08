# Phase 2: CLI Commands

## Context

- Parent: [plan.md](plan.md)
- Dependencies: Phase 1 (backend manager)
- Pattern: `cmd/juiscript/cmd-service.go`

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-08 |
| Priority | P1 |
| Effort | 1h |
| Status | done |

Create `cmd/juiscript/cmd-firewall.go` with Cobra subcommands. Follows `cmd-service.go` pattern: command group + subcommands, flags, table output.

## Key Insights

- `cmd-service.go` uses a DRY helper `serviceActionCmd` for repeated patterns
- All commands receive `*Managers` for backend access
- Flags use `cmd.Flags().StringVar` with required marking
- Table output uses `fmt.Fprintf(os.Stdout, ...)` with column alignment

## Requirements

1. `juiscript firewall status` - UFW status + rules table + Fail2ban summary
2. `juiscript firewall open-port --port N [--protocol tcp|udp|both]`
3. `juiscript firewall close-port --port N [--protocol tcp|udp|both]`
4. `juiscript firewall block-ip --ip IP [--jail sshd]`
5. `juiscript firewall unblock-ip --ip IP [--jail sshd]`
6. `juiscript firewall list-blocked` - all banned IPs across jails

## Architecture

```
cmd/juiscript/
  cmd-firewall.go    # firewallCmd() + subcommands
```

## Related Code Files

- `cmd/juiscript/cmd-service.go` - CLI pattern to follow
- `cmd/juiscript/main.go` - command registration

## Implementation Steps

### Step 1: Command group

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

func firewallCmd(mgrs *Managers) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "firewall",
        Short: "Manage firewall rules and IP blocking",
    }
    cmd.AddCommand(
        firewallStatusCmd(mgrs),
        firewallOpenPortCmd(mgrs),
        firewallClosePortCmd(mgrs),
        firewallBlockIPCmd(mgrs),
        firewallUnblockIPCmd(mgrs),
        firewallListBlockedCmd(mgrs),
    )
    return cmd
}
```

### Step 2: Status command

```go
func firewallStatusCmd(mgrs *Managers) *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show firewall status, rules, and blocked IPs",
        RunE: func(cmd *cobra.Command, args []string) error {
            ctx := context.Background()

            // UFW status
            ufwStatus, err := mgrs.Firewall.Status(ctx)
            if err != nil {
                return err
            }

            activeStr := "inactive"
            if ufwStatus.Active {
                activeStr = "active"
            }
            fmt.Fprintf(os.Stdout, "UFW Status: %s\n\n", activeStr)

            if len(ufwStatus.Rules) > 0 {
                fmt.Fprintf(os.Stdout, "%-6s %-20s %-15s %-15s\n", "NUM", "TO", "ACTION", "FROM")
                for _, r := range ufwStatus.Rules {
                    fmt.Fprintf(os.Stdout, "%-6d %-20s %-15s %-15s\n",
                        r.Num, r.To, r.Action, r.From)
                }
            } else {
                fmt.Fprintln(os.Stdout, "No UFW rules configured.")
            }

            // Fail2ban status
            fmt.Fprintln(os.Stdout, "")
            jails, err := mgrs.Firewall.F2bStatus(ctx)
            if err != nil {
                fmt.Fprintf(os.Stdout, "Fail2ban: %v\n", err)
                return nil // non-fatal; fail2ban may not be installed
            }

            fmt.Fprintln(os.Stdout, "Fail2ban Jails:")
            fmt.Fprintf(os.Stdout, "%-20s %-10s\n", "JAIL", "BANNED")
            for _, j := range jails {
                fmt.Fprintf(os.Stdout, "%-20s %-10d\n", j.Name, j.BanCount)
            }
            return nil
        },
    }
}
```

### Step 3: Port commands

```go
func firewallOpenPortCmd(mgrs *Managers) *cobra.Command {
    var port int
    var proto string
    cmd := &cobra.Command{
        Use:   "open-port",
        Short: "Allow traffic on a port",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := mgrs.Firewall.AllowPort(context.Background(), port, proto); err != nil {
                return err
            }
            fmt.Printf("Port %d/%s opened\n", port, protoDisplay(proto))
            return nil
        },
    }
    cmd.Flags().IntVar(&port, "port", 0, "Port number (required)")
    cmd.Flags().StringVar(&proto, "protocol", "both", "Protocol: tcp, udp, or both")
    cmd.MarkFlagRequired("port")
    return cmd
}

func firewallClosePortCmd(mgrs *Managers) *cobra.Command {
    var port int
    var proto string
    cmd := &cobra.Command{
        Use:   "close-port",
        Short: "Deny traffic on a port",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := mgrs.Firewall.DenyPort(context.Background(), port, proto); err != nil {
                return err
            }
            fmt.Printf("Port %d/%s closed\n", port, protoDisplay(proto))
            return nil
        },
    }
    cmd.Flags().IntVar(&port, "port", 0, "Port number (required)")
    cmd.Flags().StringVar(&proto, "protocol", "both", "Protocol: tcp, udp, or both")
    cmd.MarkFlagRequired("port")
    return cmd
}

func protoDisplay(proto string) string {
    if proto == "" || proto == "both" {
        return "tcp+udp"
    }
    return proto
}
```

### Step 4: IP block commands

```go
func firewallBlockIPCmd(mgrs *Managers) *cobra.Command {
    var ip, jail string
    cmd := &cobra.Command{
        Use:   "block-ip",
        Short: "Ban an IP address via Fail2ban",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := mgrs.Firewall.BanIP(context.Background(), ip, jail); err != nil {
                return err
            }
            fmt.Printf("IP %s banned in jail %s\n", ip, jail)
            return nil
        },
    }
    cmd.Flags().StringVar(&ip, "ip", "", "IP address to ban (required)")
    cmd.Flags().StringVar(&jail, "jail", "sshd", "Fail2ban jail name")
    cmd.MarkFlagRequired("ip")
    return cmd
}

func firewallUnblockIPCmd(mgrs *Managers) *cobra.Command {
    var ip, jail string
    cmd := &cobra.Command{
        Use:   "unblock-ip",
        Short: "Unban an IP address from Fail2ban",
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := mgrs.Firewall.UnbanIP(context.Background(), ip, jail); err != nil {
                return err
            }
            fmt.Printf("IP %s unbanned from jail %s\n", ip, jail)
            return nil
        },
    }
    cmd.Flags().StringVar(&ip, "ip", "", "IP address to unban (required)")
    cmd.Flags().StringVar(&jail, "jail", "sshd", "Fail2ban jail name")
    cmd.MarkFlagRequired("ip")
    return cmd
}
```

### Step 5: List blocked command

```go
func firewallListBlockedCmd(mgrs *Managers) *cobra.Command {
    return &cobra.Command{
        Use:   "list-blocked",
        Short: "List all banned IPs across Fail2ban jails",
        RunE: func(cmd *cobra.Command, args []string) error {
            jails, err := mgrs.Firewall.F2bStatus(context.Background())
            if err != nil {
                return err
            }

            fmt.Fprintf(os.Stdout, "%-20s %-15s\n", "JAIL", "BANNED IP")
            for _, j := range jails {
                if len(j.BannedIPs) == 0 {
                    fmt.Fprintf(os.Stdout, "%-20s %-15s\n", j.Name, "(none)")
                    continue
                }
                for i, ip := range j.BannedIPs {
                    jail := ""
                    if i == 0 {
                        jail = j.Name
                    }
                    fmt.Fprintf(os.Stdout, "%-20s %-15s\n", jail, ip)
                }
            }
            return nil
        },
    }
}
```

## Todo

- [ ] Create `cmd/juiscript/cmd-firewall.go`
- [ ] Implement `firewallCmd()` group with 6 subcommands
- [ ] Implement `firewallStatusCmd` (UFW + Fail2ban combined view)
- [ ] Implement `firewallOpenPortCmd` and `firewallClosePortCmd` with --port and --protocol flags
- [ ] Implement `firewallBlockIPCmd` and `firewallUnblockIPCmd` with --ip and --jail flags
- [ ] Implement `firewallListBlockedCmd`
- [ ] Add `protoDisplay` helper

## Success Criteria

- `juiscript firewall status` shows combined UFW + Fail2ban info
- Port commands validate input before execution
- IP commands default to sshd jail
- Table output aligned with consistent column widths
- Fail2ban errors non-fatal in status command (may not be installed)

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Fail2ban not installed | block-ip/unblock-ip fail | Clear error message; status shows graceful fallback |
| UFW not installed | All UFW commands fail | Error message suggests installing UFW |

## Security Considerations

- Port flag is `int` type (Cobra handles parsing, prevents string injection)
- IP validated in backend manager before command execution
- Jail name passed as argument to Executor.Run (no shell interpolation)

## Next Steps

After completing: proceed to Phase 3 (TUI screen)
