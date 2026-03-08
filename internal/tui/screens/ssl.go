package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/site"
	"github.com/jhin1m/juiscript/internal/ssl"
	"github.com/jhin1m/juiscript/internal/tui/components"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

func validateEmail(email string) error {
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("valid email required (e.g., admin@example.com)")
	}
	return nil
}

// SSLScreen displays SSL certificates and provides management actions.
type SSLScreen struct {
	theme  *theme.Theme
	certs  []ssl.CertInfo
	cursor int
	width  int
	height int
	err    error
	// Form for obtain cert
	form       *components.FormModel
	formActive bool
	// Confirm for revoke
	confirm       *components.ConfirmModel
	pendingTarget string // domain to revoke
}

func NewSSLScreen(t *theme.Theme) *SSLScreen {
	return &SSLScreen{
		theme:   t,
		confirm: components.NewConfirm(t),
	}
}

func (s *SSLScreen) SetCerts(certs []ssl.CertInfo) {
	s.certs = certs
	s.err = nil
}

func (s *SSLScreen) SetError(err error) {
	s.err = err
}

// StopSpinner is a no-op kept for App compatibility.
func (s *SSLScreen) StopSpinner() {}

func (s *SSLScreen) Init() tea.Cmd { return nil }

func (s *SSLScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm dialog
	if s.confirm.Active() {
		_, cmd := s.confirm.Update(msg)
		if cmd != nil {
			domain := s.pendingTarget
			s.pendingTarget = ""
			return s, func() tea.Msg {
				result := cmd()
				switch result.(type) {
				case components.ConfirmYesMsg:
					return RevokeCertMsg{Domain: domain}
				case components.ConfirmNoMsg:
					return nil
				default:
					return result
				}
			}
		}
		return s, nil
	}

	// Form
	if s.formActive {
		_, cmd := s.form.Update(msg)
		if cmd != nil {
			result := cmd()
			switch v := result.(type) {
			case components.FormSubmitMsg:
				s.formActive = false
				domain := v.Values["domain"]
				email := v.Values["email"]
				// Don't block UI — obtain runs in background
				return s, func() tea.Msg {
					return ObtainCertMsg{Domain: domain, Email: email}
				}
			case components.FormCancelMsg:
				s.formActive = false
				return s, nil
			}
		}
		return s, nil
	}

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
			if s.cursor < len(s.certs)-1 {
				s.cursor++
			}
		case "o":
			// Obtain cert form
			fields := []components.FormField{
				{Key: "domain", Label: "Domain", Type: components.FieldText,
					Placeholder: "example.com", Validate: site.ValidateDomain},
				{Key: "email", Label: "Email", Type: components.FieldText,
					Placeholder: "admin@example.com", Validate: validateEmail},
			}
			s.form = components.NewForm(s.theme, "Obtain SSL Certificate", fields)
			s.formActive = true
		case "r":
			if len(s.certs) > 0 {
				domain := s.certs[s.cursor].Domain
				s.pendingTarget = domain
				s.confirm.Show(fmt.Sprintf("Revoke certificate for '%s'?", domain))
			}
		case "f":
			if len(s.certs) > 0 {
				return s, func() tea.Msg {
					return RenewCertMsg{Domain: s.certs[s.cursor].Domain}
				}
			}
		case "esc", "q":
			return s, func() tea.Msg { return GoBackMsg{} }
		}
	}

	return s, nil
}

func (s *SSLScreen) View() string {
	title := s.theme.Title.Render("SSL Certificates")

	if s.confirm.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", s.confirm.View())
	}

	if s.formActive && s.form != nil {
		return s.form.View()
	}

	if s.err != nil {
		errMsg := s.theme.ErrorText.Render(fmt.Sprintf("Error: %v", s.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(s.certs) == 0 {
		empty := s.theme.Subtitle.Render("  No SSL certificates found.")
		help := s.theme.HelpDesc.Render("  o:obtain  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	header := fmt.Sprintf("  %-35s %-12s %-10s %s", "DOMAIN", "DAYS LEFT", "STATUS", "ISSUER")
	headerStyle := s.theme.HelpKey.Render(header)

	var rows string
	for idx, cert := range s.certs {
		cursor := "  "
		style := s.theme.Inactive
		if idx == s.cursor {
			cursor = "> "
			style = s.theme.Active
		}

		status := statusLabel(cert)
		statusStyle := s.statusStyle(cert)

		row := fmt.Sprintf("%s%-35s %-12d %-10s %s",
			cursor,
			style.Render(cert.Domain),
			cert.DaysLeft,
			statusStyle.Render(status),
			cert.Issuer,
		)
		rows += row + "\n"
	}

	help := s.theme.HelpDesc.Render("  o:obtain  r:revoke  f:force-renew  esc:back")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", headerStyle, rows, help)
}

func (s *SSLScreen) ScreenTitle() string { return "SSL" }

func statusLabel(cert ssl.CertInfo) string {
	switch {
	case !cert.Valid:
		return "EXPIRED"
	case cert.DaysLeft <= 7:
		return "CRITICAL"
	case cert.DaysLeft <= 30:
		return "EXPIRING"
	default:
		return "VALID"
	}
}

func (s *SSLScreen) statusStyle(cert ssl.CertInfo) lipgloss.Style {
	switch {
	case !cert.Valid || cert.DaysLeft <= 7:
		return s.theme.ErrorText
	case cert.DaysLeft <= 30:
		return lipgloss.NewStyle().Foreground(theme.Warning)
	default:
		return s.theme.OkText
	}
}

// Messages for SSL screen actions.
type ObtainCertMsg struct {
	Domain string
	Email  string
}

type RevokeCertMsg struct {
	Domain string
}

type RenewCertMsg struct {
	Domain string
}
