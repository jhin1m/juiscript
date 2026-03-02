package main

import (
	"fmt"
	"io"
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
	// Write logs to file instead of terminal to avoid breaking TUI display.
	// Logs go to /var/log/juiscript.log for debugging, not stdout/stderr.
	logFile, err := os.OpenFile("/var/log/juiscript.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Fallback: discard logs if we can't open log file (e.g. no permission)
		logFile = nil
	}

	var logger *slog.Logger
	if logFile != nil {
		defer logFile.Close()
		logger = slog.New(slog.NewTextHandler(logFile, nil))
	} else {
		// Discard logs silently when log file isn't available
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

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
