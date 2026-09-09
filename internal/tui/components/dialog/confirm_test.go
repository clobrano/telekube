package dialog

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewConfirm(t *testing.T) {
	c := NewConfirm("Test Title", "Test message")

	if c.Result() != ConfirmPending {
		t.Errorf("Initial result = %v, want ConfirmPending", c.Result())
	}
}

func TestConfirmYes(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")

	// Press 'y'
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if c.Result() != ConfirmYes {
		t.Errorf("Result after 'y' = %v, want ConfirmYes", c.Result())
	}
}

func TestConfirmNo(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")

	// Press 'n'
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if c.Result() != ConfirmNo {
		t.Errorf("Result after 'n' = %v, want ConfirmNo", c.Result())
	}
}

func TestConfirmEsc(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")

	// Press Esc
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if c.Result() != ConfirmNo {
		t.Errorf("Result after Esc = %v, want ConfirmNo", c.Result())
	}
}

func TestConfirmQDoesNotCancel(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")

	// Press 'q' - must NOT cancel the dialog. 'q' is reserved for quitting the
	// application; Esc is the uniform cancel/back key.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if c.Result() != ConfirmPending {
		t.Errorf("Result after 'q' = %v, want ConfirmPending", c.Result())
	}
}

func TestConfirmEnter(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")

	// Press Enter
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if c.Result() != ConfirmYes {
		t.Errorf("Result after Enter = %v, want ConfirmYes", c.Result())
	}
}

func TestConfirmReset(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	c.Reset()

	if c.Result() != ConfirmPending {
		t.Errorf("Result after Reset = %v, want ConfirmPending", c.Result())
	}
}

func TestConfirmWithActionID(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?").WithActionID("delete-pod")

	if c.ActionID() != "delete-pod" {
		t.Errorf("ActionID() = %q, want %q", c.ActionID(), "delete-pod")
	}
}

func TestConfirmView(t *testing.T) {
	c := NewConfirm("Delete?", "Are you sure?")
	view := c.View()

	if !strings.Contains(view, "Delete?") {
		t.Error("View should contain title")
	}

	if !strings.Contains(view, "Are you sure?") {
		t.Error("View should contain message")
	}

	if !strings.Contains(view, "[Y]es") {
		t.Error("View should contain Yes button")
	}

	if !strings.Contains(view, "[N]o") {
		t.Error("View should contain No button")
	}
}

func TestConfirmString(t *testing.T) {
	c := NewConfirm("Test", "Message")
	s := c.String()

	if !strings.Contains(s, "Confirm") {
		t.Errorf("String() = %q, should contain 'Confirm'", s)
	}
}
