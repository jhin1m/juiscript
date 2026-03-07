# Bubble Tea Form Input Patterns Research

## Overview
Comprehensive analysis of form input patterns in Bubble Tea/Bubbles (Go TUI framework). Examined official examples, component APIs, and best practices for building forms in terminal applications.

---

## 1. Text Input Components (charmbracelet/bubbles/textinput)

### Basic Single Field
```go
import "github.com/charmbracelet/bubbles/textinput"

m := textinput.New()
m.Placeholder = "Enter domain..."
m.CharLimit = 255      // Max characters (-1 = unlimited)
m.Width = 40            // Visible width (0 = full terminal)
m.Focus()               // Make interactive
```

### Configuration Options
- **CharLimit**: Max input length (0 or negative = unlimited)
- **Width**: Display width (0 = unlimited viewport)
- **Placeholder**: Ghost text when empty
- **EchoMode**: Hide password (TextEchoMode for normal, PasswordEchoMode for masks)
- **Validate**: Custom ValidateFunc for real-time validation

### Multi-Field Forms with Tab Navigation
Pattern from official `textinputs` example:

```go
type Model struct {
    inputs    []textinput.Model
    focusIndex int
}

// Focus/blur cycle
func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
    cmds := make([]tea.Cmd, len(m.inputs))

    for i := range m.inputs {
        m.inputs[i].Blur()
        if i == m.focusIndex {
            cmds[i] = m.inputs[i].Focus()
        }
    }
    return tea.Batch(cmds...)
}

// Tab navigation
case "tab", "down":
    m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
case "shift+tab", "up":
    m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
```

### Validation Strategy
- Use `ValidateFunc` field for real-time checks
- Don't trigger errors while typing—validate on submit only
- Store error state in model, display in View
- CharLimit prevents exceeding max length; validation checks format

---

## 2. List/Selection Components (charmbracelet/bubbles/list)

### List Item Selection
```go
import "github.com/charmbracelet/bubbles/list"

l := list.New(items, delegate, width, height)
l.SetShowTitle(false)
l.SetShowPagination(true)
l.SetShowHelp(true)

// Get selected item
selectedItem := l.SelectedItem().(ItemType)

// Move cursor
case "down", "j":
    l.CursorDown()
case "up", "k":
    l.CursorUp()
```

### Features
- Filtering support (optional)
- Pagination for large lists
- Help text display
- Custom ItemDelegate for rendering
- Supports mouse wheel (scroll)

### Picker Pattern (Version/Type Selection)
Custom cycle-through approach (used in juiscript sitecreate):
```go
options := []string{"8.3", "8.2", "8.1", "8.0"}
currentIdx := 0

case "down", "j", "tab":
    currentIdx = (currentIdx + 1) % len(options)
case "up", "k", "shift+tab":
    currentIdx = (currentIdx - 1 + len(options)) % len(options)
```

Lightweight for small fixed lists (3-5 items), preferred over full list component.

---

## 3. Confirmation Dialogs

### Pattern (No Built-in Component)
```go
type ConfirmModel struct {
    message string
    focused int  // 0=yes, 1=no
}

func (m *ConfirmModel) View() string {
    yes := "[ Yes ]"
    no := "[ No ]"
    if m.focused == 0 {
        yes = theme.Active.Render(yes)
    } else {
        no = theme.Active.Render(no)
    }
    return fmt.Sprintf("%s  %s  %s", m.message, yes, no)
}

case "tab", "right", "l": m.focused = 1
case "shift+tab", "left", "h": m.focused = 0
case "enter":
    if m.focused == 0 {
        return ConfirmYesMsg{}
    }
    return ConfirmNoMsg{}
```

### Best Practice
- No dedicated bubble component; build minimal model
- Store focused button index
- Highlight active option via theme styling
- Use lipgloss for layout

---

## 4. Spinner Component (charmbracelet/bubbles/spinner)

### Implementation
```go
import "github.com/charmbracelet/bubbles/spinner"

type Model struct {
    spinner spinner.Model
    loading bool
}

func NewModel() *Model {
    s := spinner.New()
    s.Spinner = spinner.Dot      // Predefined styles
    s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
    return &Model{spinner: s}
}

func (m *Model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.loading {
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m *Model) View() string {
    if m.loading {
        return fmt.Sprintf("%s Loading...", m.spinner.View())
    }
    return "Done"
}
```

### Pre-built Styles
`spinner.Line`, `spinner.Dot`, `spinner.MiniDot`, `spinner.Jump`, `spinner.Pulse`, `spinner.Points`, `spinner.Globe`, `spinner.Moon`, `spinner.Monkey`, `spinner.Meter`

---

## 5. Toast/Notification Patterns

### Temporary Message with Auto-Dismiss
```go
type Notification struct {
    Message string
    Expires time.Time
}

type Model struct {
    notification *Notification
    // ...
}

// Show toast for 3 seconds
func (m *Model) showNotification(msg string) tea.Cmd {
    m.notification = &Notification{
        Message: msg,
        Expires: time.Now().Add(3 * time.Second),
    }
    return tea.Tick(3 * time.Second, func(time.Time) tea.Msg {
        return NotificationExpiredMsg{}
    })
}

// In Update
case NotificationExpiredMsg:
    m.notification = nil

// In View
if m.notification != nil {
    if time.Now().Before(m.notification.Expires) {
        return theme.OkText.Render(m.notification.Message)
    }
}
```

### Design Notes
- No built-in toast component in Bubbles
- Use `tea.Tick()` for auto-dismiss timing
- Store expiration time for duration control
- Position at top or bottom via lipgloss layout

---

## 6. Form Validation Best Practices

### Validation Timing
1. **On Submit Only** (preferred for UX)
   - Collect input without error display
   - Validate on Enter/Submit button
   - Show all errors at once

2. **Real-time Validation** (textinput.Validate field)
   - Set ValidateFunc on textinput
   - Sets Err field when invalid
   - Better for format checks (email, domain)
   - Avoid blocking during typing

### Implementation Pattern
```go
// Domain validation
func ValidateDomain(s string) error {
    if len(s) == 0 {
        return fmt.Errorf("required")
    }
    if !strings.Contains(s, ".") {
        return fmt.Errorf("invalid domain format")
    }
    return nil
}

// Use in form
case "enter":
    if err := ValidateDomain(m.domain); err != nil {
        m.err = err
        return m, nil  // Stay on field
    }
    m.step++  // Move to next field
```

### Error Display
- Store error in model state
- Display below/adjacent to field via theme styling
- Clear error when user corrects input
- Example: `theme.ErrorText.Render(fmt.Sprintf("Error: %v", m.err))`

---

## 7. Multi-Field Form Navigation

### Step-by-Step Flow (juiscript pattern)
```go
const (
    stepDomain = iota
    stepProjectType
    stepPHPVersion
    stepCreateDB
    stepConfirm
)

// Show fields progressively
func (s *SiteCreate) View() string {
    // Domain (always visible once entered)
    if s.step >= stepDomain {
        render(domainField)
    }
    // Type (only after domain confirmed)
    if s.step >= stepProjectType {
        render(typeField)
    }
    // etc.
}
```

**Advantages**: Progressive disclosure, reduced cognitive load, smaller viewport requirement

### Tab Navigation Between Fields
```go
case "tab", "down":
    if m.focusIndex < len(m.inputs)-1 {
        m.focusIndex++
    } else {
        m.focusIndex = 0  // Wrap to first
    }

case "shift+tab", "up":
    if m.focusIndex > 0 {
        m.focusIndex--
    } else {
        m.focusIndex = len(m.inputs) - 1  // Wrap to last
    }
```

### Recommended Pattern for juiscript
**Hybrid**: Step-by-step for complex multi-field forms (Site Create) + Tab navigation for simple forms (settings, filters). Current sitecreate approach is well-suited; prefer it for flows >3 fields.

---

## Architecture Summary

| Component | Built-in | Approach | Use Case |
|-----------|----------|----------|----------|
| Text Input | ✓ Bubbles | charmbracelet/bubbles/textinput | Single/multi fields with Focus/Blur |
| List Select | ✓ Bubbles | charmbracelet/bubbles/list | Large dynamic lists (10+ items) |
| Picker | ✗ Custom | Manual cycle with modulo | Small fixed lists (2-5 items) |
| Confirmation | ✗ Custom | Minimal 2-button model | Yes/No prompts |
| Spinner | ✓ Bubbles | charmbracelet/bubbles/spinner | Loading indicators |
| Toast | ✗ Custom | tea.Tick + state | Temporary messages (auto-dismiss) |

---

## Unresolved Questions

1. **Modal overlays**: How to render confirmation/toast above other content without full-screen takeover? (Answer: lipgloss positioning or separate overlay layer)
2. **Form state serialization**: Best pattern for persisting form draft state across screen transitions?
3. **Accessibility**: ARIA-equivalent patterns for TUI forms beyond color coding?
