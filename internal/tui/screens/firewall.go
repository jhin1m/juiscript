package screens

import (
	"fmt"
	"net"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/firewall"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// Firewall screen action messages (screen -> app)

type OpenPortMsg struct {
	Port     int
	Protocol string
}

type ClosePortMsg struct {
	Port     int
	Protocol string
}

type DeleteUFWRuleMsg struct {
	RuleNum int
}

type BanIPMsg struct {
	IP   string
	Jail string
}

type UnbanIPMsg struct {
	IP   string
	Jail string
}

// f2bListItem is a flattened view of jail+IP for cursor navigation.
type f2bListItem struct {
	Jail string
	IP   string
}

// FirewallScreen shows UFW rules and Fail2ban blocked IPs.
type FirewallScreen struct {
	theme     *theme.Theme
	activeTab int // 0 = UFW Rules, 1 = Blocked IPs

	// UFW data
	ufwStatus *firewall.UFWStatus
	ufwCursor int

	// Fail2ban data
	jails     []firewall.F2bJailStatus
	f2bItems  []f2bListItem
	f2bCursor int

	// Input mode for forms
	inputMode   string // "", "open-port", "close-port", "ban-ip"
	inputBuffer string
	inputProto  string // for port forms: tcp/udp/both
	inputJail   string // for ban-ip form

	// Confirm mode
	confirmMode   bool
	confirmPrompt string
	pendingAction string
	pendingTarget interface{}

	width  int
	height int
	err    error
}

func NewFirewallScreen(t *theme.Theme) *FirewallScreen {
	return &FirewallScreen{theme: t}
}

func (s *FirewallScreen) SetUFWStatus(status *firewall.UFWStatus) {
	s.ufwStatus = status
	s.err = nil
}

func (s *FirewallScreen) SetJails(jails []firewall.F2bJailStatus) {
	s.jails = jails
	s.f2bItems = nil
	for _, j := range jails {
		for _, ip := range j.BannedIPs {
			s.f2bItems = append(s.f2bItems, f2bListItem{Jail: j.Name, IP: ip})
		}
	}
	s.err = nil
}

func (s *FirewallScreen) SetError(err error) { s.err = err }
func (s *FirewallScreen) ScreenTitle() string { return "Firewall" }

func (s *FirewallScreen) Init() tea.Cmd { return nil }

func (s *FirewallScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		return s, nil

	case tea.KeyMsg:
		// Input mode: collecting text input
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

func (s *FirewallScreen) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		s.activeTab = (s.activeTab + 1) % 2

	case "up", "k":
		s.moveCursor(-1)
	case "down", "j":
		s.moveCursor(1)

	// UFW tab actions
	case "o":
		if s.activeTab == 0 {
			s.inputMode = "open-port"
			s.inputBuffer = ""
			s.inputProto = "both"
		}
	case "c":
		if s.activeTab == 0 {
			s.inputMode = "close-port"
			s.inputBuffer = ""
			s.inputProto = "both"
		}
	case "d":
		if s.activeTab == 0 && s.ufwStatus != nil && s.ufwCursor < len(s.ufwStatus.Rules) {
			rule := s.ufwStatus.Rules[s.ufwCursor]
			s.confirmMode = true
			s.confirmPrompt = fmt.Sprintf("Delete rule %d (%s)? [y/n]", rule.Num, rule.To)
			s.pendingAction = "delete-rule"
			s.pendingTarget = rule.Num
		}

	// Blocked IPs tab actions
	case "u":
		if s.activeTab == 1 && s.f2bCursor < len(s.f2bItems) {
			item := s.f2bItems[s.f2bCursor]
			s.confirmMode = true
			s.confirmPrompt = fmt.Sprintf("Unban %s from %s? [y/n]", item.IP, item.Jail)
			s.pendingAction = "unban"
			s.pendingTarget = item
		}
	case "b":
		if s.activeTab == 1 {
			s.inputMode = "ban-ip"
			s.inputBuffer = ""
			s.inputJail = "sshd"
		}

	case "esc", "q":
		return s, func() tea.Msg { return GoBackMsg{} }
	}
	return s, nil
}

func (s *FirewallScreen) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "tab":
		// Cycle protocol for port forms
		if s.inputMode == "open-port" || s.inputMode == "close-port" {
			switch s.inputProto {
			case "both":
				s.inputProto = "tcp"
			case "tcp":
				s.inputProto = "udp"
			default:
				s.inputProto = "both"
			}
		}
	default:
		// Only accept printable chars
		if len(msg.String()) == 1 {
			s.inputBuffer += msg.String()
		}
	}
	return s, nil
}

func (s *FirewallScreen) submitInput() (tea.Model, tea.Cmd) {
	mode := s.inputMode
	buf := s.inputBuffer
	s.inputMode = ""
	s.inputBuffer = ""

	switch mode {
	case "open-port":
		port, err := strconv.Atoi(buf)
		if err != nil || port < 1 || port > 65535 {
			s.err = fmt.Errorf("invalid port: %s", buf)
			return s, nil
		}
		proto := s.inputProto
		return s, func() tea.Msg { return OpenPortMsg{Port: port, Protocol: proto} }

	case "close-port":
		port, err := strconv.Atoi(buf)
		if err != nil || port < 1 || port > 65535 {
			s.err = fmt.Errorf("invalid port: %s", buf)
			return s, nil
		}
		proto := s.inputProto
		return s, func() tea.Msg { return ClosePortMsg{Port: port, Protocol: proto} }

	case "ban-ip":
		ip := buf
		if net.ParseIP(ip) == nil {
			s.err = fmt.Errorf("invalid IP address: %s", ip)
			return s, nil
		}
		jail := s.inputJail
		return s, func() tea.Msg { return BanIPMsg{IP: ip, Jail: jail} }
	}
	return s, nil
}

func (s *FirewallScreen) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		s.confirmMode = false
		return s.handleConfirm()
	case "n", "N", "esc":
		s.confirmMode = false
		s.pendingAction = ""
		s.pendingTarget = nil
	}
	return s, nil
}

func (s *FirewallScreen) handleConfirm() (tea.Model, tea.Cmd) {
	action := s.pendingAction
	target := s.pendingTarget
	s.pendingAction = ""
	s.pendingTarget = nil

	switch action {
	case "delete-rule":
		ruleNum := target.(int)
		return s, func() tea.Msg { return DeleteUFWRuleMsg{RuleNum: ruleNum} }
	case "unban":
		item := target.(f2bListItem)
		return s, func() tea.Msg { return UnbanIPMsg{IP: item.IP, Jail: item.Jail} }
	}
	return s, nil
}

func (s *FirewallScreen) moveCursor(delta int) {
	if s.activeTab == 0 && s.ufwStatus != nil {
		s.ufwCursor += delta
		if s.ufwCursor < 0 {
			s.ufwCursor = 0
		}
		max := len(s.ufwStatus.Rules) - 1
		if max < 0 {
			max = 0
		}
		if s.ufwCursor > max {
			s.ufwCursor = max
		}
	} else if s.activeTab == 1 {
		s.f2bCursor += delta
		if s.f2bCursor < 0 {
			s.f2bCursor = 0
		}
		max := len(s.f2bItems) - 1
		if max < 0 {
			max = 0
		}
		if s.f2bCursor > max {
			s.f2bCursor = max
		}
	}
}

// --- View rendering ---

func (s *FirewallScreen) View() string {
	title := s.theme.Title.Render("Firewall")

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

	// Tab bar
	tabs := s.renderTabs()

	// Content based on active tab
	var content string
	if s.activeTab == 0 {
		content = s.renderUFWTab()
	} else {
		content = s.renderBlockedTab()
	}

	help := s.renderHelp()
	return lipgloss.JoinVertical(lipgloss.Left, title, "", tabs, "", content, "", help)
}

func (s *FirewallScreen) renderTabs() string {
	tab0 := " UFW Rules "
	tab1 := " Blocked IPs "
	if s.activeTab == 0 {
		tab0 = s.theme.Active.Render(tab0)
		tab1 = s.theme.Inactive.Render(tab1)
	} else {
		tab0 = s.theme.Inactive.Render(tab0)
		tab1 = s.theme.Active.Render(tab1)
	}
	return "  " + tab0 + " " + tab1
}

func (s *FirewallScreen) renderUFWTab() string {
	if s.ufwStatus == nil {
		return s.theme.Subtitle.Render("  Loading...")
	}

	statusStr := "inactive"
	statusStyle := s.theme.Subtitle
	if s.ufwStatus.Active {
		statusStr = "active"
		statusStyle = s.theme.OkText
	}
	statusLine := fmt.Sprintf("  UFW: %s", statusStyle.Render(statusStr))

	if len(s.ufwStatus.Rules) == 0 {
		return statusLine + "\n\n" + s.theme.Subtitle.Render("  No rules configured.")
	}

	header := fmt.Sprintf("  %-6s %-20s %-15s %-15s", "NUM", "TO", "ACTION", "FROM")
	headerStyle := s.theme.HelpKey.Render(header)

	var rows string
	for i, r := range s.ufwStatus.Rules {
		cursor := "  "
		style := s.theme.Inactive
		if i == s.ufwCursor {
			cursor = "> "
			style = s.theme.Active
		}
		row := fmt.Sprintf("%s%-6d %s %-15s %-15s",
			cursor, r.Num, style.Render(fmt.Sprintf("%-20s", r.To)), r.Action, r.From)
		rows += row + "\n"
	}

	return lipgloss.JoinVertical(lipgloss.Left, statusLine, "", headerStyle, rows)
}

func (s *FirewallScreen) renderBlockedTab() string {
	if len(s.f2bItems) == 0 {
		return s.theme.Subtitle.Render("  No banned IPs.")
	}

	header := fmt.Sprintf("  %-20s %-15s", "JAIL", "IP")
	headerStyle := s.theme.HelpKey.Render(header)

	var rows string
	for i, item := range s.f2bItems {
		cursor := "  "
		style := s.theme.Inactive
		if i == s.f2bCursor {
			cursor = "> "
			style = s.theme.Active
		}
		row := fmt.Sprintf("%s%-20s %s", cursor, item.Jail, style.Render(item.IP))
		rows += row + "\n"
	}

	return lipgloss.JoinVertical(lipgloss.Left, headerStyle, rows)
}

func (s *FirewallScreen) renderInput() string {
	var label string
	switch s.inputMode {
	case "open-port":
		label = fmt.Sprintf("  Open port (protocol: %s, tab to change): ", s.inputProto)
	case "close-port":
		label = fmt.Sprintf("  Close port (protocol: %s, tab to change): ", s.inputProto)
	case "ban-ip":
		label = fmt.Sprintf("  Ban IP (jail: %s): ", s.inputJail)
	}
	input := s.theme.Active.Render(s.inputBuffer + "█")
	help := s.theme.HelpDesc.Render("  enter:submit  esc:cancel")
	return lipgloss.JoinVertical(lipgloss.Left,
		s.theme.Subtitle.Render(label)+input, "", help)
}

func (s *FirewallScreen) renderHelp() string {
	if s.activeTab == 0 {
		return s.theme.HelpDesc.Render("  o:open  c:close  d:delete  tab:switch  esc:back")
	}
	return s.theme.HelpDesc.Render("  b:block  u:unblock  tab:switch  esc:back")
}
