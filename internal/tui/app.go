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
	theme      *theme.Theme
	header     *components.Header
	statusBar  *components.StatusBar
	current    Screen
	previous   Screen // for back navigation from sub-screens
	dashboard  *screens.Dashboard
	siteList   *screens.SiteList
	siteCreate *screens.SiteCreate
	siteDetail *screens.SiteDetail
	width      int
	height     int
}

// NewApp creates the root TUI application.
func NewApp() *App {
	t := theme.New()

	return &App{
		theme:      t,
		header:     components.NewHeader(t),
		statusBar:  components.NewStatusBar(t),
		current:    ScreenDashboard,
		dashboard:  screens.NewDashboard(t),
		siteList:   screens.NewSiteList(t),
		siteCreate: screens.NewSiteCreate(t),
		siteDetail: screens.NewSiteDetail(t),
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
	default:
		return append([]components.KeyBinding{
			{Key: "esc", Desc: "back"},
		}, base...)
	}
}
