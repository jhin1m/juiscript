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

			// Fail2ban status (non-fatal if not installed)
			fmt.Fprintln(os.Stdout, "")
			jails, err := mgrs.Firewall.F2bStatus(ctx)
			if err != nil {
				fmt.Fprintf(os.Stdout, "Fail2ban: %v\n", err)
				return nil
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
					jailName := ""
					if i == 0 {
						jailName = j.Name
					}
					fmt.Fprintf(os.Stdout, "%-20s %-15s\n", jailName, ip)
				}
			}
			return nil
		},
	}
}

// protoDisplay formats protocol for user-facing output.
func protoDisplay(proto string) string {
	if proto == "" || proto == "both" {
		return "tcp+udp"
	}
	return proto
}
