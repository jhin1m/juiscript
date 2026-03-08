package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jhin1m/juiscript/internal/backup"
	"github.com/spf13/cobra"
)

func backupCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage backups",
	}
	cmd.AddCommand(
		backupListCmd(mgrs),
		backupCreateCmd(mgrs),
		backupRestoreCmd(mgrs),
		backupDeleteCmd(mgrs),
		backupCleanupCmd(mgrs),
		backupCronSetupCmd(mgrs),
		backupCronRemoveCmd(mgrs),
	)
	return cmd
}

func backupListCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups for a domain",
		RunE: func(cmd *cobra.Command, args []string) error {
			backups, err := mgrs.Backup.List(domain)
			if err != nil {
				return err
			}
			if len(backups) == 0 {
				fmt.Printf("No backups found for %s.\n", domain)
				return nil
			}
			fmt.Fprintf(os.Stdout, "%-60s %-10s %-20s\n", "PATH", "SIZE", "CREATED")
			for _, b := range backups {
				fmt.Fprintf(os.Stdout, "%-60s %-10s %-20s\n",
					b.Path, backup.FormatSize(b.Size),
					b.CreatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func backupCreateCmd(mgrs *Managers) *cobra.Command {
	var (
		domain     string
		backupType string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := mgrs.Backup.Create(context.Background(), backup.Options{
				Domain: domain,
				Type:   backup.BackupType(backupType),
			})
			if err != nil {
				return err
			}
			fmt.Printf("Backup created: %s (%s)\n", info.Path, backup.FormatSize(info.Size))
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.Flags().StringVar(&backupType, "type", "full", "Backup type: full, files, or database")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func backupRestoreCmd(mgrs *Managers) *cobra.Command {
	var (
		path   string
		domain string
	)
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore a site from backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Backup.Restore(context.Background(), path, domain); err != nil {
				return err
			}
			fmt.Printf("Backup restored: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Path to backup archive (required)")
	cmd.Flags().StringVar(&domain, "domain", "", "Target domain to restore to (required)")
	cmd.MarkFlagRequired("path")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func backupDeleteCmd(mgrs *Managers) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a backup file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Backup.Delete(path); err != nil {
				return err
			}
			fmt.Printf("Backup deleted: %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Path to backup archive (required)")
	cmd.MarkFlagRequired("path")
	return cmd
}

func backupCleanupCmd(mgrs *Managers) *cobra.Command {
	var (
		domain string
		keep   int
	)
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove old backups, keeping only the N most recent",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Backup.Cleanup(domain, keep); err != nil {
				return err
			}
			fmt.Printf("Cleanup complete: keeping last %d backups for %s\n", keep, domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.Flags().IntVar(&keep, "keep", 5, "Number of recent backups to keep")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func backupCronSetupCmd(mgrs *Managers) *cobra.Command {
	var (
		domain   string
		schedule string
	)
	cmd := &cobra.Command{
		Use:   "cron-setup",
		Short: "Set up scheduled backup cron job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Backup.SetupCron(domain, schedule); err != nil {
				return err
			}
			fmt.Printf("Cron job created for: %s (%s)\n", domain, schedule)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.Flags().StringVar(&schedule, "schedule", "", "Cron schedule, e.g. \"0 2 * * *\" (required)")
	cmd.MarkFlagRequired("domain")
	cmd.MarkFlagRequired("schedule")
	return cmd
}

func backupCronRemoveCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "cron-remove",
		Short: "Remove scheduled backup cron job",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Backup.RemoveCron(domain); err != nil {
				return err
			}
			fmt.Printf("Cron job removed for: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}
