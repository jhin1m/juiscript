package main

import (
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/provisioner"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/system"
	"github.com/jhin1m/juiscript/internal/tui"
	"github.com/spf13/cobra"
)

// Build-time variables injected via ldflags in Makefile.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "juiscript",
		Short: "LEMP server management TUI",
		Long:  "juiscript - Manage Nginx, PHP-FPM, MariaDB, Redis on Ubuntu with a beautiful TUI",
		// Default action: launch TUI
		RunE: runTUI,
	}

	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runTUI launches the Bubble Tea TUI application.
func runTUI(cmd *cobra.Command, args []string) error {
	logger := slog.Default()
	exec := system.NewExecutor(logger)
	svcMgr := service.NewManager(exec)
	prov := provisioner.NewProvisioner(exec, nil) // PHP manager created later when needed
	app := tui.NewApp(svcMgr, prov)

	// tea.WithAltScreen uses the alternate terminal buffer
	// so the TUI doesn't mess up your terminal history
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("juiscript %s (commit: %s)\n", version, commit)
		},
	}
}
