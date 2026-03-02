package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// SiteCreate is a form for creating a new site.
// Uses a simple step-by-step approach (field-by-field input).
type SiteCreate struct {
	theme      *theme.Theme
	step       int
	domain     string
	projType   site.ProjectType
	phpVersion string
	createDB   bool
	input      string // current input buffer
	err        error
	width      int
	height     int
}

// Steps for site creation form
const (
	stepDomain = iota
	stepProjectType
	stepPHPVersion
	stepCreateDB
	stepConfirm
)

var projectTypes = []site.ProjectType{site.ProjectLaravel, site.ProjectWordPress}
var phpVersions = []string{"8.3", "8.2", "8.1", "8.0"}

func NewSiteCreate(t *theme.Theme) *SiteCreate {
	return &SiteCreate{
		theme:      t,
		projType:   site.ProjectLaravel,
		phpVersion: "8.3",
	}
}

// Reset clears the form for a new entry.
func (s *SiteCreate) Reset() {
	s.step = stepDomain
	s.domain = ""
	s.projType = site.ProjectLaravel
	s.phpVersion = "8.3"
	s.createDB = true
	s.input = ""
	s.err = nil
}

func (s *SiteCreate) Init() tea.Cmd { return nil }

func (s *SiteCreate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			s.Reset()
			return s, func() tea.Msg { return GoBackMsg{} }

		case "enter":
			return s.handleEnter()

		case "tab", "down", "j":
			s.handleNext()

		case "shift+tab", "up", "k":
			s.handlePrev()

		case "backspace":
			if s.step == stepDomain && len(s.input) > 0 {
				s.input = s.input[:len(s.input)-1]
			}

		default:
			// Text input for domain step
			if s.step == stepDomain && len(msg.String()) == 1 {
				s.input += msg.String()
			}
		}
	}

	return s, nil
}

func (s *SiteCreate) handleEnter() (*SiteCreate, tea.Cmd) {
	switch s.step {
	case stepDomain:
		if err := site.ValidateDomain(s.input); err != nil {
			s.err = err
			return s, nil
		}
		s.domain = s.input
		s.err = nil
		s.step++

	case stepProjectType:
		s.step++

	case stepPHPVersion:
		s.step++

	case stepCreateDB:
		s.step++

	case stepConfirm:
		// Submit the form
		opts := site.CreateOptions{
			Domain:      s.domain,
			ProjectType: s.projType,
			PHPVersion:  s.phpVersion,
			CreateDB:    s.createDB,
		}
		s.Reset()
		return s, func() tea.Msg { return CreateSiteMsg{Options: opts} }
	}

	return s, nil
}

func (s *SiteCreate) handleNext() {
	switch s.step {
	case stepProjectType:
		// Cycle through project types
		for i, pt := range projectTypes {
			if pt == s.projType {
				s.projType = projectTypes[(i+1)%len(projectTypes)]
				break
			}
		}
	case stepPHPVersion:
		for i, v := range phpVersions {
			if v == s.phpVersion {
				s.phpVersion = phpVersions[(i+1)%len(phpVersions)]
				break
			}
		}
	case stepCreateDB:
		s.createDB = !s.createDB
	}
}

func (s *SiteCreate) handlePrev() {
	switch s.step {
	case stepProjectType:
		for i, pt := range projectTypes {
			if pt == s.projType {
				idx := (i - 1 + len(projectTypes)) % len(projectTypes)
				s.projType = projectTypes[idx]
				break
			}
		}
	case stepPHPVersion:
		for i, v := range phpVersions {
			if v == s.phpVersion {
				idx := (i - 1 + len(phpVersions)) % len(phpVersions)
				s.phpVersion = phpVersions[idx]
				break
			}
		}
	case stepCreateDB:
		s.createDB = !s.createDB
	}
}

func (s *SiteCreate) View() string {
	title := s.theme.Title.Render("Create New Site")

	var fields string

	// Domain field
	domainLabel := s.fieldLabel("Domain:", s.step == stepDomain)
	domainValue := s.input
	if s.step > stepDomain {
		domainValue = s.domain
	}
	if s.step == stepDomain {
		domainValue += "_" // cursor
	}
	fields += fmt.Sprintf("  %s %s\n", domainLabel, domainValue)

	// Project type
	if s.step >= stepProjectType {
		typeLabel := s.fieldLabel("Type:", s.step == stepProjectType)
		fields += fmt.Sprintf("  %s %s\n", typeLabel, s.projType)
	}

	// PHP version
	if s.step >= stepPHPVersion {
		phpLabel := s.fieldLabel("PHP:", s.step == stepPHPVersion)
		fields += fmt.Sprintf("  %s %s\n", phpLabel, s.phpVersion)
	}

	// Create DB toggle
	if s.step >= stepCreateDB {
		dbLabel := s.fieldLabel("Create DB:", s.step == stepCreateDB)
		dbVal := "yes"
		if !s.createDB {
			dbVal = "no"
		}
		fields += fmt.Sprintf("  %s %s\n", dbLabel, dbVal)
	}

	// Confirm step
	if s.step == stepConfirm {
		confirm := s.theme.OkText.Render("\n  Press Enter to create site, Esc to cancel")
		fields += confirm
	}

	// Error display
	var errLine string
	if s.err != nil {
		errLine = "\n" + s.theme.ErrorText.Render(fmt.Sprintf("  Error: %v", s.err))
	}

	help := s.theme.HelpDesc.Render("\n  enter:next  tab/j/k:cycle options  esc:cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", fields, errLine, help)
}

func (s *SiteCreate) fieldLabel(label string, active bool) string {
	if active {
		return s.theme.Active.Render(label)
	}
	return s.theme.Subtitle.Render(label)
}

func (s *SiteCreate) ScreenTitle() string { return "Create Site" }

// CreateSiteMsg carries the form data to the app for processing.
type CreateSiteMsg struct {
	Options site.CreateOptions
}
