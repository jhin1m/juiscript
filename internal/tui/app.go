package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/screens"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

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
}

// App is the root Bubble Tea model.
// It acts as a screen router, delegating to child models.
type App struct {
	theme       *theme.Theme
	header      *components.Header
	statusBar   *components.StatusBar
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
	width            int
	height           int
}

// NewApp creates the root TUI application.
func NewApp() *App {
	t := theme.New()

	return &App{
		theme:       t,
		header:      components.NewHeader(t),
		statusBar:   components.NewStatusBar(t),
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
	}
}

func (a *App) Init() tea.Cmd {
	return a.dashboard.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.header.SetWidth(msg.Width)
		a.statusBar.SetWidth(msg.Width)
		return a, a.updateActiveScreen(msg)

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
		return a, nil

	case screens.GoBackMsg:
		return a.goBack(), nil

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

	// Service screen messages
	case screens.StartServiceMsg:
		// TODO: Call service manager to start
		return a, nil

	case screens.StopServiceMsg:
		// TODO: Call service manager to stop
		return a, nil

	case screens.RestartServiceMsg:
		// TODO: Call service manager to restart
		return a, nil

	case screens.ReloadServiceMsg:
		// TODO: Call service manager to reload
		return a, nil

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
	case ScreenServices:
		updated, cmd := a.servicesScreen.Update(msg)
		a.servicesScreen = updated.(*screens.ServicesScreen)
		return cmd
	case ScreenQueues:
		updated, cmd := a.queuesScreen.Update(msg)
		a.queuesScreen = updated.(*screens.QueuesScreen)
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
	case ScreenServices:
		content = a.servicesScreen.View()
	case ScreenQueues:
		content = a.queuesScreen.View()
	default:
		content = a.theme.Subtitle.Render(
			fmt.Sprintf("\n  [%s] screen - Coming soon...\n\n  Press 'esc' to go back",
				a.screenTitle()))
	}

	bindings := a.currentBindings()
	statusBar := a.statusBar.View(bindings)

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, statusBar)
}

func (a *App) screenTitle() string {
	switch a.current {
	case ScreenSiteCreate:
		return "Create Site"
	case ScreenSiteDetail:
		return a.siteDetail.ScreenTitle()
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
			{Key: "1-8", Desc: "jump to"},
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
	default:
		return append([]components.KeyBinding{
			{Key: "esc", Desc: "back"},
		}, base...)
	}
}
