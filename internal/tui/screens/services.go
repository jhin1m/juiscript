package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/service"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// ServicesScreen displays all managed services with start/stop/restart actions.
type ServicesScreen struct {
	theme    *theme.Theme
	services []service.Status
	cursor   int
	width    int
	height   int
	err      error
}

// NewServicesScreen creates the service management screen.
func NewServicesScreen(t *theme.Theme) *ServicesScreen {
	return &ServicesScreen{theme: t}
}

// SetServices updates the service list.
func (s *ServicesScreen) SetServices(services []service.Status) {
	s.services = services
	s.err = nil
}

// SetError sets an error to display.
func (s *ServicesScreen) SetError(err error) {
	s.err = err
}

func (s *ServicesScreen) Init() tea.Cmd { return nil }

func (s *ServicesScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.services)-1 {
				s.cursor++
			}
		case "s":
			if len(s.services) > 0 {
				return s, s.serviceCmd(StartServiceMsg{
					Name: s.services[s.cursor].Name,
				})
			}
		case "x":
			if len(s.services) > 0 {
				return s, s.serviceCmd(StopServiceMsg{
					Name: s.services[s.cursor].Name,
				})
			}
		case "r":
			if len(s.services) > 0 {
				return s, s.serviceCmd(RestartServiceMsg{
					Name: s.services[s.cursor].Name,
				})
			}
		case "l":
			if len(s.services) > 0 {
				return s, s.serviceCmd(ReloadServiceMsg{
					Name: s.services[s.cursor].Name,
				})
			}
		case "esc", "q":
			return s, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return s, nil
}

// serviceCmd wraps a service message into a tea.Cmd.
func (s *ServicesScreen) serviceCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (s *ServicesScreen) View() string {
	title := s.theme.Title.Render("Services")

	if s.err != nil {
		errMsg := s.theme.ErrorText.Render(fmt.Sprintf("Error: %v", s.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(s.services) == 0 {
		empty := s.theme.Subtitle.Render("  No services detected.")
		help := s.theme.HelpDesc.Render("  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	// Table header
	header := fmt.Sprintf("  %-20s %-10s %-10s %-8s %-10s", "SERVICE", "STATE", "SUBSTATE", "PID", "MEMORY")
	headerStyle := s.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for i, svc := range s.services {
		cursor := "  "
		style := s.theme.Inactive
		if i == s.cursor {
			cursor = "> "
			style = s.theme.Active
		}

		// State color coding
		stateStr, stateStyle := s.stateDisplay(svc.State)

		// PID display
		pidStr := "-"
		if svc.PID > 0 {
			pidStr = fmt.Sprintf("%d", svc.PID)
		}

		// Memory display
		memStr := "-"
		if svc.MemoryMB > 0 {
			memStr = fmt.Sprintf("%.1fMB", svc.MemoryMB)
		}

		row := fmt.Sprintf("%s%-20s %s  %-10s %-8s %-10s",
			cursor,
			style.Render(string(svc.Name)),
			stateStyle.Render(fmt.Sprintf("%-10s", stateStr)),
			s.theme.Subtitle.Render(svc.SubState),
			s.theme.Subtitle.Render(pidStr),
			s.theme.Subtitle.Render(memStr),
		)
		rows += row + "\n"
	}

	help := s.theme.HelpDesc.Render("  s:start  x:stop  r:restart  l:reload  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

// stateDisplay returns a display string and style for a service state.
func (s *ServicesScreen) stateDisplay(state string) (string, lipgloss.Style) {
	switch state {
	case "active":
		return "active", s.theme.OkText
	case "failed":
		return "failed", s.theme.ErrorText
	default:
		return state, s.theme.Subtitle
	}
}

func (s *ServicesScreen) ScreenTitle() string { return "Services" }

// Messages for service screen actions
type StartServiceMsg struct {
	Name service.ServiceName
}

type StopServiceMsg struct {
	Name service.ServiceName
}

type RestartServiceMsg struct {
	Name service.ServiceName
}

type ReloadServiceMsg struct {
	Name service.ServiceName
}
