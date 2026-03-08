package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func phpCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "php",
		Short: "Manage PHP versions",
	}
	cmd.AddCommand(
		phpListCmd(mgrs),
		phpInstallCmd(mgrs),
		phpRemoveCmd(mgrs),
	)
	return cmd
}

func phpListCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed PHP versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			versions, err := mgrs.PHP.ListVersions(context.Background())
			if err != nil {
				return err
			}
			if len(versions) == 0 {
				fmt.Println("No PHP versions installed.")
				return nil
			}
			fmt.Fprintf(os.Stdout, "%-10s %-10s %-10s\n", "VERSION", "FPM", "ENABLED")
			for _, v := range versions {
				fpm := "inactive"
				if v.Active {
					fpm = "active"
				}
				enabled := "no"
				if v.Enabled {
					enabled = "yes"
				}
				fmt.Fprintf(os.Stdout, "%-10s %-10s %-10s\n", v.Version, fpm, enabled)
			}
			return nil
		},
	}
}

func phpInstallCmd(mgrs *Managers) *cobra.Command {
	var ver string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a PHP version",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.PHP.InstallVersion(context.Background(), ver); err != nil {
				return err
			}
			fmt.Printf("PHP %s installed\n", ver)
			return nil
		},
	}
	cmd.Flags().StringVar(&ver, "version", "", "PHP version to install, e.g. 8.3 (required)")
	cmd.MarkFlagRequired("version")
	return cmd
}

func phpRemoveCmd(mgrs *Managers) *cobra.Command {
	var ver string
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a PHP version",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Pass nil for activeSites — CLI user is responsible for checking dependencies
			if err := mgrs.PHP.RemoveVersion(context.Background(), ver, nil); err != nil {
				return err
			}
			fmt.Printf("PHP %s removed\n", ver)
			return nil
		},
	}
	cmd.Flags().StringVar(&ver, "version", "", "PHP version to remove, e.g. 8.2 (required)")
	cmd.MarkFlagRequired("version")
	return cmd
}
