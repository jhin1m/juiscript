package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

// ToastType determines the visual style of the notification.
type ToastType int

const (
	ToastSuccess ToastType = iota
	ToastError
	ToastWarning
)

// Toast auto-dismiss durations: 3s for success, 5s for errors.
var toastDurations = map[ToastType]time.Duration{
	ToastSuccess: 3 * time.Second,
	ToastError:   5 * time.Second,
	ToastWarning: 5 * time.Second,
}

// ToastDismissMsg is sent by tea.Tick to auto-dismiss a toast.
type ToastDismissMsg struct {
	ID int
}

// ShowToastMsg can be sent from anywhere to trigger a toast in App.
type ShowToastMsg struct {
	Type    ToastType
	Message string
}

// ToastModel manages a single auto-dismissing notification.
type ToastModel struct {
	theme     *theme.Theme
	message   string
	toastType ToastType
	visible   bool
	id        int // incremented per Show, used to match dismiss ticks
}

// NewToast creates a new toast notification manager.
func NewToast(t *theme.Theme) *ToastModel {
	return &ToastModel{theme: t}
}

// Show displays a toast and returns a tick cmd for auto-dismiss.
func (t *ToastModel) Show(typ ToastType, message string) tea.Cmd {
	t.id++
	t.toastType = typ
	t.message = message
	t.visible = true

	id := t.id
	duration, ok := toastDurations[typ]
	if !ok {
		duration = 3 * time.Second
	}
	return tea.Tick(duration, func(_ time.Time) tea.Msg {
		return ToastDismissMsg{ID: id}
	})
}

// Update handles dismiss messages. Only dismisses if ID matches current toast.
func (t *ToastModel) Update(msg tea.Msg) (*ToastModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ToastDismissMsg:
		if msg.ID == t.id {
			t.visible = false
		}
	}
	return t, nil
}

// Dismiss hides the toast immediately.
func (t *ToastModel) Dismiss() {
	t.visible = false
}

// Visible returns whether a toast is currently showing.
func (t *ToastModel) Visible() bool {
	return t.visible
}

// View returns the styled toast line, or empty string if hidden.
func (t *ToastModel) View() string {
	if !t.visible {
		return ""
	}

	var prefix string
	switch t.toastType {
	case ToastSuccess:
		prefix = t.theme.OkText.Render("  SUCCESS: ")
	case ToastError:
		prefix = t.theme.ErrorText.Render("  ERROR: ")
	case ToastWarning:
		prefix = t.theme.WarnText.Render("  WARNING: ")
	}

	return fmt.Sprintf("%s%s", prefix, t.message)
}
