package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// FieldType identifies the kind of form field.
type FieldType int

const (
	FieldText    FieldType = iota // free text input
	FieldSelect                   // pick from options list
	FieldConfirm                  // yes/no toggle
)

// FormField defines a single form field.
type FormField struct {
	Key         string             // unique key for value lookup
	Label       string             // display label
	Type        FieldType          // field type
	Options     []string           // for FieldSelect
	Default     string             // initial value
	Placeholder string             // for FieldText
	Validate    func(string) error // optional validator (nil = always valid)
}

// FormSubmitMsg carries completed form data back to the parent screen.
type FormSubmitMsg struct {
	Values map[string]string
}

// FormCancelMsg signals the user cancelled the form.
type FormCancelMsg struct{}

// FormModel is a reusable step-by-step form component.
// Screens embed this and toggle formActive to show/hide the form.
type FormModel struct {
	theme     *theme.Theme
	title     string
	fields    []FormField
	step      int               // current field index; len(fields)=confirm; len(fields)+1=done
	values    map[string]string // field key -> confirmed value
	input     string            // current text buffer for FieldText
	selectIdx map[string]int    // current selection index per field key
	confirmOn map[string]bool   // per-field confirm toggle state
	err       error
}

// NewForm creates a form with the given title and fields.
// Default values from FormField.Default are pre-set.
func NewForm(t *theme.Theme, title string, fields []FormField) *FormModel {
	m := &FormModel{
		theme:     t,
		title:     title,
		fields:    fields,
		values:    make(map[string]string),
		selectIdx: make(map[string]int),
		confirmOn: make(map[string]bool),
	}
	m.applyDefaults()
	return m
}

// applyDefaults sets initial values from field definitions.
func (m *FormModel) applyDefaults() {
	for _, f := range m.fields {
		if f.Default != "" {
			m.values[f.Key] = f.Default
		}
		if f.Type == FieldSelect && f.Default != "" {
			for i, opt := range f.Options {
				if opt == f.Default {
					m.selectIdx[f.Key] = i
					break
				}
			}
		}
		if f.Type == FieldConfirm {
			m.confirmOn[f.Key] = f.Default == "yes"
		}
	}
}

// Active returns true if the form is still in progress (not yet submitted/cancelled).
func (m *FormModel) Active() bool {
	return m.step <= len(m.fields)
}

// Values returns a copy of the current form values.
func (m *FormModel) Values() map[string]string {
	cp := make(map[string]string, len(m.values))
	for k, v := range m.values {
		cp[k] = v
	}
	return cp
}

// Reset clears all state for reuse.
func (m *FormModel) Reset() {
	m.step = 0
	m.values = make(map[string]string)
	m.selectIdx = make(map[string]int)
	m.confirmOn = make(map[string]bool)
	m.input = ""
	m.err = nil
	m.applyDefaults()
}

// Update handles key messages and returns commands.
// Returns FormSubmitMsg on confirm, FormCancelMsg on esc.
func (m *FormModel) Update(msg tea.Msg) (*FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return FormCancelMsg{} }

		case "enter":
			return m.handleEnter()

		case "tab", "down":
			m.handleCycleForward()

		case "shift+tab", "up":
			m.handleCycleBackward()

		case "j":
			// Only navigate when NOT on a text field; otherwise treat as text input
			if m.step < len(m.fields) && m.fields[m.step].Type == FieldText {
				m.input += msg.String()
			} else {
				m.handleCycleForward()
			}

		case "k":
			if m.step < len(m.fields) && m.fields[m.step].Type == FieldText {
				m.input += msg.String()
			} else {
				m.handleCycleBackward()
			}

		case "backspace":
			if m.step < len(m.fields) && m.fields[m.step].Type == FieldText && len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}

		default:
			// Text input: append single printable chars
			if m.step < len(m.fields) && m.fields[m.step].Type == FieldText && len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}
	}
	return m, nil
}

func (m *FormModel) handleEnter() (*FormModel, tea.Cmd) {
	// Confirm step (after all fields) - submit
	if m.step == len(m.fields) {
		vals := m.Values()
		m.step++ // mark as done so Active() returns false
		return m, func() tea.Msg { return FormSubmitMsg{Values: vals} }
	}

	f := m.fields[m.step]
	switch f.Type {
	case FieldText:
		val := m.input
		if f.Validate != nil {
			if err := f.Validate(val); err != nil {
				m.err = err
				return m, nil
			}
		}
		m.values[f.Key] = val
		m.input = ""
		m.err = nil
		m.step++

	case FieldSelect:
		idx := m.selectIdx[f.Key]
		if len(f.Options) > 0 {
			m.values[f.Key] = f.Options[idx]
		}
		m.err = nil
		m.step++

	case FieldConfirm:
		if m.confirmOn[f.Key] {
			m.values[f.Key] = "yes"
		} else {
			m.values[f.Key] = "no"
		}
		m.err = nil
		m.step++
	}

	return m, nil
}

func (m *FormModel) handleCycleForward() {
	if m.step >= len(m.fields) {
		return
	}
	f := m.fields[m.step]
	switch f.Type {
	case FieldSelect:
		if len(f.Options) > 0 {
			m.selectIdx[f.Key] = (m.selectIdx[f.Key] + 1) % len(f.Options)
		}
	case FieldConfirm:
		m.confirmOn[f.Key] = !m.confirmOn[f.Key]
	}
}

func (m *FormModel) handleCycleBackward() {
	if m.step >= len(m.fields) {
		return
	}
	f := m.fields[m.step]
	switch f.Type {
	case FieldSelect:
		if len(f.Options) > 0 {
			idx := (m.selectIdx[f.Key] - 1 + len(f.Options)) % len(f.Options)
			m.selectIdx[f.Key] = idx
		}
	case FieldConfirm:
		m.confirmOn[f.Key] = !m.confirmOn[f.Key]
	}
}

// View renders the form in its current state.
func (m *FormModel) View() string {
	title := m.theme.Title.Render(m.title)

	var fields string
	for i, f := range m.fields {
		if i > m.step {
			break // only show completed + current field
		}
		active := i == m.step
		label := m.fieldLabel(f.Label+":", active)

		var val string
		switch {
		case i < m.step:
			val = m.values[f.Key]
		case f.Type == FieldText:
			val = m.input
			if f.Placeholder != "" && val == "" {
				val = m.theme.Inactive.Render(f.Placeholder)
			}
			if active {
				val += "_"
			}
		case f.Type == FieldSelect:
			idx := m.selectIdx[f.Key]
			if len(f.Options) > 0 {
				val = f.Options[idx]
			}
		case f.Type == FieldConfirm:
			if m.confirmOn[f.Key] {
				val = "yes"
			} else {
				val = "no"
			}
		}

		fields += fmt.Sprintf("  %s %s\n", label, val)
	}

	// Confirm step - show summary prompt
	if m.step == len(m.fields) {
		confirm := m.theme.OkText.Render("\n  Press Enter to confirm, Esc to cancel")
		fields += confirm
	}

	// Error display
	var errLine string
	if m.err != nil {
		errLine = "\n" + m.theme.ErrorText.Render(fmt.Sprintf("  Error: %v", m.err))
	}

	help := m.theme.HelpDesc.Render("\n  enter:next  tab/↑↓:cycle options  esc:cancel")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", fields, errLine, help)
}

func (m *FormModel) fieldLabel(label string, active bool) string {
	if active {
		return m.theme.Active.Render(label)
	}
	return m.theme.Subtitle.Render(label)
}
