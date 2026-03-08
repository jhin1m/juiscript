package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jhin1m/juiscript/internal/supervisor"
	"github.com/spf13/cobra"
)

func queueCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "Manage queue workers (Supervisor)",
	}
	cmd.AddCommand(
		queueListCmd(mgrs),
		queueCreateCmd(mgrs),
		queueDeleteCmd(mgrs),
		queueStartCmd(mgrs),
		queueStopCmd(mgrs),
		queueRestartCmd(mgrs),
		queueStatusCmd(mgrs),
	)
	return cmd
}

func queueListCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all queue workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			workers, err := mgrs.Super.ListAll(context.Background())
			if err != nil {
				return err
			}
			if len(workers) == 0 {
				fmt.Println("No queue workers found.")
				return nil
			}
			fmt.Fprintf(os.Stdout, "%-40s %-10s %-8s %-12s\n",
				"NAME", "STATE", "PID", "UPTIME")
			for _, w := range workers {
				pid := "-"
				if w.PID > 0 {
					pid = fmt.Sprintf("%d", w.PID)
				}
				uptime := "-"
				if w.Uptime > 0 {
					uptime = fmtDuration(w.Uptime)
				}
				fmt.Fprintf(os.Stdout, "%-40s %-10s %-8s %-12s\n",
					w.Name, w.State, pid, uptime)
			}
			return nil
		},
	}
}

func queueCreateCmd(mgrs *Managers) *cobra.Command {
	var cfg supervisor.WorkerConfig
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a queue worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Super.Create(context.Background(), cfg); err != nil {
				return err
			}
			fmt.Printf("Queue worker created for: %s\n", cfg.Domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&cfg.Domain, "domain", "", "Site domain (required)")
	cmd.Flags().StringVar(&cfg.Username, "username", "", "Linux user to run as (required)")
	cmd.Flags().StringVar(&cfg.SitePath, "site-path", "", "Laravel project root path (required)")
	cmd.Flags().StringVar(&cfg.PHPBinary, "php", "", "PHP binary path, e.g. /usr/bin/php8.3 (required)")
	cmd.Flags().StringVar(&cfg.Connection, "connection", "redis", "Queue connection type")
	cmd.Flags().StringVar(&cfg.Queue, "queue", "default", "Queue name")
	cmd.Flags().IntVar(&cfg.Processes, "processes", 1, "Number of worker processes")
	cmd.Flags().IntVar(&cfg.Tries, "tries", 3, "Max retry attempts per job")
	cmd.Flags().IntVar(&cfg.MaxTime, "max-time", 3600, "Max seconds before worker restart")
	cmd.Flags().IntVar(&cfg.Sleep, "sleep", 3, "Seconds to sleep when no jobs")
	cmd.MarkFlagRequired("domain")
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("site-path")
	cmd.MarkFlagRequired("php")
	return cmd
}

func queueDeleteCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a queue worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Super.Delete(context.Background(), domain); err != nil {
				return err
			}
			fmt.Printf("Queue worker deleted for: %s\n", domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

// queueDomainActionCmd creates start/stop/restart commands (DRY helper).
func queueDomainActionCmd(mgrs *Managers, action string, fn func(ctx context.Context, domain string) error) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   action,
		Short: fmt.Sprintf("%s a queue worker", action),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fn(context.Background(), domain); err != nil {
				return err
			}
			fmt.Printf("Queue worker %sed: %s\n", action, domain)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func queueStartCmd(mgrs *Managers) *cobra.Command {
	return queueDomainActionCmd(mgrs, "start", mgrs.Super.Start)
}

func queueStopCmd(mgrs *Managers) *cobra.Command {
	return queueDomainActionCmd(mgrs, "stop", mgrs.Super.Stop)
}

func queueRestartCmd(mgrs *Managers) *cobra.Command {
	return queueDomainActionCmd(mgrs, "restart", mgrs.Super.Restart)
}

func queueStatusCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show queue worker status",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := mgrs.Super.Status(context.Background(), domain)
			if err != nil {
				return err
			}
			fmt.Printf("Name:    %s\n", w.Name)
			fmt.Printf("State:   %s\n", w.State)
			fmt.Printf("PID:     %d\n", w.PID)
			fmt.Printf("Uptime:  %s\n", fmtDuration(w.Uptime))
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

// fmtDuration formats duration as "Xh Ym" or "Xm Ys".
func fmtDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
