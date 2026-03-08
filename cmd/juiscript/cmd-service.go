package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jhin1m/juiscript/internal/service"
	"github.com/spf13/cobra"
)

func serviceCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage system services",
	}
	cmd.AddCommand(
		serviceListCmd(mgrs),
		serviceStatusCmd(mgrs),
		serviceStartCmd(mgrs),
		serviceStopCmd(mgrs),
		serviceRestartCmd(mgrs),
		serviceReloadCmd(mgrs),
	)
	return cmd
}

func serviceListCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all managed services",
		RunE: func(cmd *cobra.Command, args []string) error {
			statuses, err := mgrs.Service.ListAll(context.Background())
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "%-20s %-10s %-10s %-8s %-12s\n",
				"SERVICE", "STATE", "SUB-STATE", "PID", "MEMORY (MB)")
			for _, s := range statuses {
				pid := "-"
				if s.PID > 0 {
					pid = fmt.Sprintf("%d", s.PID)
				}
				mem := "-"
				if s.MemoryMB > 0 {
					mem = fmt.Sprintf("%.1f", s.MemoryMB)
				}
				fmt.Fprintf(os.Stdout, "%-20s %-10s %-10s %-8s %-12s\n",
					s.Name, s.State, s.SubState, pid, mem)
			}
			return nil
		},
	}
}

func serviceStatusCmd(mgrs *Managers) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := mgrs.Service.Status(context.Background(), service.ServiceName(name))
			if err != nil {
				return err
			}
			fmt.Printf("Name:      %s\n", s.Name)
			fmt.Printf("Active:    %v\n", s.Active)
			fmt.Printf("State:     %s\n", s.State)
			fmt.Printf("SubState:  %s\n", s.SubState)
			fmt.Printf("PID:       %d\n", s.PID)
			fmt.Printf("Memory:    %.1f MB\n", s.MemoryMB)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Service name (required)")
	cmd.MarkFlagRequired("name")
	return cmd
}

// serviceActionCmd creates start/stop/restart/reload commands (DRY helper).
func serviceActionCmd(mgrs *Managers, action string, fn func(ctx context.Context, name service.ServiceName) error) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   action,
		Short: fmt.Sprintf("%s a service", action),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fn(context.Background(), service.ServiceName(name)); err != nil {
				return err
			}
			fmt.Printf("Service %sed: %s\n", action, name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Service name (required)")
	cmd.MarkFlagRequired("name")
	return cmd
}

func serviceStartCmd(mgrs *Managers) *cobra.Command {
	return serviceActionCmd(mgrs, "start", mgrs.Service.Start)
}

func serviceStopCmd(mgrs *Managers) *cobra.Command {
	return serviceActionCmd(mgrs, "stop", mgrs.Service.Stop)
}

func serviceRestartCmd(mgrs *Managers) *cobra.Command {
	return serviceActionCmd(mgrs, "restart", mgrs.Service.Restart)
}

func serviceReloadCmd(mgrs *Managers) *cobra.Command {
	return serviceActionCmd(mgrs, "reload", mgrs.Service.Reload)
}
