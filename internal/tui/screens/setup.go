package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/provisioner"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// --- State machine ---

type setupState int

const (
	stateChecklist  setupState = iota // select packages to install
	stateConfirm                      // confirm selection before install
	stateInstalling                   // installation in progress
	stateDone                         // show results summary
)

// --- Messages ---

// SetupProgressMsg wraps a ProgressEvent from the provisioner during install.
type SetupProgressMsg struct{ Event provisioner.ProgressEvent }

// SetupDoneMsg signals that all installations have completed.
type SetupDoneMsg struct{ Summary *provisioner.InstallSummary }

// RunSetupMsg tells app.go to start the install with the given package names.
type RunSetupMsg struct{ Names []string }

// --- Progress line for install state ---

type progressLine struct {
	Name    string
	Status  string // "installing", "done", "failed", "skipped"
	Message string
}

// --- SetupScreen ---

// SetupScreen is the TUI screen for detecting and installing LEMP packages.
// Static packages (Nginx, MariaDB, Redis) are shown at top-level.
// PHP versions are grouped under a submenu item.
type SetupScreen struct {
	theme *theme.Theme
	state setupState

	// Separated package groups
	staticPkgs []provisioner.PackageInfo // nginx, mariadb, redis
	phpPkgs    []provisioner.PackageInfo // PHP versions

	// Top-level: staticPkgs + PHP group item (last)
	cursor   int
	selected map[int]bool // static package index → selected

	// PHP submenu
	inPHPSub    bool
	phpCursor   int
	phpSelected map[int]bool // PHP package index → selected

	spinner  spinner.Model
	progress []progressLine
	summary  *provisioner.InstallSummary
	width    int
	height   int
}

// NewSetupScreen creates a SetupScreen with empty state.
func NewSetupScreen(t *theme.Theme) *SetupScreen {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Primary)

	return &SetupScreen{
		theme:       t,
		selected:    make(map[int]bool),
		phpSelected: make(map[int]bool),
		spinner:     sp,
	}
}

// SetPackages splits detection results into static and PHP groups.
// Pre-selects missing static packages. PHP versions are not pre-selected
// since users explicitly enter the submenu to choose.
func (s *SetupScreen) SetPackages(pkgs []provisioner.PackageInfo) {
	// Don't reset if user is already confirming or installing
	if s.state != stateChecklist && s.state != stateDone && len(s.staticPkgs) > 0 {
		return
	}

	s.staticPkgs = nil
	s.phpPkgs = nil
	s.selected = make(map[int]bool)
	s.phpSelected = make(map[int]bool)
	s.cursor = 0
	s.phpCursor = 0
	s.inPHPSub = false
	s.state = stateChecklist

	for _, pkg := range pkgs {
		if pkg.Name == "php" {
			s.phpPkgs = append(s.phpPkgs, pkg)
		} else {
			idx := len(s.staticPkgs)
			s.staticPkgs = append(s.staticPkgs, pkg)
			// Pre-select missing static packages
			if !pkg.Installed {
				s.selected[idx] = true
			}
		}
	}
}

func (s *SetupScreen) Init() tea.Cmd {
	return nil
}

func (s *SetupScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window resize globally (not per-state)
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		s.width = sz.Width
		s.height = sz.Height
		return s, nil
	}

	switch s.state {
	case stateChecklist:
		return s.updateChecklist(msg)
	case stateConfirm:
		return s.updateConfirm(msg)
	case stateInstalling:
		return s.updateInstalling(msg)
	case stateDone:
		return s.updateDone(msg)
	}
	return s, nil
}

func (s *SetupScreen) View() string {
	switch s.state {
	case stateChecklist:
		return s.viewChecklist()
	case stateConfirm:
		return s.viewConfirm()
	case stateInstalling:
		return s.viewInstalling()
	case stateDone:
		return s.viewDone()
	}
	return ""
}

// ScreenTitle returns the title for the header component.
func (s *SetupScreen) ScreenTitle() string {
	return "Setup"
}

// topLevelCount = static packages + 1 PHP group item
func (s *SetupScreen) topLevelCount() int {
	return len(s.staticPkgs) + 1
}

// isOnPHPGroup returns true when cursor is on the PHP group item
func (s *SetupScreen) isOnPHPGroup() bool {
	return s.cursor == len(s.staticPkgs)
}

// --- Checklist state ---

func (s *SetupScreen) updateChecklist(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.inPHPSub {
			return s.updatePHPSubmenu(msg)
		}
		return s.updateTopLevel(msg)
	}
	return s, nil
}

// updateTopLevel handles keys for the main checklist (static packages + PHP group)
func (s *SetupScreen) updateTopLevel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < s.topLevelCount()-1 {
			s.cursor++
		}
	case " ":
		// Toggle selection — only for static uninstalled packages, not PHP group
		if !s.isOnPHPGroup() && !s.staticPkgs[s.cursor].Installed {
			s.selected[s.cursor] = !s.selected[s.cursor]
		}
	case "enter", "right", "l":
		if s.isOnPHPGroup() {
			// Enter PHP submenu
			s.inPHPSub = true
			return s, nil
		}
		// Confirm selection if anything is selected
		names := s.selectedNames()
		if len(names) > 0 {
			s.state = stateConfirm
		}
	case "esc":
		return s, func() tea.Msg { return GoBackMsg{} }
	}
	return s, nil
}

// updatePHPSubmenu handles keys inside the PHP version submenu
func (s *SetupScreen) updatePHPSubmenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if s.phpCursor > 0 {
			s.phpCursor--
		}
	case "down", "j":
		if s.phpCursor < len(s.phpPkgs)-1 {
			s.phpCursor++
		}
	case " ":
		// Toggle PHP version selection (only uninstalled)
		if !s.phpPkgs[s.phpCursor].Installed {
			s.phpSelected[s.phpCursor] = !s.phpSelected[s.phpCursor]
		}
	case "enter":
		// Exit submenu and confirm selection if anything is selected
		s.inPHPSub = false
		names := s.selectedNames()
		if len(names) > 0 {
			s.state = stateConfirm
		}
	case "esc", "left", "h":
		// Back to top-level
		s.inPHPSub = false
	}
	return s, nil
}

func (s *SetupScreen) viewChecklist() string {
	if s.inPHPSub {
		return s.viewPHPSubmenu()
	}
	return s.viewTopLevel()
}

// viewTopLevel renders static packages + PHP group item
func (s *SetupScreen) viewTopLevel() string {
	title := s.theme.Title.Render("Server Setup")
	subtitle := s.theme.Subtitle.Render("Select packages to install")

	var items string

	// Static packages
	for i, pkg := range s.staticPkgs {
		cursor := "  "
		if i == s.cursor {
			cursor = "> "
		}

		check := "[ ]"
		if pkg.Installed {
			check = s.theme.OkText.Render("[✓]")
		} else if s.selected[i] {
			check = s.theme.Active.Render("[x]")
		}

		name := pkg.DisplayName
		var status string
		if pkg.Installed {
			name = s.theme.Inactive.Render(name)
			status = s.theme.OkText.Render(fmt.Sprintf(" (installed %s)", pkg.Version))
		} else {
			status = s.theme.ErrorText.Render(" (missing)")
		}

		if i == s.cursor {
			name = s.theme.Active.Render(pkg.DisplayName)
		}

		items += fmt.Sprintf("%s%s %s%s\n", cursor, check, name, status)
	}

	// PHP group item with summary
	phpIdx := len(s.staticPkgs)
	cursor := "  "
	if s.cursor == phpIdx {
		cursor = "> "
	}

	phpSummary := s.phpGroupSummary()
	phpLabel := "PHP Versions ▸"
	if s.cursor == phpIdx {
		phpLabel = s.theme.Active.Render(phpLabel)
	}
	items += fmt.Sprintf("%s    %s  %s\n", cursor, phpLabel, s.theme.Subtitle.Render(phpSummary))

	help := s.theme.HelpDesc.Render("\n  space: toggle  enter: confirm/open  esc: back")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", items, help)
}

// phpGroupSummary returns a summary like "2 selected, 3 available"
func (s *SetupScreen) phpGroupSummary() string {
	installed := 0
	selected := 0
	available := 0
	for i, pkg := range s.phpPkgs {
		if pkg.Installed {
			installed++
		} else {
			available++
			if s.phpSelected[i] {
				selected++
			}
		}
	}

	var parts []string
	if installed > 0 {
		parts = append(parts, fmt.Sprintf("%d installed", installed))
	}
	if selected > 0 {
		parts = append(parts, fmt.Sprintf("%d selected", selected))
	}
	if available > 0 {
		parts = append(parts, fmt.Sprintf("%d available", available))
	}
	if len(parts) == 0 {
		return "no versions"
	}
	return strings.Join(parts, ", ")
}

// viewPHPSubmenu renders the PHP version selection list
func (s *SetupScreen) viewPHPSubmenu() string {
	title := s.theme.Title.Render("PHP Versions")
	subtitle := s.theme.Subtitle.Render("Select PHP versions to install")

	var items string
	for i, pkg := range s.phpPkgs {
		cursor := "  "
		if i == s.phpCursor {
			cursor = "> "
		}

		check := "[ ]"
		if pkg.Installed {
			check = s.theme.OkText.Render("[✓]")
		} else if s.phpSelected[i] {
			check = s.theme.Active.Render("[x]")
		}

		name := pkg.DisplayName
		var status string
		if pkg.Installed {
			name = s.theme.Inactive.Render(name)
			status = s.theme.OkText.Render(fmt.Sprintf(" (installed %s)", pkg.Version))
		} else {
			status = s.theme.ErrorText.Render(" (missing)")
		}

		if i == s.phpCursor {
			name = s.theme.Active.Render(pkg.DisplayName)
		}

		items += fmt.Sprintf("%s%s %s%s\n", cursor, check, name, status)
	}

	help := s.theme.HelpDesc.Render("\n  space: toggle  enter: confirm  esc: back to setup")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", items, help)
}

// --- Confirm state ---

func (s *SetupScreen) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			names := s.selectedNames()
			// Init progress lines for install
			s.progress = make([]progressLine, len(names))
			for i, name := range names {
				s.progress[i] = progressLine{Name: name, Status: "pending"}
			}
			s.state = stateInstalling
			return s, tea.Batch(
				s.spinner.Tick,
				func() tea.Msg { return RunSetupMsg{Names: names} },
			)
		case "esc":
			s.state = stateChecklist
		}
	}
	return s, nil
}

func (s *SetupScreen) viewConfirm() string {
	title := s.theme.Title.Render("Confirm Installation")

	names := s.selectedNames()
	var list string
	for _, name := range names {
		list += fmt.Sprintf("  • %s\n", name)
	}

	prompt := fmt.Sprintf("\nInstall %d package(s)?\n\n%s", len(names), list)
	help := s.theme.HelpDesc.Render("\n  enter: install  esc: back")

	return lipgloss.JoinVertical(lipgloss.Left, title, prompt, help)
}

// --- Installing state ---

func (s *SetupScreen) updateInstalling(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd

	case SetupProgressMsg:
		s.updateProgress(msg.Event)
		return s, nil

	case SetupDoneMsg:
		s.summary = msg.Summary
		s.state = stateDone
		return s, nil
	}
	return s, nil
}

func (s *SetupScreen) updateProgress(ev provisioner.ProgressEvent) {
	for i := range s.progress {
		if s.progress[i].Name == ev.PackageName {
			switch ev.Status {
			case provisioner.ProgressStarting:
				s.progress[i].Status = "installing"
			case provisioner.ProgressDone:
				s.progress[i].Status = "done"
				s.progress[i].Message = ev.Message
			case provisioner.ProgressError:
				s.progress[i].Status = "failed"
				s.progress[i].Message = ev.Message
			}
			return
		}
	}
}

func (s *SetupScreen) viewInstalling() string {
	title := s.theme.Title.Render("Installing")
	spin := s.spinner.View() + " Installing packages..."

	var lines string
	for _, p := range s.progress {
		var icon, status string
		switch p.Status {
		case "pending":
			icon = "○"
			status = s.theme.Inactive.Render("waiting")
		case "installing":
			icon = s.spinner.View()
			status = s.theme.Active.Render("installing...")
		case "done":
			icon = s.theme.OkText.Render("✓")
			status = s.theme.OkText.Render("done")
		case "failed":
			icon = s.theme.ErrorText.Render("✗")
			status = s.theme.ErrorText.Render("failed: " + p.Message)
		}
		lines += fmt.Sprintf("  %s %s  %s\n", icon, p.Name, status)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, "", spin, "", lines)
}

// --- Done state ---

func (s *SetupScreen) updateDone(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter", "esc":
			return s, func() tea.Msg { return GoBackMsg{} }
		}
	}
	return s, nil
}

func (s *SetupScreen) viewDone() string {
	title := s.theme.Title.Render("Setup Complete")

	// Count results
	var installed, skipped, failed int
	if s.summary != nil {
		for _, r := range s.summary.Results {
			switch r.Status {
			case provisioner.StatusInstalled:
				installed++
			case provisioner.StatusSkipped:
				skipped++
			case provisioner.StatusFailed:
				failed++
			}
		}
	}

	summary := fmt.Sprintf("\n  %s  %s  %s  Time: %s\n",
		s.theme.OkText.Render(fmt.Sprintf("%d installed", installed)),
		s.theme.WarnText.Render(fmt.Sprintf("%d skipped", skipped)),
		s.theme.ErrorText.Render(fmt.Sprintf("%d failed", failed)),
		s.formatDuration(),
	)

	// Per-package results
	var details string
	if s.summary != nil {
		for _, r := range s.summary.Results {
			var icon string
			switch r.Status {
			case provisioner.StatusInstalled:
				icon = s.theme.OkText.Render("✓")
			case provisioner.StatusSkipped:
				icon = s.theme.WarnText.Render("–")
			case provisioner.StatusFailed:
				icon = s.theme.ErrorText.Render("✗")
			}
			details += fmt.Sprintf("  %s %s: %s\n", icon, r.Package, r.Message)
		}
	}

	help := s.theme.HelpDesc.Render("\n  enter/esc: back to dashboard")
	return lipgloss.JoinVertical(lipgloss.Left, title, summary, details, help)
}

// --- Helpers ---

// selectedNames returns package names for all selected (uninstalled) packages
// from both static and PHP groups.
func (s *SetupScreen) selectedNames() []string {
	var names []string

	// Static packages
	for i, pkg := range s.staticPkgs {
		if s.selected[i] && !pkg.Installed {
			names = append(names, pkg.Name)
		}
	}

	// PHP packages
	for i, pkg := range s.phpPkgs {
		if s.phpSelected[i] && !pkg.Installed {
			ver := strings.TrimPrefix(pkg.DisplayName, "PHP ")
			names = append(names, "php"+ver)
		}
	}

	return names
}

func (s *SetupScreen) formatDuration() string {
	if s.summary == nil {
		return "0s"
	}
	d := s.summary.TotalTime
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-d.Minutes()*60)
	}
	return fmt.Sprintf("%.0fs", d.Seconds())
}
