package components

import (
	"testing"

	"github.com/jhin1m/juiscript/internal/tui/theme"
)

func newTestToast() *ToastModel {
	return NewToast(theme.New())
}

func TestToast_Show(t *testing.T) {
	toast := newTestToast()
	cmd := toast.Show(ToastSuccess, "Done!")

	if !toast.Visible() {
		t.Error("expected visible after Show")
	}
	if toast.message != "Done!" {
		t.Errorf("expected message=Done!, got %q", toast.message)
	}
	if toast.toastType != ToastSuccess {
		t.Errorf("expected type=ToastSuccess, got %d", toast.toastType)
	}
	if cmd == nil {
		t.Error("expected tick cmd from Show")
	}
}

func TestToast_DismissMatchingID(t *testing.T) {
	toast := newTestToast()
	toast.Show(ToastSuccess, "Done!")
	id := toast.id

	toast, _ = toast.Update(ToastDismissMsg{ID: id})
	if toast.Visible() {
		t.Error("expected hidden after matching dismiss")
	}
}

func TestToast_DismissMismatchID(t *testing.T) {
	toast := newTestToast()
	toast.Show(ToastSuccess, "Done!")

	// Old ID should not dismiss current toast
	toast, _ = toast.Update(ToastDismissMsg{ID: toast.id - 1})
	if !toast.Visible() {
		t.Error("expected still visible with mismatched ID")
	}
}

func TestToast_ManualDismiss(t *testing.T) {
	toast := newTestToast()
	toast.Show(ToastError, "Failed!")
	toast.Dismiss()

	if toast.Visible() {
		t.Error("expected hidden after Dismiss()")
	}
}

func TestToast_ViewVariants(t *testing.T) {
	toast := newTestToast()

	// Hidden
	if toast.View() != "" {
		t.Error("expected empty view when hidden")
	}

	// Success
	toast.Show(ToastSuccess, "Created")
	view := toast.View()
	if view == "" {
		t.Error("expected non-empty success view")
	}

	// Error
	toast.Show(ToastError, "Failed")
	view = toast.View()
	if view == "" {
		t.Error("expected non-empty error view")
	}

	// Warning
	toast.Show(ToastWarning, "Careful")
	view = toast.View()
	if view == "" {
		t.Error("expected non-empty warning view")
	}
}

func TestToast_NewShowReplacesOld(t *testing.T) {
	toast := newTestToast()
	toast.Show(ToastSuccess, "First")
	firstID := toast.id

	toast.Show(ToastError, "Second")
	if toast.message != "Second" {
		t.Errorf("expected message=Second, got %q", toast.message)
	}

	// Old dismiss should not hide new toast
	toast, _ = toast.Update(ToastDismissMsg{ID: firstID})
	if !toast.Visible() {
		t.Error("expected still visible - old ID should not dismiss new toast")
	}
}
