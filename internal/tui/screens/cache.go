package screens

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/cache"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// Cache screen action messages (screen -> app)

// FlushRedisDBMsg requests flushing a specific Redis database.
type FlushRedisDBMsg struct{ DB int }

// FlushRedisAllMsg requests flushing all Redis databases.
type FlushRedisAllMsg struct{}

// ResetOpcacheMsg requests PHP Opcache reset via FPM restart.
type ResetOpcacheMsg struct{ PHPVersion string }

// CacheScreen shows Redis status and cache management actions.
type CacheScreen struct {
	theme  *theme.Theme
	status *cache.CacheStatus

	// Input mode for forms
	inputMode   string // "", "flush-db", "opcache-version"
	inputBuffer string

	// Confirm mode for destructive actions
	confirmMode   bool
	confirmPrompt string
	pendingAction string

	width  int
	height int
	err    error
}

// NewCacheScreen creates a cache management screen.
func NewCacheScreen(t *theme.Theme) *CacheScreen {
	return &CacheScreen{theme: t}
}

// SetStatus updates the displayed cache status.
func (s *CacheScreen) SetStatus(status *cache.CacheStatus) {
	s.status = status
	s.err = nil
}

// SetError displays an error on the screen.
func (s *CacheScreen) SetError(err error) { s.err = err }

func (s *CacheScreen) Init() tea.Cmd { return nil }

func (s *CacheScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case tea.KeyMsg:
		// Input mode: collecting text
		if s.inputMode != "" {
			return s.updateInput(msg)
		}
		// Confirm mode: y/n
		if s.confirmMode {
			return s.updateConfirm(msg)
		}
		// Normal mode
		return s.updateNormal(msg)
	}
	return s, nil
}

func (s *CacheScreen) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "f":
		// Flush specific DB — prompt for DB number
		s.inputMode = "flush-db"
		s.inputBuffer = ""
	case "F":
		// Flush ALL — confirm first (destructive)
		s.confirmMode = true
		s.confirmPrompt = "Flush ALL Redis databases? This deletes all cached data, sessions, queues. [y/n]"
		s.pendingAction = "flush-all"
	case "o":
		// Opcache reset — prompt for PHP version
		s.inputMode = "opcache-version"
		s.inputBuffer = ""
	case "esc", "q":
		return s, func() tea.Msg { return GoBackMsg{} }
	}
	return s, nil
}

func (s *CacheScreen) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return s.submitInput()
	case "esc":
		s.inputMode = ""
		s.inputBuffer = ""
		return s, nil
	case "backspace":
		if len(s.inputBuffer) > 0 {
			s.inputBuffer = s.inputBuffer[:len(s.inputBuffer)-1]
		}
	default:
		if len(msg.String()) == 1 {
			s.inputBuffer += msg.String()
		}
	}
	return s, nil
}

func (s *CacheScreen) submitInput() (tea.Model, tea.Cmd) {
	mode := s.inputMode
	buf := s.inputBuffer
	s.inputMode = ""
	s.inputBuffer = ""

	switch mode {
	case "flush-db":
		db, err := strconv.Atoi(buf)
		if err != nil || db < 0 {
			s.err = fmt.Errorf("invalid database number: %s", buf)
			return s, nil
		}
		return s, func() tea.Msg { return FlushRedisDBMsg{DB: db} }

	case "opcache-version":
		ver := buf
		if ver == "" {
			s.err = fmt.Errorf("PHP version required (e.g. 8.3)")
			return s, nil
		}
		return s, func() tea.Msg { return ResetOpcacheMsg{PHPVersion: ver} }
	}
	return s, nil
}

func (s *CacheScreen) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		s.confirmMode = false
		return s.handleConfirm()
	case "n", "N", "esc":
		s.confirmMode = false
		s.pendingAction = ""
	}
	return s, nil
}

func (s *CacheScreen) handleConfirm() (tea.Model, tea.Cmd) {
	action := s.pendingAction
	s.pendingAction = ""

	switch action {
	case "flush-all":
		return s, func() tea.Msg { return FlushRedisAllMsg{} }
	}
	return s, nil
}

// --- View rendering ---

func (s *CacheScreen) View() string {
	title := s.theme.Title.Render("Cache")

	// Input mode overlay
	if s.inputMode != "" {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", s.renderInput())
	}

	// Confirm mode overlay
	if s.confirmMode {
		return lipgloss.JoinVertical(lipgloss.Left, title, "",
			s.theme.WarnText.Render("  "+s.confirmPrompt))
	}

	if s.err != nil {
		errMsg := s.theme.ErrorText.Render(fmt.Sprintf("  Error: %v", s.err))
		help := s.theme.HelpDesc.Render("  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg, "", help)
	}

	// Redis status section
	statusSection := s.renderStatus()

	// Actions section
	actions := s.renderActions()

	help := s.theme.HelpDesc.Render("  f:flush db  F:flush all  o:opcache reset  esc:back")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", statusSection, "", actions, "", help)
}

func (s *CacheScreen) renderStatus() string {
	if s.status == nil {
		return s.theme.Subtitle.Render("  Loading...")
	}

	// Redis status
	redisLabel := "  Redis: "
	var redisInfo string
	if s.status.RedisRunning {
		statusBadge := s.theme.OkText.Render("running")
		version := ""
		if s.status.RedisVersion != "" {
			version = fmt.Sprintf("  v%s", s.status.RedisVersion)
		}
		memory := ""
		if s.status.RedisMemory != "" {
			memory = fmt.Sprintf("  mem: %s", s.status.RedisMemory)
		}
		redisInfo = statusBadge + s.theme.Subtitle.Render(version+memory)
	} else {
		redisInfo = s.theme.ErrorText.Render("not running")
	}
	redisLine := s.theme.Subtitle.Render(redisLabel) + redisInfo

	return redisLine
}

func (s *CacheScreen) renderActions() string {
	header := s.theme.HelpKey.Render("  Available Actions")
	actions := []string{
		"  [f] Flush Redis database (by number)",
		"  [F] Flush ALL Redis databases",
		"  [o] Reset PHP Opcache (restart FPM)",
	}

	var lines string
	for _, a := range actions {
		lines += s.theme.Subtitle.Render(a) + "\n"
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, lines)
}

func (s *CacheScreen) renderInput() string {
	var label string
	switch s.inputMode {
	case "flush-db":
		label = "  Redis DB number (0-15): "
	case "opcache-version":
		label = "  PHP version (e.g. 8.3): "
	}
	input := s.theme.Active.Render(s.inputBuffer + "█")
	help := s.theme.HelpDesc.Render("  enter:submit  esc:cancel")
	return lipgloss.JoinVertical(lipgloss.Left,
		s.theme.Subtitle.Render(label)+input, "", help)
}
