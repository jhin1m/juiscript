package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/config"
	"github.com/jhin1m/juiscript/internal/php"
	"github.com/jhin1m/juiscript/internal/provisioner"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/screens"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// ServiceStatusMsg delivers fresh service statuses to the App.
type ServiceStatusMsg struct {
	Services []service.Status
}

// ServiceStatusErrMsg reports a failure to read service status.
type ServiceStatusErrMsg struct {
	Err error
}

// DetectPackagesMsg carries the result of package detection on startup.
type DetectPackagesMsg struct {
	Packages []provisioner.PackageInfo
}

// DetectPackagesErrMsg reports a failure to detect packages.
type DetectPackagesErrMsg struct {
	Err error
}

// PHPVersionsMsg delivers installed PHP versions to the PHP screen.
type PHPVersionsMsg struct {
	Versions []php.VersionInfo
}

// PHPVersionsErrMsg reports a failure to list PHP versions.
type PHPVersionsErrMsg struct {
	Err error
}

// Screen identifies which screen is currently active.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenSites
	ScreenSiteCreate
	ScreenSiteDetail
	ScreenNginx
	ScreenPHP
	ScreenDatabase
	ScreenSSL
	ScreenServices
	ScreenQueues
	ScreenBackup
	ScreenSetup
)

// screenNames maps screen titles to Screen enum.
var screenNames = map[string]Screen{
	"Sites":    ScreenSites,
	"Nginx":    ScreenNginx,
	"PHP":      ScreenPHP,
	"Database": ScreenDatabase,
	"SSL":      ScreenSSL,
	"Services": ScreenServices,
	"Queues":   ScreenQueues,
	"Backup":   ScreenBackup,
	"Setup":    ScreenSetup,
}

// App is the root Bubble Tea model.
// It acts as a screen router, delegating to child models.
type App struct {
	theme      *theme.Theme
	header     *components.Header
	statusBar  *components.StatusBar
	serviceBar *components.ServiceStatusBar
	cfg        *config.Config
	svcMgr     *service.Manager
	prov       *provisioner.Provisioner
	phpMgr     *php.Manager
	current     Screen
	previous    Screen // for back navigation from sub-screens
	dashboard   *screens.Dashboard
	siteList    *screens.SiteList
	siteCreate  *screens.SiteCreate
	siteDetail  *screens.SiteDetail
	nginxScreen *screens.NginxScreen
	phpScreen        *screens.PHPScreen
	databaseScreen   *screens.DatabaseScreen
	servicesScreen   *screens.ServicesScreen
	queuesScreen     *screens.QueuesScreen
	sslScreen        *screens.SSLScreen
	backupScreen     *screens.BackupScreen
	setupScreen      *screens.SetupScreen
	setupProgressCh  chan provisioner.ProgressEvent // nil when not installing
	installSummary   *provisioner.InstallSummary   // captured from goroutine for done screen
	width            int
	height           int
}

// NewApp creates the root TUI application.
// cfg, svcMgr, prov and phpMgr can be nil — graceful degradation.
func NewApp(cfg *config.Config, svcMgr *service.Manager, prov *provisioner.Provisioner, phpMgr *php.Manager) *App {
	t := theme.New()
	if cfg == nil {
		cfg = config.Default()
	}

	return &App{
		theme:      t,
		header:     components.NewHeader(t),
		statusBar:  components.NewStatusBar(t),
		serviceBar: components.NewServiceStatusBar(t),
		cfg:        cfg,
		svcMgr:     svcMgr,
		prov:       prov,
		phpMgr:     phpMgr,
		current:     ScreenDashboard,
		dashboard:   screens.NewDashboard(t),
		siteList:    screens.NewSiteList(t),
		siteCreate:  screens.NewSiteCreate(t),
		siteDetail:  screens.NewSiteDetail(t),
		nginxScreen: screens.NewNginxScreen(t),
		phpScreen:      screens.NewPHPScreen(t),
		databaseScreen: screens.NewDatabaseScreen(t),
		servicesScreen: screens.NewServicesScreen(t),
		queuesScreen:   screens.NewQueuesScreen(t),
		sslScreen:      screens.NewSSLScreen(t),
		backupScreen:   screens.NewBackupScreen(t),
		setupScreen:    screens.NewSetupScreen(t),
	}
}

func (a *App) Init() tea.Cmd {
	// Pass default PHP version from config to PHP screen
	a.phpScreen.SetDefaultVersion(a.cfg.PHP.DefaultVersion)
	return tea.Batch(a.dashboard.Init(), a.fetchServiceStatus(), a.detectPackages())
}

// fetchServiceStatus returns a tea.Cmd that fetches service statuses asynchronously.
func (a *App) fetchServiceStatus() tea.Cmd {
	if a.svcMgr == nil {
		return func() tea.Msg {
			return ServiceStatusErrMsg{Err: fmt.Errorf("service manager not available")}
		}
	}
	return func() tea.Msg {
		statuses, err := a.svcMgr.ListAll(context.Background())
		if err != nil {
			return ServiceStatusErrMsg{Err: err}
		}
		return ServiceStatusMsg{Services: statuses}
	}
}

// fetchPHPVersions returns a tea.Cmd that loads installed PHP versions asynchronously.
func (a *App) fetchPHPVersions() tea.Cmd {
	if a.phpMgr == nil {
		return nil
	}
	return func() tea.Msg {
		versions, err := a.phpMgr.ListVersions(context.Background())
		if err != nil {
			return PHPVersionsErrMsg{Err: err}
		}
		return PHPVersionsMsg{Versions: versions}
	}
}

// detectPackages runs package detection asynchronously on startup.
func (a *App) detectPackages() tea.Cmd {
	if a.prov == nil {
		return nil
	}
	return func() tea.Msg {
		pkgs, err := a.prov.DetectAll(context.Background())
		if err != nil {
			return DetectPackagesErrMsg{Err: err}
		}
		return DetectPackagesMsg{Packages: pkgs}
	}
}

// waitForProgress reads the next progress event from the channel.
// Returns SetupDoneMsg when channel is closed (install complete).
func waitForProgress(ch <-chan provisioner.ProgressEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return screens.SetupDoneMsg{}
		}
		return screens.SetupProgressMsg{Event: ev}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.header.SetWidth(msg.Width)
		a.statusBar.SetWidth(msg.Width)
		a.serviceBar.SetWidth(msg.Width)
		return a, a.updateActiveScreen(msg)

	case ServiceStatusMsg:
		a.serviceBar.SetServices(msg.Services)
		return a, nil

	case ServiceStatusErrMsg:
		a.serviceBar.SetError(msg.Err.Error())
		return a, nil

	case DetectPackagesMsg:
		// Count missing: static packages individually, PHP only if zero versions installed
		missing := 0
		phpHasAny := false
		for _, pkg := range msg.Packages {
			if pkg.Name == "php" {
				if pkg.Installed {
					phpHasAny = true
				}
			} else if !pkg.Installed {
				missing++
			}
		}
		if !phpHasAny {
			missing++
		}
		a.dashboard.SetMissingCount(missing)
		a.setupScreen.SetPackages(msg.Packages)
		return a, nil

	case DetectPackagesErrMsg:
		// Silently ignore — setup screen remains empty
		return a, nil

	case PHPVersionsMsg:
		a.phpScreen.SetVersions(msg.Versions)
		return a, nil

	case PHPVersionsErrMsg:
		a.phpScreen.SetError(msg.Err)
		return a, nil

	case tea.KeyMsg:
		// Global quit: ctrl+c always quits
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		// 'q' only quits from dashboard
		if msg.String() == "q" && a.current == ScreenDashboard {
			return a, tea.Quit
		}

	// Navigation messages from child screens
	case screens.NavigateMsg:
		if screen, ok := screenNames[msg.Screen]; ok {
			a.previous = a.current
			a.current = screen
		}
		// Fetch PHP versions when navigating to PHP screen
		cmds := []tea.Cmd{a.fetchServiceStatus()}
		if a.current == ScreenPHP {
			cmds = append(cmds, a.fetchPHPVersions())
		}
		return a, tea.Batch(cmds...)

	case screens.GoBackMsg:
		return a.goBack(), a.fetchServiceStatus()

	// --- Setup/Provisioner messages ---

	case screens.RunSetupMsg:
		// Start install: create channel, spawn goroutine
		ch := make(chan provisioner.ProgressEvent, 10)
		a.setupProgressCh = ch
		go func() {
			summary, _ := a.prov.InstallSelected(context.Background(), msg.Names, func(ev provisioner.ProgressEvent) {
				ch <- ev
			})
			// Store summary so SetupDoneMsg can carry it to the done screen
			a.installSummary = summary
			close(ch)
		}()
		return a, waitForProgress(ch)

	case screens.SetupProgressMsg:
		// Forward to setup screen, then continue listening
		a.setupScreen.Update(msg)
		return a, waitForProgress(a.setupProgressCh)

	case screens.SetupDoneMsg:
		// Attach captured summary so done screen shows correct counts
		msg.Summary = a.installSummary
		a.installSummary = nil
		a.setupScreen.Update(msg)
		a.setupProgressCh = nil
		return a, tea.Batch(a.detectPackages(), a.fetchServiceStatus(), a.fetchPHPVersions())

	// Site-specific navigation messages
	case screens.ShowCreateFormMsg:
		a.previous = a.current
		a.current = ScreenSiteCreate
		a.siteCreate.Reset()
		return a, nil

	case screens.ShowSiteDetailMsg:
		a.previous = a.current
		a.current = ScreenSiteDetail
		// TODO: Load site detail from manager
		return a, nil

	case screens.CreateSiteMsg:
		// TODO: Call site manager to create site
		// For now, go back to site list
		a.current = ScreenSites
		return a, nil

	case screens.ToggleSiteMsg:
		// TODO: Call site manager to toggle
		return a, nil

	case screens.DeleteSiteMsg:
		// TODO: Call site manager to delete
		return a, nil

	// Nginx screen messages
	case screens.ToggleVhostMsg:
		// TODO: Call nginx manager to toggle
		return a, nil

	case screens.DeleteVhostMsg:
		// TODO: Call nginx manager to delete
		return a, nil

	case screens.TestNginxMsg:
		// TODO: Call nginx manager to test config
		return a, nil

	// PHP screen messages
	case screens.InstallPHPMsg:
		// TODO: Call PHP manager to install version
		return a, nil

	case screens.RemovePHPMsg:
		// TODO: Call PHP manager to remove version
		return a, nil

	case screens.SetDefaultPHPMsg:
		// Update config and save, then refresh PHP screen
		a.cfg.PHP.DefaultVersion = msg.Version
		a.phpScreen.SetDefaultVersion(msg.Version)
		go func() {
			_ = a.cfg.Save(config.ConfigPath())
		}()
		return a, nil

	// Database screen messages
	case screens.CreateDBMsg:
		// TODO: Call database manager to create DB
		return a, nil

	case screens.DropDBMsg:
		// TODO: Call database manager to drop DB
		return a, nil

	case screens.ImportDBMsg:
		// TODO: Call database manager to import
		return a, nil

	case screens.ExportDBMsg:
		// TODO: Call database manager to export
		return a, nil

	// SSL screen messages
	case screens.ObtainCertMsg:
		// TODO: Call SSL manager to obtain cert (needs domain, webroot, email input)
		return a, nil

	case screens.RevokeCertMsg:
		// TODO: Call SSL manager to revoke cert for msg.Domain
		return a, nil

	case screens.RenewCertMsg:
		// TODO: Call SSL manager to renew cert for msg.Domain
		return a, nil

	// Service screen messages — re-fetch status after each action
	case screens.StartServiceMsg:
		// TODO: Call service manager to start
		return a, a.fetchServiceStatus()

	case screens.StopServiceMsg:
		// TODO: Call service manager to stop
		return a, a.fetchServiceStatus()

	case screens.RestartServiceMsg:
		// TODO: Call service manager to restart
		return a, a.fetchServiceStatus()

	case screens.ReloadServiceMsg:
		// TODO: Call service manager to reload
		return a, a.fetchServiceStatus()

	// Queue worker screen messages
	case screens.StartWorkerMsg:
		// TODO: Call supervisor manager to start worker
		return a, nil

	case screens.StopWorkerMsg:
		// TODO: Call supervisor manager to stop worker
		return a, nil

	case screens.RestartWorkerMsg:
		// TODO: Call supervisor manager to restart worker
		return a, nil

	case screens.DeleteWorkerMsg:
		// TODO: Call supervisor manager to delete worker
		return a, nil

	// Backup screen messages
	case screens.CreateBackupMsg:
		// TODO: Call backup manager to create backup
		return a, nil

	case screens.RestoreBackupMsg:
		// TODO: Call backup manager to restore
		return a, nil

	case screens.DeleteBackupMsg:
		// TODO: Call backup manager to delete
		return a, nil
	}

	// Delegate to active screen
	return a, a.updateActiveScreen(msg)
}

func (a *App) goBack() *App {
	switch a.current {
	case ScreenSiteCreate, ScreenSiteDetail:
		a.current = ScreenSites
	default:
		a.current = ScreenDashboard
	}
	return a
}

func (a *App) updateActiveScreen(msg tea.Msg) tea.Cmd {
	switch a.current {
	case ScreenDashboard:
		updated, cmd := a.dashboard.Update(msg)
		a.dashboard = updated.(*screens.Dashboard)
		return cmd
	case ScreenSites:
		updated, cmd := a.siteList.Update(msg)
		a.siteList = updated.(*screens.SiteList)
		return cmd
	case ScreenSiteCreate:
		updated, cmd := a.siteCreate.Update(msg)
		a.siteCreate = updated.(*screens.SiteCreate)
		return cmd
	case ScreenSiteDetail:
		updated, cmd := a.siteDetail.Update(msg)
		a.siteDetail = updated.(*screens.SiteDetail)
		return cmd
	case ScreenNginx:
		updated, cmd := a.nginxScreen.Update(msg)
		a.nginxScreen = updated.(*screens.NginxScreen)
		return cmd
	case ScreenPHP:
		updated, cmd := a.phpScreen.Update(msg)
		a.phpScreen = updated.(*screens.PHPScreen)
		return cmd
	case ScreenDatabase:
		updated, cmd := a.databaseScreen.Update(msg)
		a.databaseScreen = updated.(*screens.DatabaseScreen)
		return cmd
	case ScreenSSL:
		updated, cmd := a.sslScreen.Update(msg)
		a.sslScreen = updated.(*screens.SSLScreen)
		return cmd
	case ScreenServices:
		updated, cmd := a.servicesScreen.Update(msg)
		a.servicesScreen = updated.(*screens.ServicesScreen)
		return cmd
	case ScreenQueues:
		updated, cmd := a.queuesScreen.Update(msg)
		a.queuesScreen = updated.(*screens.QueuesScreen)
		return cmd
	case ScreenBackup:
		updated, cmd := a.backupScreen.Update(msg)
		a.backupScreen = updated.(*screens.BackupScreen)
		return cmd
	case ScreenSetup:
		updated, cmd := a.setupScreen.Update(msg)
		a.setupScreen = updated.(*screens.SetupScreen)
		return cmd
	default:
		return nil
	}
}

func (a *App) View() string {
	header := a.header.View(a.screenTitle())

	var content string
	switch a.current {
	case ScreenDashboard:
		content = a.dashboard.View()
	case ScreenSites:
		content = a.siteList.View()
	case ScreenSiteCreate:
		content = a.siteCreate.View()
	case ScreenSiteDetail:
		content = a.siteDetail.View()
	case ScreenNginx:
		content = a.nginxScreen.View()
	case ScreenPHP:
		content = a.phpScreen.View()
	case ScreenDatabase:
		content = a.databaseScreen.View()
	case ScreenSSL:
		content = a.sslScreen.View()
	case ScreenServices:
		content = a.servicesScreen.View()
	case ScreenQueues:
		content = a.queuesScreen.View()
	case ScreenBackup:
		content = a.backupScreen.View()
	case ScreenSetup:
		content = a.setupScreen.View()
	default:
		content = a.theme.Subtitle.Render(
			fmt.Sprintf("\n  [%s] screen - Coming soon...\n\n  Press 'esc' to go back",
				a.screenTitle()))
	}

	bindings := a.currentBindings()
	statusBar := a.statusBar.View(bindings)

	svcBar := a.serviceBar.View()
	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s", header, svcBar, content, statusBar)
}

func (a *App) screenTitle() string {
	switch a.current {
	case ScreenSiteCreate:
		return "Create Site"
	case ScreenSiteDetail:
		return a.siteDetail.ScreenTitle()
	case ScreenSetup:
		return "Setup"
	default:
		for name, screen := range screenNames {
			if screen == a.current {
				return name
			}
		}
		return "Dashboard"
	}
}

func (a *App) currentBindings() []components.KeyBinding {
	base := []components.KeyBinding{
		{Key: "ctrl+c", Desc: "quit"},
	}

	switch a.current {
	case ScreenDashboard:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "enter", Desc: "select"},
			{Key: "1-9", Desc: "jump to"},
			{Key: "q", Desc: "quit"},
		}, base...)
	case ScreenSiteCreate:
		return append([]components.KeyBinding{
			{Key: "enter", Desc: "next"},
			{Key: "tab", Desc: "cycle"},
			{Key: "esc", Desc: "cancel"},
		}, base...)
	case ScreenNginx:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "e", Desc: "enable/disable"},
			{Key: "d", Desc: "delete"},
			{Key: "t", Desc: "test config"},
			{Key: "esc", Desc: "back"},
		}, base...)
	case ScreenPHP:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "d", Desc: "set default"},
			{Key: "i", Desc: "install"},
			{Key: "r", Desc: "remove"},
			{Key: "esc", Desc: "back"},
		}, base...)
	case ScreenDatabase:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "c", Desc: "create"},
			{Key: "d", Desc: "drop"},
			{Key: "i", Desc: "import"},
			{Key: "e", Desc: "export"},
			{Key: "esc", Desc: "back"},
		}, base...)
	case ScreenServices:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "s", Desc: "start"},
			{Key: "x", Desc: "stop"},
			{Key: "r", Desc: "restart"},
			{Key: "l", Desc: "reload"},
			{Key: "esc", Desc: "back"},
		}, base...)
	case ScreenQueues:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "s", Desc: "start"},
			{Key: "x", Desc: "stop"},
			{Key: "r", Desc: "restart"},
			{Key: "d", Desc: "delete"},
			{Key: "esc", Desc: "back"},
		}, base...)
	case ScreenBackup:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "c", Desc: "create"},
			{Key: "r", Desc: "restore"},
			{Key: "d", Desc: "delete"},
			{Key: "esc", Desc: "back"},
		}, base...)
	case ScreenSetup:
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "space", Desc: "toggle"},
			{Key: "enter", Desc: "confirm"},
			{Key: "esc", Desc: "back"},
		}, base...)
	default:
		return append([]components.KeyBinding{
			{Key: "esc", Desc: "back"},
		}, base...)
	}
}
