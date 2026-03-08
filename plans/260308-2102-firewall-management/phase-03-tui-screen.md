# Phase 3: TUI Screen

## Context

- Parent: [plan.md](plan.md)
- Dependencies: Phase 1 (backend manager)
- Pattern: `internal/tui/screens/services.go`

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-03-08 |
| Priority | P1 |
| Effort | 2h |
| Status | done |

Create `internal/tui/screens/firewall.go` with two-tab layout: UFW Rules and Blocked IPs. Follows `screens/services.go` pattern for struct, SetData, Update/View. Uses form/confirm/toast components from existing library.

## Key Insights

- Services screen pattern: struct with theme, cursor, data slice, key handlers emitting messages
- Forms use `components.FormField` with `FieldText` or `FieldSelect` types
- Confirmation for destructive actions (delete rule, unblock IP)
- Spinner for async operations; toast for results
- Tab switching is simple: track activeTab int, render conditionally

## Requirements

1. Two tabs: "UFW Rules" and "Blocked IPs" (tab key switches)
2. UFW tab: table of rules, cursor navigation, actions: open port (o), close port (c), delete rule (d)
3. Blocked IPs tab: list by jail, actions: unblock (u), block new (b)
4. Forms for port input and IP input
5. Confirmation for destructive ops (delete rule, unblock)
6. Async operations with toast notifications

## Architecture

```
internal/tui/screens/
  firewall.go    # FirewallScreen struct + messages
```

## Related Code Files

- `internal/tui/screens/services.go` - Screen struct pattern
- `internal/tui/screens/php.go` - Form + spinner + confirm pattern
- `internal/tui/screens/database.go` - Form with text input
- `internal/tui/components/form.go` - Form component
- `internal/tui/components/confirm.go` - Confirm dialog

## Implementation Steps

### Step 1: Screen struct and messages

```go
package screens

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/jhin1m/juiscript/internal/firewall"
    "github.com/jhin1m/juiscript/internal/tui/components"
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

// FirewallScreen shows UFW rules and Fail2ban blocked IPs.
type FirewallScreen struct {
    theme     *theme.Theme
    activeTab int // 0 = UFW Rules, 1 = Blocked IPs

    // UFW data
    ufwStatus *firewall.UFWStatus
    ufwCursor int

    // Fail2ban data
    jails    []firewall.F2bJailStatus
    f2bItems []f2bListItem // flattened for cursor navigation
    f2bCursor int

    // Components
    form    *components.FormModel
    confirm *components.ConfirmModel
    spinner *components.SpinnerModel

    formActive    bool
    confirmActive bool
    pendingAction string
    pendingTarget interface{}

    width  int
    height int
    err    error
}

// f2bListItem is a flattened view of jail+IP for cursor navigation.
type f2bListItem struct {
    Jail string
    IP   string
}

func NewFirewallScreen(t *theme.Theme) *FirewallScreen {
    return &FirewallScreen{
        theme:   t,
        form:    components.NewForm(t, "", nil),
        confirm: components.NewConfirm(t),
        spinner: components.NewSpinner(t),
    }
}
```

### Step 2: SetData methods

```go
func (s *FirewallScreen) SetUFWStatus(status *firewall.UFWStatus) {
    s.ufwStatus = status
    s.err = nil
}

func (s *FirewallScreen) SetJails(jails []firewall.F2bJailStatus) {
    s.jails = jails
    // Flatten into list items for cursor navigation
    s.f2bItems = nil
    for _, j := range jails {
        if len(j.BannedIPs) == 0 {
            continue
        }
        for _, ip := range j.BannedIPs {
            s.f2bItems = append(s.f2bItems, f2bListItem{Jail: j.Name, IP: ip})
        }
    }
    s.err = nil
}

func (s *FirewallScreen) SetError(err error) { s.err = err }
func (s *FirewallScreen) StopSpinner()        { s.spinner.Stop() }
func (s *FirewallScreen) ScreenTitle() string  { return "Firewall" }
```

### Step 3: Update - key handling with form/confirm priority

```go
func (s *FirewallScreen) Init() tea.Cmd { return nil }

func (s *FirewallScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Component priority: form > confirm > spinner > normal
    if s.formActive {
        return s.updateForm(msg)
    }
    if s.confirmActive {
        return s.updateConfirm(msg)
    }

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        s.width = msg.Width
        s.height = msg.Height

    case tea.KeyMsg:
        switch msg.String() {
        case "tab":
            s.activeTab = (s.activeTab + 1) % 2

        case "up", "k":
            s.moveCursor(-1)
        case "down", "j":
            s.moveCursor(1)

        // UFW tab actions
        case "o": // open port
            if s.activeTab == 0 {
                return s, s.showOpenPortForm()
            }
        case "c": // close port
            if s.activeTab == 0 {
                return s, s.showClosePortForm()
            }
        case "d": // delete rule
            if s.activeTab == 0 && s.ufwStatus != nil && s.ufwCursor < len(s.ufwStatus.Rules) {
                rule := s.ufwStatus.Rules[s.ufwCursor]
                s.confirm.Show(fmt.Sprintf("Delete rule %d (%s)?", rule.Num, rule.To))
                s.confirmActive = true
                s.pendingAction = "delete-rule"
                s.pendingTarget = rule.Num
            }

        // Blocked IPs tab actions
        case "u": // unblock
            if s.activeTab == 1 && s.f2bCursor < len(s.f2bItems) {
                item := s.f2bItems[s.f2bCursor]
                s.confirm.Show(fmt.Sprintf("Unban %s from %s?", item.IP, item.Jail))
                s.confirmActive = true
                s.pendingAction = "unban"
                s.pendingTarget = item
            }
        case "b": // block new IP
            if s.activeTab == 1 {
                return s, s.showBanIPForm()
            }

        case "esc", "q":
            return s, func() tea.Msg { return GoBackMsg{} }
        }
    }
    return s, nil
}
```

### Step 4: Form handlers

```go
func (s *FirewallScreen) showOpenPortForm() tea.Cmd {
    fields := []components.FormField{
        {Key: "port", Label: "Port Number", Type: components.FieldText},
        {Key: "protocol", Label: "Protocol", Type: components.FieldSelect,
            Options: []string{"tcp", "udp", "both"}, Default: "both"},
    }
    s.form = components.NewForm(s.theme, "Open Port", fields)
    s.formActive = true
    s.pendingAction = "open-port"
    return nil
}

func (s *FirewallScreen) showClosePortForm() tea.Cmd {
    fields := []components.FormField{
        {Key: "port", Label: "Port Number", Type: components.FieldText},
        {Key: "protocol", Label: "Protocol", Type: components.FieldSelect,
            Options: []string{"tcp", "udp", "both"}, Default: "both"},
    }
    s.form = components.NewForm(s.theme, "Close Port", fields)
    s.formActive = true
    s.pendingAction = "close-port"
    return nil
}

func (s *FirewallScreen) showBanIPForm() tea.Cmd {
    // Build jail options from available jails
    jailNames := []string{"sshd"}
    for _, j := range s.jails {
        found := false
        for _, n := range jailNames {
            if n == j.Name { found = true; break }
        }
        if !found {
            jailNames = append(jailNames, j.Name)
        }
    }
    fields := []components.FormField{
        {Key: "ip", Label: "IP Address", Type: components.FieldText},
        {Key: "jail", Label: "Jail", Type: components.FieldSelect,
            Options: jailNames, Default: "sshd"},
    }
    s.form = components.NewForm(s.theme, "Ban IP", fields)
    s.formActive = true
    s.pendingAction = "ban-ip"
    return nil
}

func (s *FirewallScreen) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
    updated, cmd := s.form.Update(msg)
    s.form = updated.(*components.FormModel)

    switch msg.(type) {
    case components.FormSubmitMsg:
        s.formActive = false
        return s, s.handleFormSubmit(msg.(components.FormSubmitMsg).Values)
    case components.FormCancelMsg:
        s.formActive = false
    }
    return s, cmd
}

func (s *FirewallScreen) handleFormSubmit(values map[string]string) tea.Cmd {
    switch s.pendingAction {
    case "open-port":
        port, _ := strconv.Atoi(values["port"])
        return func() tea.Msg {
            return OpenPortMsg{Port: port, Protocol: values["protocol"]}
        }
    case "close-port":
        port, _ := strconv.Atoi(values["port"])
        return func() tea.Msg {
            return ClosePortMsg{Port: port, Protocol: values["protocol"]}
        }
    case "ban-ip":
        return func() tea.Msg {
            return BanIPMsg{IP: values["ip"], Jail: values["jail"]}
        }
    }
    return nil
}
```

### Step 5: Confirm handler

```go
func (s *FirewallScreen) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
    updated, cmd := s.confirm.Update(msg)
    s.confirm = updated.(*components.ConfirmModel)

    switch msg.(type) {
    case components.ConfirmYesMsg:
        s.confirmActive = false
        return s, s.handleConfirm()
    case components.ConfirmNoMsg:
        s.confirmActive = false
        s.pendingAction = ""
    }
    return s, cmd
}

func (s *FirewallScreen) handleConfirm() tea.Cmd {
    switch s.pendingAction {
    case "delete-rule":
        ruleNum := s.pendingTarget.(int)
        return func() tea.Msg { return DeleteUFWRuleMsg{RuleNum: ruleNum} }
    case "unban":
        item := s.pendingTarget.(f2bListItem)
        return func() tea.Msg { return UnbanIPMsg{IP: item.IP, Jail: item.Jail} }
    }
    return nil
}
```

### Step 6: View rendering

```go
func (s *FirewallScreen) View() string {
    if s.formActive {
        return s.form.View()
    }
    if s.confirmActive {
        return s.confirm.View()
    }

    title := s.theme.Title.Render("Firewall")

    if s.err != nil {
        return lipgloss.JoinVertical(lipgloss.Left, title, "",
            s.theme.ErrorText.Render(fmt.Sprintf("Error: %v", s.err)))
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
    if s.ufwStatus.Active {
        statusStr = s.theme.OkText.Render("active")
    }
    statusLine := fmt.Sprintf("  UFW: %s", statusStr)

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
        row := fmt.Sprintf("%s%-6d %s %-15s %-15s", cursor,
            r.Num, style.Render(fmt.Sprintf("%-20s", r.To)), r.Action, r.From)
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

func (s *FirewallScreen) renderHelp() string {
    if s.activeTab == 0 {
        return s.theme.HelpDesc.Render("  o:open  c:close  d:delete  tab:switch  esc:back")
    }
    return s.theme.HelpDesc.Render("  b:block  u:unblock  tab:switch  esc:back")
}

func (s *FirewallScreen) moveCursor(delta int) {
    if s.activeTab == 0 && s.ufwStatus != nil {
        s.ufwCursor += delta
        if s.ufwCursor < 0 { s.ufwCursor = 0 }
        max := len(s.ufwStatus.Rules) - 1
        if max < 0 { max = 0 }
        if s.ufwCursor > max { s.ufwCursor = max }
    } else if s.activeTab == 1 {
        s.f2bCursor += delta
        if s.f2bCursor < 0 { s.f2bCursor = 0 }
        max := len(s.f2bItems) - 1
        if max < 0 { max = 0 }
        if s.f2bCursor > max { s.f2bCursor = max }
    }
}
```

## Todo

- [ ] Create `internal/tui/screens/firewall.go`
- [ ] Implement FirewallScreen struct with theme, tabs, cursors
- [ ] Implement message types (OpenPortMsg, ClosePortMsg, DeleteUFWRuleMsg, BanIPMsg, UnbanIPMsg)
- [ ] Implement SetUFWStatus, SetJails, SetError, StopSpinner methods
- [ ] Implement two-tab View rendering (UFW Rules + Blocked IPs)
- [ ] Implement form flows for open port, close port, ban IP
- [ ] Implement confirmation for delete rule and unban IP
- [ ] Implement cursor navigation per tab
- [ ] Implement help text per tab

## Success Criteria

- Tab switching works with tab key
- UFW rules displayed in aligned table with cursor
- Blocked IPs displayed with jail grouping
- Forms collect port/protocol and IP/jail inputs
- Confirmation gates destructive actions
- Messages emitted correctly for app-level handling

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Form component API mismatch | Build errors | Verify exact FormField/FormModel signatures from components/form.go |
| Tab rendering alignment | UI glitch | Test with various terminal widths |

## Security Considerations

- Port input from form parsed as int in screen (reject non-numeric before message)
- IP input validated in backend manager (defense in depth)

## Next Steps

After completing: proceed to Phase 4 (integration wiring)
