package screens

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// Default PHP versions used when dynamic list is not yet available.
var defaultPHPVersions = []string{"8.3", "8.2", "8.1", "8.0"}

// SiteCreate is a form screen for creating a new site.
// Uses the reusable FormModel component for step-by-step input.
type SiteCreate struct {
	theme       *theme.Theme
	form        *components.FormModel
	phpVersions []string // dynamic list from phpMgr; falls back to defaults
	width       int
	height      int
}

func NewSiteCreate(t *theme.Theme) *SiteCreate {
	s := &SiteCreate{
		theme:       t,
		phpVersions: defaultPHPVersions,
	}
	s.form = s.buildForm()
	return s
}

// SetPHPVersions updates available PHP versions and rebuilds the form.
func (s *SiteCreate) SetPHPVersions(versions []string) {
	if len(versions) > 0 {
		s.phpVersions = versions
		s.form = s.buildForm() // rebuild with new versions
	}
}

// buildForm creates the form with current phpVersions.
func (s *SiteCreate) buildForm() *components.FormModel {
	fields := []components.FormField{
		{Key: "domain", Label: "Domain", Type: components.FieldText,
			Placeholder: "example.com", Validate: site.ValidateDomain},
		{Key: "projectType", Label: "Type", Type: components.FieldSelect,
			Options: []string{string(site.ProjectLaravel), string(site.ProjectWordPress)},
			Default: string(site.ProjectLaravel)},
		{Key: "phpVersion", Label: "PHP", Type: components.FieldSelect,
			Options: s.phpVersions, Default: s.phpVersions[0]},
		{Key: "createDB", Label: "Create DB", Type: components.FieldConfirm,
			Default: "yes"},
	}
	return components.NewForm(s.theme, "Create New Site", fields)
}

// Reset clears the form for a new entry.
func (s *SiteCreate) Reset() {
	s.form = s.buildForm()
}

func (s *SiteCreate) Init() tea.Cmd { return nil }

func (s *SiteCreate) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil
	case tea.KeyMsg:
		// Delegate all keys to form
		_, cmd := s.form.Update(msg)
		if cmd == nil {
			return s, nil
		}
		// Wrap cmd to intercept form messages without double-firing
		return s, func() tea.Msg {
			result := cmd()
			switch v := result.(type) {
			case components.FormSubmitMsg:
				opts := site.CreateOptions{
					Domain:      v.Values["domain"],
					ProjectType: site.ProjectType(v.Values["projectType"]),
					PHPVersion:  v.Values["phpVersion"],
					CreateDB:    v.Values["createDB"] == "yes",
				}
				s.Reset()
				return CreateSiteMsg{Options: opts}
			case components.FormCancelMsg:
				s.Reset()
				return GoBackMsg{}
			default:
				return result
			}
		}
	}
	return s, nil
}

func (s *SiteCreate) View() string {
	return s.form.View()
}

func (s *SiteCreate) ScreenTitle() string { return "Create Site" }

// CreateSiteMsg carries the form data to the app for processing.
type CreateSiteMsg struct {
	Options site.CreateOptions
}
