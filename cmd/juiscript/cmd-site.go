package main

import (
	"fmt"
	"os"

	"github.com/jhin1m/juiscript/internal/site"
	"github.com/spf13/cobra"
)

func siteCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Manage sites",
	}
	cmd.AddCommand(
		siteListCmd(mgrs),
		siteInfoCmd(mgrs),
		siteCreateCmd(mgrs),
		siteDeleteCmd(mgrs),
		siteEnableCmd(mgrs),
		siteDisableCmd(mgrs),
	)
	return cmd
}

func siteListCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all managed sites",
		RunE: func(cmd *cobra.Command, args []string) error {
			sites, err := mgrs.Site.List()
			if err != nil {
				return err
			}
			if len(sites) == 0 {
				fmt.Println("No sites found.")
				return nil
			}
			// Table header
			fmt.Fprintf(os.Stdout, "%-30s %-12s %-6s %-8s %-20s\n",
				"DOMAIN", "TYPE", "PHP", "STATUS", "CREATED")
			for _, s := range sites {
				status := "enabled"
				if !s.Enabled {
					status = "disabled"
				}
				fmt.Fprintf(os.Stdout, "%-30s %-12s %-6s %-8s %-20s\n",
					s.Domain, s.ProjectType, s.PHPVersion, status,
					s.CreatedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
}

func siteInfoCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show site details",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := mgrs.Site.Get(domain)
			if err != nil {
				return err
			}
			fmt.Printf("Domain:      %s\n", s.Domain)
			fmt.Printf("User:        %s\n", s.User)
			fmt.Printf("Type:        %s\n", s.ProjectType)
			fmt.Printf("PHP:         %s\n", s.PHPVersion)
			fmt.Printf("WebRoot:     %s\n", s.WebRoot)
			fmt.Printf("Database:    %s\n", s.DBName)
			fmt.Printf("DB User:     %s\n", s.DBUser)
			fmt.Printf("SSL:         %v\n", s.SSLEnabled)
			fmt.Printf("Status:      %s\n", enabledStr(s.Enabled))
			fmt.Printf("Created:     %s\n", s.CreatedAt.Format("2006-01-02 15:04:05"))
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func siteCreateCmd(mgrs *Managers) *cobra.Command {
	var (
		domain      string
		projectType string
		phpVersion  string
		createDB    bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new site",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := mgrs.Site.Create(site.CreateOptions{
				Domain:      domain,
				ProjectType: site.ProjectType(projectType),
				PHPVersion:  phpVersion,
				CreateDB:    createDB,
			})
			if err != nil {
				return err
			}
			fmt.Printf("Site created: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.Flags().StringVar(&projectType, "type", "laravel", "Project type: laravel or wordpress")
	cmd.Flags().StringVar(&phpVersion, "php", "8.3", "PHP version")
	cmd.Flags().BoolVar(&createDB, "create-db", false, "Create database for site")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func siteDeleteCmd(mgrs *Managers) *cobra.Command {
	var (
		domain   string
		removeDB bool
	)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a site and its resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Site.Delete(domain, removeDB); err != nil {
				return err
			}
			fmt.Printf("Site deleted: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.Flags().BoolVar(&removeDB, "remove-db", false, "Also remove site database")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func siteEnableCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Site.Enable(domain); err != nil {
				return err
			}
			fmt.Printf("Site enabled: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func siteDisableCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Site.Disable(domain); err != nil {
				return err
			}
			fmt.Printf("Site disabled: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

// enabledStr returns human-readable enabled/disabled string.
func enabledStr(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

