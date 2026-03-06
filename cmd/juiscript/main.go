package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/backup"
	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/database"
	"github.com/jhin1m/juiscript/internal/nginx"
	"github.com/jhin1m/juiscript/internal/php"
	"github.com/jhin1m/juiscript/internal/provisioner"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/ssl"
	"github.com/jhin1m/juiscript/internal/supervisor"
	"github.com/jhin1m/juiscript/internal/system"
	"github.com/jhin1m/juiscript/internal/template"
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

	// Load config — falls back to sensible defaults if file doesn't exist
	cfg, err := config.Load(config.ConfigPath())
	if err != nil {
		logger.Warn("failed to load config, using defaults", "error", err)
		cfg = config.Default()
	}

	exec := system.NewExecutor(logger)
	fileMgr := system.NewFileManager()
	tplEngine, _ := template.New() // template engine for PHP-FPM pool configs
	userMgr := system.NewUserManager(exec)

	// Core managers (already existed)
	phpMgr := php.NewManager(exec, fileMgr, tplEngine)
	svcMgr := service.NewManager(exec)
	prov := provisioner.NewProvisioner(exec, phpMgr)

	// Domain managers (newly wired)
	nginxMgr := nginx.NewManager(exec, fileMgr, tplEngine, cfg.Nginx.SitesAvailable, cfg.Nginx.SitesEnabled)
	dbMgr := database.NewManager(exec)
	siteMgr := site.NewManager(cfg, exec, fileMgr, userMgr, tplEngine)
	sslMgr := ssl.NewManager(exec, nginxMgr, fileMgr)
	supervisorMgr := supervisor.NewManager(exec, fileMgr, tplEngine)
	backupMgr := backup.NewManager(cfg, exec, fileMgr, dbMgr)

	app := tui.NewApp(cfg, tui.AppDeps{
		SvcMgr:    svcMgr,
		Prov:      prov,
		PHPMgr:    phpMgr,
		SiteMgr:   siteMgr,
		NginxMgr:  nginxMgr,
		DBMgr:     dbMgr,
		SSLMgr:    sslMgr,
		SuperMgr:  supervisorMgr,
		BackupMgr: backupMgr,
	})

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
