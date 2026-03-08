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
	"github.com/jhin1m/juiscript/internal/firewall"
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

// Managers holds all backend managers shared between TUI and CLI commands.
type Managers struct {
	Cfg     *config.Config
	Logger  *slog.Logger
	Site    *site.Manager
	DB      *database.Manager
	SSL     *ssl.Manager
	Backup  *backup.Manager
	Super   *supervisor.Manager
	Service *service.Manager
	PHP     *php.Manager
	Nginx   *nginx.Manager
	Prov     *provisioner.Provisioner
	Firewall *firewall.Manager
}

// initManagers creates logger, loads config, and constructs all backend managers.
func initManagers() (*Managers, error) {
	// Write logs to file instead of terminal to avoid breaking TUI display.
	logFile, err := os.OpenFile("/var/log/juiscript.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile = nil
	}

	var logger *slog.Logger
	if logFile != nil {
		// Note: logFile is not closed here — it stays open for the process lifetime.
		logger = slog.New(slog.NewTextHandler(logFile, nil))
	} else {
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
	tplEngine, err := template.New()
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}
	userMgr := system.NewUserManager(exec)

	phpMgr := php.NewManager(exec, fileMgr, tplEngine)
	svcMgr := service.NewManager(exec)
	prov := provisioner.NewProvisioner(exec, phpMgr)
	nginxMgr := nginx.NewManager(exec, fileMgr, tplEngine, cfg.Nginx.SitesAvailable, cfg.Nginx.SitesEnabled)
	dbMgr := database.NewManager(exec)
	siteMgr := site.NewManager(cfg, exec, fileMgr, userMgr, tplEngine)
	sslMgr := ssl.NewManager(exec, nginxMgr, fileMgr)
	supervisorMgr := supervisor.NewManager(exec, fileMgr, tplEngine)
	backupMgr := backup.NewManager(cfg, exec, fileMgr, dbMgr)
	firewallMgr := firewall.NewManager(exec)

	return &Managers{
		Cfg:     cfg,
		Logger:  logger,
		Site:    siteMgr,
		DB:      dbMgr,
		SSL:     sslMgr,
		Backup:  backupMgr,
		Super:   supervisorMgr,
		Service: svcMgr,
		PHP:     phpMgr,
		Nginx:   nginxMgr,
		Prov:     prov,
		Firewall: firewallMgr,
	}, nil
}

func main() {
	// Init managers once — shared between TUI and all CLI subcommands.
	mgrs, err := initManagers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:   "juiscript",
		Short: "LEMP server management CLI & TUI",
		Long:  "juiscript - Manage Nginx, PHP-FPM, MariaDB, Redis on Ubuntu",
		// Default action: launch TUI when no subcommand given
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(mgrs)
		},
	}

	// Root check: require root for all commands except "version"
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.CommandPath() == "juiscript version" {
			return nil
		}
		if os.Geteuid() != 0 {
			return fmt.Errorf("juiscript requires root privileges, run with sudo")
		}
		return nil
	}

	// Register all command groups
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(siteCmd(mgrs))
	rootCmd.AddCommand(dbCmd(mgrs))
	rootCmd.AddCommand(sslCmd(mgrs))
	rootCmd.AddCommand(serviceCmd(mgrs))
	rootCmd.AddCommand(phpCmd(mgrs))
	rootCmd.AddCommand(backupCmd(mgrs))
	rootCmd.AddCommand(queueCmd(mgrs))
	rootCmd.AddCommand(firewallCmd(mgrs))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runTUI launches the Bubble Tea TUI application.
func runTUI(mgrs *Managers) error {
	app := tui.NewApp(mgrs.Cfg, tui.AppDeps{
		SvcMgr:    mgrs.Service,
		Prov:      mgrs.Prov,
		PHPMgr:    mgrs.PHP,
		SiteMgr:   mgrs.Site,
		NginxMgr:  mgrs.Nginx,
		DBMgr:     mgrs.DB,
		SSLMgr:    mgrs.SSL,
		SuperMgr:  mgrs.Super,
		BackupMgr:   mgrs.Backup,
		FirewallMgr: mgrs.Firewall,
	})

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
