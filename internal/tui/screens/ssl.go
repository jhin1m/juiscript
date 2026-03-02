package screens

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/ssl"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// SSLScreen displays SSL certificates and provides management actions.
type SSLScreen struct {
	theme  *theme.Theme
	certs  []ssl.CertInfo
	cursor int
	width  int
	height int
	err    error
}

// NewSSLScreen creates the SSL management screen.
func NewSSLScreen(t *theme.Theme) *SSLScreen {
	return &SSLScreen{theme: t}
}

// SetCerts updates the certificate list.
func (s *SSLScreen) SetCerts(certs []ssl.CertInfo) {
	s.certs = certs
	s.err = nil
}

// SetError sets an error to display.
func (s *SSLScreen) SetError(err error) {
	s.err = err
}

func (s *SSLScreen) Init() tea.Cmd { return nil }

func (s *SSLScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return s, func() tea.Msg { return ObtainCertMsg{} }
		case "r":
			if len(s.certs) > 0 {
				return s, func() tea.Msg {
					return RevokeCertMsg{Domain: s.certs[s.cursor].Domain}
				}
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

	if s.err != nil {
		errMsg := s.theme.ErrorText.Render(fmt.Sprintf("Error: %v", s.err))
		return lipgloss.JoinVertical(lipgloss.Left, title, "", errMsg)
	}

	if len(s.certs) == 0 {
		empty := s.theme.Subtitle.Render("  No SSL certificates found.")
		help := s.theme.HelpDesc.Render("  o:obtain  esc:back")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", empty, "", help)
	}

	// Table header
	header := fmt.Sprintf("  %-35s %-12s %-10s %s", "DOMAIN", "DAYS LEFT", "STATUS", "ISSUER")
	headerStyle := s.theme.HelpKey.Render(header)

	// Table rows
	var rows string
	for idx, cert := range s.certs {
		cursor := "  "
		style := s.theme.Inactive
		if idx == s.cursor {
			cursor = "> "
			style = s.theme.Active
		}

		// Color-code status based on days left
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

// ScreenTitle returns the title for the header component.
func (s *SSLScreen) ScreenTitle() string { return "SSL" }

// statusLabel returns a human-readable status string.
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

// statusStyle returns the appropriate color for the cert status.
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
type ObtainCertMsg struct{}

type RevokeCertMsg struct {
	Domain string
}

type RenewCertMsg struct {
	Domain string
}
