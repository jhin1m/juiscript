package tui

import (
	"context"
	"fmt"

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
// Action indicates what triggered this refresh ("install", "remove", or "" for initial load).
type PHPVersionsMsg struct {
	Versions []php.VersionInfo
	Action   string // "install", "remove", or "" (initial load)
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

// AppDeps groups all backend managers for injection into App.
// All fields can be nil for graceful degradation.
type AppDeps struct {
	SvcMgr    *service.Manager
	Prov      *provisioner.Provisioner
	PHPMgr    *php.Manager
	SiteMgr   *site.Manager
	NginxMgr  *nginx.Manager
	DBMgr     *database.Manager
	SSLMgr    *ssl.Manager
	SuperMgr  *supervisor.Manager
	BackupMgr *backup.Manager
}

// App is the root Bubble Tea model.
// It acts as a screen router, delegating to child models.
type App struct {
	theme      *theme.Theme
	header     *components.Header
	statusBar  *components.StatusBar
	serviceBar *components.ServiceStatusBar
	cfg        *config.Config
	svcMgr        *service.Manager
	prov          *provisioner.Provisioner
	phpMgr        *php.Manager
	siteMgr       *site.Manager
	nginxMgr      *nginx.Manager
	dbMgr         *database.Manager
	sslMgr        *ssl.Manager
	supervisorMgr *supervisor.Manager
	backupMgr     *backup.Manager
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
	toast            *components.ToastModel
	setupProgressCh  chan provisioner.ProgressEvent // nil when not installing
	installSummary   *provisioner.InstallSummary   // captured from goroutine for done screen
	width            int
	height           int
}

// NewApp creates the root TUI application.
// cfg can be nil (uses defaults). All managers in deps can be nil — graceful degradation.
func NewApp(cfg *config.Config, deps AppDeps) *App {
	t := theme.New()
	if cfg == nil {
		cfg = config.Default()
	}

	return &App{
		theme:      t,
		header:     components.NewHeader(t),
		statusBar:  components.NewStatusBar(t),
		serviceBar: components.NewServiceStatusBar(t),
		cfg:           cfg,
		svcMgr:        deps.SvcMgr,
		prov:          deps.Prov,
		phpMgr:        deps.PHPMgr,
		siteMgr:       deps.SiteMgr,
		nginxMgr:      deps.NginxMgr,
		dbMgr:         deps.DBMgr,
		sslMgr:        deps.SSLMgr,
		supervisorMgr: deps.SuperMgr,
		backupMgr:     deps.BackupMgr,
		toast:       components.NewToast(t),
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

	case components.ToastDismissMsg:
		a.toast.Update(msg)
		return a, nil

	case ServiceStatusMsg:
		a.serviceBar.SetServices(msg.Services)
		a.servicesScreen.SetServices(msg.Services)
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
		a.phpScreen.StopSpinner()
		a.phpScreen.SetVersions(msg.Versions)
		// Show toast only after install/remove actions, not initial load
		if msg.Action != "" {
			toastCmd := a.toast.Show(components.ToastSuccess, "PHP "+msg.Action+" completed successfully")
			return a, toastCmd
		}
		return a, nil

	case PHPVersionsErrMsg:
		a.phpScreen.StopSpinner()
		a.phpScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd

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
		// Fetch screen-specific data on navigation
		cmds := []tea.Cmd{a.fetchServiceStatus()}
		switch a.current {
		case ScreenSites:
			cmds = append(cmds, a.fetchSites())
		case ScreenNginx:
			cmds = append(cmds, a.fetchVhosts())
		case ScreenPHP:
			cmds = append(cmds, a.fetchPHPVersions())
		case ScreenDatabase:
			cmds = append(cmds, a.fetchDatabases())
		case ScreenSSL:
			cmds = append(cmds, a.fetchCerts())
		case ScreenQueues:
			cmds = append(cmds, a.fetchWorkers())
		case ScreenBackup:
			cmds = append(cmds, a.fetchBackups(""))
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
		return a, a.fetchSiteDetail(msg.Domain)

	case screens.CreateSiteMsg:
		return a, a.handleCreateSite(msg.Options)

	case screens.ToggleSiteMsg:
		return a, a.handleToggleSite(msg.Domain)

	case screens.DeleteSiteMsg:
		return a, a.handleDeleteSite(msg.Domain)

	// Nginx screen messages
	case screens.ToggleVhostMsg:
		return a, a.handleToggleVhost(msg.Domain, msg.CurrentlyEnabled)

	case screens.DeleteVhostMsg:
		return a, a.handleDeleteVhost(msg.Domain)

	case screens.TestNginxMsg:
		return a, a.handleTestNginx()

	// PHP screen messages — install/remove run in background, user can navigate freely
	case screens.InstallPHPMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Installing PHP "+msg.Version+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleInstallPHP(msg.Version))

	case screens.RemovePHPMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Removing PHP "+msg.Version+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleRemovePHP(msg.Version))

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
		return a, a.handleCreateDB(msg.Name)

	case screens.DropDBMsg:
		return a, a.handleDropDB(msg.Name)

	case screens.ImportDBMsg:
		return a, a.handleImportDB(msg.Name, msg.Path)

	case screens.ExportDBMsg:
		return a, a.handleExportDB(msg.Name)

	// SSL screen messages — operations run in background
	case screens.ObtainCertMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Obtaining certificate for "+msg.Domain+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleObtainCert(msg.Domain, msg.Email))

	case screens.RevokeCertMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Revoking certificate for "+msg.Domain+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleRevokeCert(msg.Domain))

	case screens.RenewCertMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Renewing certificate for "+msg.Domain+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleRenewCert(msg.Domain))

	// Service screen messages
	case screens.StartServiceMsg:
		return a, a.handleServiceAction(msg.Name, "start")

	case screens.StopServiceMsg:
		return a, a.handleServiceAction(msg.Name, "stop")

	case screens.RestartServiceMsg:
		return a, a.handleServiceAction(msg.Name, "restart")

	case screens.ReloadServiceMsg:
		return a, a.handleServiceAction(msg.Name, "reload")

	// Queue worker screen messages
	case screens.StartWorkerMsg:
		return a, a.handleWorkerAction(msg.Name, "start")

	case screens.StopWorkerMsg:
		return a, a.handleWorkerAction(msg.Name, "stop")

	case screens.RestartWorkerMsg:
		return a, a.handleWorkerAction(msg.Name, "restart")

	case screens.DeleteWorkerMsg:
		return a, a.handleDeleteWorker(msg.Name)

	// Backup screen messages — operations run in background
	case screens.CreateBackupMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Creating backup for "+msg.Domain+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleCreateBackup(msg.Domain, msg.Type))

	case screens.RestoreBackupMsg:
		toastCmd := a.toast.Show(components.ToastWarning, "Restoring backup for "+msg.Domain+"... (running in background)")
		return a, tea.Batch(toastCmd, a.handleRestoreBackup(msg.Path, msg.Domain))

	case screens.DeleteBackupMsg:
		return a, a.handleDeleteBackup(msg.Path)

	// --- Result messages from async operations ---

	// Site results
	case SiteListMsg:
		a.siteList.SetSites(msg.Sites)
		return a, nil
	case SiteListErrMsg:
		a.siteList.SetError(msg.Err)
		return a, nil
	case SiteDetailMsg:
		a.siteDetail.SetSite(msg.Site)
		return a, nil
	case SiteCreatedMsg:
		a.current = ScreenSites
		toastCmd := a.toast.Show(components.ToastSuccess, "Site created successfully")
		return a, tea.Batch(toastCmd, a.fetchSites())
	case SiteOpDoneMsg:
		toastCmd := a.toast.Show(components.ToastSuccess, "Site operation completed")
		return a, tea.Batch(toastCmd, a.fetchSites())
	case SiteOpErrMsg:
		if a.current == ScreenSiteDetail {
			a.current = ScreenSites
		}
		a.siteList.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd

	// Nginx results
	case VhostListMsg:
		a.nginxScreen.SetVhosts(msg.Vhosts)
		return a, nil
	case VhostListErrMsg:
		a.nginxScreen.SetError(msg.Err)
		return a, nil
	case NginxOpDoneMsg:
		toastCmd := a.toast.Show(components.ToastSuccess, "Nginx operation completed")
		return a, tea.Batch(toastCmd, a.fetchVhosts())
	case NginxOpErrMsg:
		a.nginxScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd
	case NginxTestOkMsg:
		toastCmd := a.toast.Show(components.ToastSuccess, "Nginx config test passed")
		return a, tea.Batch(toastCmd, a.fetchVhosts())

	// Database results
	case DBListMsg:
		a.databaseScreen.SetDatabases(msg.Databases)
		return a, nil
	case DBListErrMsg:
		a.databaseScreen.SetError(msg.Err)
		return a, nil
	case DBOpDoneMsg:
		toastCmd := a.toast.Show(components.ToastSuccess, "Database operation completed")
		return a, tea.Batch(toastCmd, a.fetchDatabases())
	case DBOpErrMsg:
		a.databaseScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd

	// SSL results
	case CertListMsg:
		a.sslScreen.StopSpinner()
		a.sslScreen.SetCerts(msg.Certs)
		return a, nil
	case CertListErrMsg:
		a.sslScreen.StopSpinner()
		a.sslScreen.SetError(msg.Err)
		return a, nil
	case SSLOpDoneMsg:
		a.sslScreen.StopSpinner()
		toastCmd := a.toast.Show(components.ToastSuccess, "SSL operation completed")
		return a, tea.Batch(toastCmd, a.fetchCerts())
	case SSLOpErrMsg:
		a.sslScreen.StopSpinner()
		a.sslScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd

	// Service results (ServiceStatusMsg already handled above)
	case ServiceOpErrMsg:
		a.servicesScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd

	// Queue results
	case WorkerListMsg:
		a.queuesScreen.SetWorkers(msg.Workers)
		return a, nil
	case WorkerListErrMsg:
		a.queuesScreen.SetError(msg.Err)
		return a, nil
	case QueueOpDoneMsg:
		toastCmd := a.toast.Show(components.ToastSuccess, "Queue operation completed")
		return a, tea.Batch(toastCmd, a.fetchWorkers())
	case QueueOpErrMsg:
		a.queuesScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd

	// Backup results
	case BackupListMsg:
		a.backupScreen.StopSpinner()
		a.backupScreen.SetBackups(msg.Backups)
		return a, nil
	case BackupListErrMsg:
		a.backupScreen.StopSpinner()
		a.backupScreen.SetError(msg.Err)
		return a, nil
	case BackupOpDoneMsg:
		a.backupScreen.StopSpinner()
		toastCmd := a.toast.Show(components.ToastSuccess, "Backup operation completed")
		return a, tea.Batch(toastCmd, a.fetchBackups(""))
	case BackupOpErrMsg:
		a.backupScreen.StopSpinner()
		a.backupScreen.SetError(msg.Err)
		toastCmd := a.toast.Show(components.ToastError, msg.Err.Error())
		return a, toastCmd
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

	// Toast notification between content and status bar
	toastLine := a.toast.View()
	if toastLine != "" {
		toastLine = "\n" + toastLine
	}

	return fmt.Sprintf("%s\n%s\n\n%s%s\n\n%s", header, svcBar, content, toastLine, statusBar)
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
