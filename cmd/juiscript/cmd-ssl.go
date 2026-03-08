package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func sslCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates",
	}
	cmd.AddCommand(
		sslListCmd(mgrs),
		sslObtainCmd(mgrs),
		sslRevokeCmd(mgrs),
		sslRenewCmd(mgrs),
	)
	return cmd
}

func sslListCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List SSL certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			certs, err := mgrs.SSL.ListCerts()
			if err != nil {
				return err
			}
			if len(certs) == 0 {
				fmt.Println("No SSL certificates found.")
				return nil
			}
			fmt.Fprintf(os.Stdout, "%-30s %-20s %-10s %-6s\n",
				"DOMAIN", "EXPIRY", "DAYS LEFT", "VALID")
			for _, c := range certs {
				valid := "yes"
				if !c.Valid {
					valid = "no"
				}
				fmt.Fprintf(os.Stdout, "%-30s %-20s %-10d %-6s\n",
					c.Domain, c.Expiry.Format("2006-01-02 15:04"), c.DaysLeft, valid)
			}
			return nil
		},
	}
}

func sslObtainCmd(mgrs *Managers) *cobra.Command {
	var (
		domain  string
		email   string
		webroot string
	)
	cmd := &cobra.Command{
		Use:   "obtain",
		Short: "Obtain SSL certificate via Let's Encrypt",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Derive webroot from site metadata if not explicitly provided
			if webroot == "" {
				s, err := mgrs.Site.Get(domain)
				if err != nil {
					return fmt.Errorf("cannot determine webroot: provide --webroot or ensure site exists: %w", err)
				}
				webroot = s.WebRoot
			}
			if err := mgrs.SSL.Obtain(domain, webroot, email); err != nil {
				return err
			}
			fmt.Printf("SSL certificate obtained for: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain name (required)")
	cmd.Flags().StringVar(&email, "email", "", "Email for Let's Encrypt (required)")
	cmd.Flags().StringVar(&webroot, "webroot", "", "Web root path (auto-detected from site if omitted)")
	cmd.MarkFlagRequired("domain")
	cmd.MarkFlagRequired("email")
	return cmd
}

func sslRevokeCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke SSL certificate",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.SSL.Revoke(domain); err != nil {
				return err
			}
			fmt.Printf("SSL certificate revoked for: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain name (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func sslRenewCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "renew",
		Short: "Renew SSL certificate",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.SSL.Renew(domain); err != nil {
				return err
			}
			fmt.Printf("SSL certificate renewed for: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain name (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}
