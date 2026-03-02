package theme

import "github.com/charmbracelet/lipgloss"

// Color palette for juiscript TUI.
// Consistent across all screens for professional look.
var (
	// Brand colors
	Primary   = lipgloss.Color("#7C3AED") // violet
	Secondary = lipgloss.Color("#06B6D4") // cyan
	Accent    = lipgloss.Color("#F59E0B") // amber

	// Status colors
	Success = lipgloss.Color("#10B981") // green
	Warning = lipgloss.Color("#F59E0B") // amber
	Error   = lipgloss.Color("#EF4444") // red
	Info    = lipgloss.Color("#3B82F6") // blue

	// Neutral colors
	TextPrimary   = lipgloss.Color("#F8FAFC")
	TextSecondary = lipgloss.Color("#94A3B8")
	TextMuted     = lipgloss.Color("#64748B")
	Background    = lipgloss.Color("#0F172A") // dark navy
	Surface       = lipgloss.Color("#1E293B")
	Border        = lipgloss.Color("#334155")
)

// Theme holds pre-computed styles used throughout the TUI.
// Caching styles here avoids re-creating them on every render.
type Theme struct {
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Header    lipgloss.Style
	StatusBar lipgloss.Style
	Panel     lipgloss.Style
	Active    lipgloss.Style
	Inactive  lipgloss.Style
	ErrorText lipgloss.Style
	WarnText  lipgloss.Style
	OkText    lipgloss.Style
	HelpKey   lipgloss.Style
	HelpDesc  lipgloss.Style
}

// New creates a Theme with pre-built styles.
func New() *Theme {
	return &Theme{
		Title: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1),

		Subtitle: lipgloss.NewStyle().
			Foreground(TextSecondary),

		Header: lipgloss.NewStyle().
			Foreground(TextPrimary).
			Background(Primary).
			Bold(true).
			Padding(0, 2),

		StatusBar: lipgloss.NewStyle().
			Foreground(TextSecondary).
			Background(Surface).
			Padding(0, 1),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Border).
			Padding(1, 2),

		Active: lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true),

		Inactive: lipgloss.NewStyle().
			Foreground(TextMuted),

		ErrorText: lipgloss.NewStyle().
			Foreground(Error),

		WarnText: lipgloss.NewStyle().
			Foreground(Warning),

		OkText: lipgloss.NewStyle().
			Foreground(Success),

		HelpKey: lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(TextMuted),
	}
}
