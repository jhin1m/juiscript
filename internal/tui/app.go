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
	ScreenNginx
	ScreenPHP
	ScreenDatabase
	ScreenSSL
	ScreenServices
	ScreenQueues
	ScreenBackup
)

// screenNames maps screen titles to Screen enum.
// Used by NavigateMsg to resolve screen names from dashboard.
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
	theme     *theme.Theme
	header    *components.Header
	statusBar *components.StatusBar
	current   Screen
	dashboard *screens.Dashboard
	width     int
	height    int
}

// NewApp creates the root TUI application.
func NewApp() *App {
	t := theme.New()

	return &App{
		theme:     t,
		header:    components.NewHeader(t),
		statusBar: components.NewStatusBar(t),
		current:   ScreenDashboard,
		dashboard: screens.NewDashboard(t),
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
		// Propagate size to active screen
		return a, a.updateActiveScreen(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if a.current == ScreenDashboard {
				return a, tea.Quit
			}
			// 'q' goes back from sub-screens
			a.current = ScreenDashboard
			return a, nil
		case "esc":
			if a.current != ScreenDashboard {
				a.current = ScreenDashboard
				return a, nil
			}
		}

	case screens.NavigateMsg:
		if screen, ok := screenNames[msg.Screen]; ok {
			a.current = screen
		}
		return a, nil

	case screens.GoBackMsg:
		a.current = ScreenDashboard
		return a, nil
	}

	// Delegate to active screen
	return a, a.updateActiveScreen(msg)
}

func (a *App) updateActiveScreen(msg tea.Msg) tea.Cmd {
	switch a.current {
	case ScreenDashboard:
		updated, cmd := a.dashboard.Update(msg)
		a.dashboard = updated.(*screens.Dashboard)
		return cmd
	default:
		// Other screens not yet implemented
		return nil
	}
}

func (a *App) View() string {
	// Header
	header := a.header.View(a.screenTitle())

	// Active screen content
	var content string
	switch a.current {
	case ScreenDashboard:
		content = a.dashboard.View()
	default:
		content = a.theme.Subtitle.Render(
			fmt.Sprintf("\n  [%s] screen - Coming soon...\n\n  Press 'q' or 'esc' to go back",
				a.screenTitle()))
	}

	// Status bar with context-appropriate keybindings
	bindings := a.currentBindings()
	statusBar := a.statusBar.View(bindings)

	// Layout: header + content + spacer + status bar
	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, statusBar)
}

func (a *App) screenTitle() string {
	for name, screen := range screenNames {
		if screen == a.current {
			return name
		}
	}
	return "Dashboard"
}

func (a *App) currentBindings() []components.KeyBinding {
	base := []components.KeyBinding{
		{Key: "q", Desc: "quit"},
	}

	if a.current == ScreenDashboard {
		return append([]components.KeyBinding{
			{Key: "j/k", Desc: "navigate"},
			{Key: "enter", Desc: "select"},
			{Key: "1-8", Desc: "jump to"},
		}, base...)
	}

	return append([]components.KeyBinding{
		{Key: "esc", Desc: "back"},
	}, base...)
}
