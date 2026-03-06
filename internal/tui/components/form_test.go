package components

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jhin1m/juiscript/internal/tui/theme"
)

func newTestForm(fields []FormField) *FormModel {
	t := theme.New()
	return NewForm(t, "Test Form", fields)
}

func sendKey(m *FormModel, key string) (*FormModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
}

func sendSpecialKey(m *FormModel, keyType tea.KeyType) (*FormModel, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: keyType})
}

func TestFormModel_TextInput(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText},
	})

	// Type "hello"
	for _, ch := range "hello" {
		form, _ = sendKey(form, string(ch))
	}

	// Press enter to confirm field
	form, _ = sendSpecialKey(form, tea.KeyEnter)

	if form.values["name"] != "hello" {
		t.Errorf("expected name=hello, got %q", form.values["name"])
	}
	// Should be on confirm step now
	if form.step != 1 {
		t.Errorf("expected step=1 (confirm), got %d", form.step)
	}
}

func TestFormModel_TextInput_Backspace(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText},
	})

	for _, ch := range "hello" {
		form, _ = sendKey(form, string(ch))
	}
	form, _ = sendSpecialKey(form, tea.KeyBackspace)

	if form.input != "hell" {
		t.Errorf("expected input=hell after backspace, got %q", form.input)
	}
}

func TestFormModel_SelectCycle(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "color", Label: "Color", Type: FieldSelect, Options: []string{"red", "green", "blue"}, Default: "red"},
	})

	// Tab cycles forward: red -> green
	form, _ = sendSpecialKey(form, tea.KeyTab)
	idx := form.selectIdx["color"]
	if idx != 1 {
		t.Errorf("expected selectIdx=1 (green), got %d", idx)
	}

	// Tab again: green -> blue
	form, _ = sendSpecialKey(form, tea.KeyTab)
	idx = form.selectIdx["color"]
	if idx != 2 {
		t.Errorf("expected selectIdx=2 (blue), got %d", idx)
	}

	// Tab wraps: blue -> red
	form, _ = sendSpecialKey(form, tea.KeyTab)
	idx = form.selectIdx["color"]
	if idx != 0 {
		t.Errorf("expected selectIdx=0 (red, wrap), got %d", idx)
	}
}

func TestFormModel_SelectCycleBackward(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "color", Label: "Color", Type: FieldSelect, Options: []string{"red", "green", "blue"}, Default: "red"},
	})

	// Shift+tab cycles backward: red -> blue (wraps)
	form, _ = sendSpecialKey(form, tea.KeyShiftTab)
	idx := form.selectIdx["color"]
	if idx != 2 {
		t.Errorf("expected selectIdx=2 (blue, backward wrap), got %d", idx)
	}
}

func TestFormModel_ConfirmToggle(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "agree", Label: "Agree", Type: FieldConfirm, Default: "yes"},
	})

	if !form.confirmOn["agree"] {
		t.Error("expected confirmOn[agree]=true from default=yes")
	}

	// Tab toggles
	form, _ = sendSpecialKey(form, tea.KeyTab)
	if form.confirmOn["agree"] {
		t.Error("expected confirmOn[agree]=false after toggle")
	}

	form, _ = sendSpecialKey(form, tea.KeyTab)
	if !form.confirmOn["agree"] {
		t.Error("expected confirmOn[agree]=true after second toggle")
	}
}

func TestFormModel_ValidationError(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "email", Label: "Email", Type: FieldText, Validate: func(s string) error {
			if s == "" {
				return errors.New("required")
			}
			return nil
		}},
	})

	// Press enter with empty input -> validation error
	form, _ = sendSpecialKey(form, tea.KeyEnter)
	if form.err == nil {
		t.Error("expected validation error for empty input")
	}
	if form.step != 0 {
		t.Errorf("expected step=0 (blocked), got %d", form.step)
	}

	// Type valid input and retry
	for _, ch := range "test@mail.com" {
		form, _ = sendKey(form, string(ch))
	}
	form, _ = sendSpecialKey(form, tea.KeyEnter)
	if form.err != nil {
		t.Errorf("expected no error, got %v", form.err)
	}
	if form.step != 1 {
		t.Errorf("expected step=1 after valid input, got %d", form.step)
	}
}

func TestFormModel_Submit(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText},
		{Key: "color", Label: "Color", Type: FieldSelect, Options: []string{"red", "green"}, Default: "red"},
	})

	// Fill name
	for _, ch := range "alice" {
		form, _ = sendKey(form, string(ch))
	}
	form, _ = sendSpecialKey(form, tea.KeyEnter)

	// Confirm select (default=red)
	form, _ = sendSpecialKey(form, tea.KeyEnter)

	// Now on confirm step
	if form.step != 2 {
		t.Fatalf("expected step=2 (confirm), got %d", form.step)
	}

	// Press enter to submit
	var cmd tea.Cmd
	form, cmd = sendSpecialKey(form, tea.KeyEnter)
	if cmd == nil {
		t.Fatal("expected submit command")
	}

	msg := cmd()
	submitMsg, ok := msg.(FormSubmitMsg)
	if !ok {
		t.Fatalf("expected FormSubmitMsg, got %T", msg)
	}
	if submitMsg.Values["name"] != "alice" {
		t.Errorf("expected name=alice, got %q", submitMsg.Values["name"])
	}
	if submitMsg.Values["color"] != "red" {
		t.Errorf("expected color=red, got %q", submitMsg.Values["color"])
	}
}

func TestFormModel_Cancel(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText},
	})

	_, cmd := sendSpecialKey(form, tea.KeyEscape)
	if cmd == nil {
		t.Fatal("expected cancel command")
	}

	msg := cmd()
	if _, ok := msg.(FormCancelMsg); !ok {
		t.Fatalf("expected FormCancelMsg, got %T", msg)
	}
}

func TestFormModel_Reset(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText, Default: ""},
		{Key: "color", Label: "Color", Type: FieldSelect, Options: []string{"red", "green"}, Default: "red"},
	})

	// Fill and advance
	for _, ch := range "alice" {
		form, _ = sendKey(form, string(ch))
	}
	form, _ = sendSpecialKey(form, tea.KeyEnter)

	form.Reset()

	if form.step != 0 {
		t.Errorf("expected step=0 after reset, got %d", form.step)
	}
	if form.input != "" {
		t.Errorf("expected empty input after reset, got %q", form.input)
	}
	if form.err != nil {
		t.Errorf("expected nil error after reset, got %v", form.err)
	}
}

func TestFormModel_Active(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText},
	})

	if !form.Active() {
		t.Error("expected Active()=true at start")
	}

	// Fill and enter field
	for _, ch := range "test" {
		form, _ = sendKey(form, string(ch))
	}
	form, _ = sendSpecialKey(form, tea.KeyEnter)

	// On confirm step - still active
	if !form.Active() {
		t.Error("expected Active()=true on confirm step")
	}

	// Submit -> Active() should return false
	form, _ = sendSpecialKey(form, tea.KeyEnter)
	if form.Active() {
		t.Error("expected Active()=false after submit")
	}
}

func TestFormModel_View(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "name", Label: "Name", Type: FieldText, Placeholder: "enter name"},
	})

	view := form.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestFormModel_DefaultValues(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "color", Label: "Color", Type: FieldSelect, Options: []string{"red", "green", "blue"}, Default: "green"},
	})

	// Default should set selectIdx to 1 (green)
	if form.selectIdx["color"] != 1 {
		t.Errorf("expected selectIdx=1 for default=green, got %d", form.selectIdx["color"])
	}

	// Confirm default without cycling
	form, _ = sendSpecialKey(form, tea.KeyEnter)
	if form.values["color"] != "green" {
		t.Errorf("expected color=green, got %q", form.values["color"])
	}
}

func TestFormModel_ConfirmFieldSubmit(t *testing.T) {
	form := newTestForm([]FormField{
		{Key: "agree", Label: "Agree", Type: FieldConfirm, Default: "no"},
	})

	// Toggle to yes
	form, _ = sendSpecialKey(form, tea.KeyTab)
	// Confirm
	form, _ = sendSpecialKey(form, tea.KeyEnter)

	if form.values["agree"] != "yes" {
		t.Errorf("expected agree=yes, got %q", form.values["agree"])
	}
}
