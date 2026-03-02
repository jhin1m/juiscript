package screens

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/supervisor"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// QueuesScreen displays Supervisor queue workers with management actions.
type QueuesScreen struct {
	theme   *theme.Theme
	workers []supervisor.WorkerStatus
	cursor  int
	width   int
	height  int
	err     error
}

// NewQueuesScreen creates the queue worker management screen.
func NewQueuesScreen(t *theme.Theme) *QueuesScreen {
	return &QueuesScreen{theme: t}
}

// SetWorkers updates the worker list.
func (q *QueuesScreen) SetWorkers(workers []supervisor.WorkerStatus) {
	q.workers = workers
	q.err = nil
}

// SetError sets an error to display.
func (q *QueuesScreen) SetError(err error) {
	q.err = err
}

func (q *QueuesScreen) Init() tea.Cmd { return nil }

func (q *QueuesScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		q.width = msg.Width
		q.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if q.cursor > 0 {
				q.cursor--
			}
		case "down", "j":
			if q.cursor < len(q.workers)-1 {
				q.cursor++
			}
		case "s":
			if len(q.workers) > 0 {
				return q, q.workerCmd(StartWorkerMsg{Name: q.workers[q.cursor].Name})
			}
		case "x":
			if len(q.workers) > 0 {
				return q, q.workerCmd(StopWorkerMsg{Name: q.workers[q.cursor].Name})
			}
		case "r":
			if len(q.workers) > 0 {
				return q, q.workerCmd(RestartWorkerMsg{Name: q.workers[q.cursor].Name})
			}
		case "d":
			if len(q.workers) > 0 {
				return q, q.workerCmd(DeleteWorkerMsg{Name: q.workers[q.cursor].Name})
			}
		case "esc", "q":
			return q, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return q, nil
}

// workerCmd wraps a worker message into a tea.Cmd.
func (q *QueuesScreen) workerCmd(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}

func (q *QueuesScreen) View() string {
	title := q.theme.Title.Render("Queue Workers")

	if q.err != nil {
		errMsg := q.theme.ErrorText.Render(fmt.Sprintf("Error: %v", q.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(q.workers) == 0 {
		empty := q.theme.Subtitle.Render("  No queue workers found.")
		help := q.theme.HelpDesc.Render("  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	// Table header
	header := fmt.Sprintf("  %-40s %-10s %-8s %-12s", "WORKER", "STATE", "PID", "UPTIME")
	headerStyle := q.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for i, w := range q.workers {
		cursor := "  "
		style := q.theme.Inactive
		if i == q.cursor {
			cursor = "> "
			style = q.theme.Active
		}

		stateStr, stateStyle := q.stateDisplay(w.State)

		pidStr := "-"
		if w.PID > 0 {
			pidStr = fmt.Sprintf("%d", w.PID)
		}

		uptimeStr := "-"
		if w.Uptime > 0 {
			uptimeStr = formatUptime(w.Uptime)
		}

		row := fmt.Sprintf("%s%-40s %s  %-8s %-12s",
			cursor,
			style.Render(w.Name),
			stateStyle.Render(fmt.Sprintf("%-10s", stateStr)),
			q.theme.Subtitle.Render(pidStr),
			q.theme.Subtitle.Render(uptimeStr),
		)
		rows += row + "\n"
	}

	help := q.theme.HelpDesc.Render("  s:start  x:stop  r:restart  d:delete  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

// stateDisplay returns a display string and style for a worker state.
func (q *QueuesScreen) stateDisplay(state string) (string, lipgloss.Style) {
	switch state {
	case "RUNNING":
		return "RUNNING", q.theme.OkText
	case "FATAL":
		return "FATAL", q.theme.ErrorText
	case "STOPPED":
		return "STOPPED", q.theme.WarnText
	default:
		return state, q.theme.Subtitle
	}
}

// formatUptime converts a duration into a human-readable "Xh Ym" string.
func formatUptime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

func (q *QueuesScreen) ScreenTitle() string { return "Queues" }

// Messages for queue screen actions
type StartWorkerMsg struct{ Name string }
type StopWorkerMsg struct{ Name string }
type RestartWorkerMsg struct{ Name string }
type DeleteWorkerMsg struct{ Name string }
