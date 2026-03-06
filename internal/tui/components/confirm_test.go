package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

func newTestConfirm() *ConfirmModel {
	return NewConfirm(theme.New())
}

func TestConfirm_DefaultNo(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete this?")

	if !c.Active() {
		t.Error("expected active after Show")
	}
	if c.selected {
		t.Error("expected default selection = No (false)")
	}
}

func TestConfirm_TabToggle(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete?")

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !c.selected {
		t.Error("expected Yes after tab")
	}

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if c.selected {
		t.Error("expected No after second tab")
	}
}

func TestConfirm_YKey(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete?")

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd == nil {
		t.Fatal("expected command from y key")
	}
	if _, ok := cmd().(ConfirmYesMsg); !ok {
		t.Error("expected ConfirmYesMsg")
	}
	if c.Active() {
		t.Error("expected inactive after confirm")
	}
}

func TestConfirm_NKey(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete?")

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if cmd == nil {
		t.Fatal("expected command from n key")
	}
	if _, ok := cmd().(ConfirmNoMsg); !ok {
		t.Error("expected ConfirmNoMsg")
	}
	if c.Active() {
		t.Error("expected inactive after cancel")
	}
}

func TestConfirm_EscCancels(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete?")

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected command from esc")
	}
	if _, ok := cmd().(ConfirmNoMsg); !ok {
		t.Error("expected ConfirmNoMsg from esc")
	}
}

func TestConfirm_EnterWithNo(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete?")
	// Default is No

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from enter")
	}
	if _, ok := cmd().(ConfirmNoMsg); !ok {
		t.Error("expected ConfirmNoMsg when No selected")
	}
}

func TestConfirm_EnterWithYes(t *testing.T) {
	c := newTestConfirm()
	c.Show("Delete?")

	// Toggle to Yes
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command from enter")
	}
	if _, ok := cmd().(ConfirmYesMsg); !ok {
		t.Error("expected ConfirmYesMsg when Yes selected")
	}
}

func TestConfirm_InactiveIgnoresKeys(t *testing.T) {
	c := newTestConfirm()
	// Not shown - should ignore keys
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if cmd != nil {
		t.Error("expected nil cmd when inactive")
	}
}

func TestConfirm_View(t *testing.T) {
	c := newTestConfirm()
	if c.View() != "" {
		t.Error("expected empty view when inactive")
	}

	c.Show("Drop database?")
	view := c.View()
	if view == "" {
		t.Error("expected non-empty view when active")
	}
}
