package dialog

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSelector(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	s := NewSelector("Select Item", items)

	if s.Result() != SelectorPending {
		t.Errorf("Initial result = %v, want SelectorPending", s.Result())
	}

	if s.SelectedIndex() != 0 {
		t.Errorf("Initial SelectedIndex() = %d, want 0", s.SelectedIndex())
	}

	if s.SelectedItem() != "item1" {
		t.Errorf("Initial SelectedItem() = %q, want %q", s.SelectedItem(), "item1")
	}
}

func TestSelectorNavigation(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	s := NewSelector("Select", items)

	// Move down
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.SelectedIndex() != 1 {
		t.Errorf("SelectedIndex() after j = %d, want 1", s.SelectedIndex())
	}

	// Move down again
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyDown})
	if s.SelectedIndex() != 2 {
		t.Errorf("SelectedIndex() after down = %d, want 2", s.SelectedIndex())
	}

	// Try to move past end
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.SelectedIndex() != 2 {
		t.Errorf("SelectedIndex() should stay at 2, got %d", s.SelectedIndex())
	}

	// Move up
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if s.SelectedIndex() != 1 {
		t.Errorf("SelectedIndex() after k = %d, want 1", s.SelectedIndex())
	}
}

func TestSelectorTopBottom(t *testing.T) {
	items := []string{"item1", "item2", "item3", "item4", "item5"}
	s := NewSelector("Select", items)

	// Go to bottom
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if s.SelectedIndex() != 4 {
		t.Errorf("SelectedIndex() after G = %d, want 4", s.SelectedIndex())
	}

	// Go to top
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if s.SelectedIndex() != 0 {
		t.Errorf("SelectedIndex() after g = %d, want 0", s.SelectedIndex())
	}
}

func TestSelectorSelect(t *testing.T) {
	items := []string{"item1", "item2", "item3"}
	s := NewSelector("Select", items)

	// Move to second item and select
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if s.Result() != SelectorSelected {
		t.Errorf("Result after Enter = %v, want SelectorSelected", s.Result())
	}

	if s.SelectedItem() != "item2" {
		t.Errorf("SelectedItem() = %q, want %q", s.SelectedItem(), "item2")
	}
}

func TestSelectorCancel(t *testing.T) {
	items := []string{"item1", "item2"}
	s := NewSelector("Select", items)

	// Press Esc
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if s.Result() != SelectorCancelled {
		t.Errorf("Result after Esc = %v, want SelectorCancelled", s.Result())
	}
}

func TestSelectorQCancels(t *testing.T) {
	items := []string{"item1", "item2"}
	s := NewSelector("Select", items)

	// Press 'q' - cancels the selector. Safe (never quits the app from a dialog)
	// and works where a lone Esc is delayed (e.g. tmux).
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if s.Result() != SelectorCancelled {
		t.Errorf("Result after 'q' = %v, want SelectorCancelled", s.Result())
	}
}

func TestSelectorReset(t *testing.T) {
	items := []string{"item1", "item2"}
	s := NewSelector("Select", items)

	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	s.Reset()

	if s.Result() != SelectorPending {
		t.Errorf("Result after Reset = %v, want SelectorPending", s.Result())
	}

	if s.SelectedIndex() != 0 {
		t.Errorf("SelectedIndex() after Reset = %d, want 0", s.SelectedIndex())
	}
}

func TestSelectorView(t *testing.T) {
	items := []string{"option1", "option2"}
	s := NewSelector("Choose Option", items)
	view := s.View()

	if !strings.Contains(view, "Choose Option") {
		t.Error("View should contain title")
	}

	if !strings.Contains(view, "option1") {
		t.Error("View should contain first item")
	}

	if !strings.Contains(view, "option2") {
		t.Error("View should contain second item")
	}

	if !strings.Contains(view, ">") {
		t.Error("View should contain cursor indicator")
	}
}

func TestSelectorWithActionID(t *testing.T) {
	s := NewSelector("Select", []string{"a", "b"}).WithActionID("context-switch")

	if s.ActionID() != "context-switch" {
		t.Errorf("ActionID() = %q, want %q", s.ActionID(), "context-switch")
	}
}

func TestSelectorString(t *testing.T) {
	s := NewSelector("Test", []string{"a", "b", "c"})
	str := s.String()

	if !strings.Contains(str, "Selector") {
		t.Errorf("String() = %q, should contain 'Selector'", str)
	}

	if !strings.Contains(str, "items=3") {
		t.Errorf("String() = %q, should contain 'items=3'", str)
	}
}

func TestSelectorEmpty(t *testing.T) {
	s := NewSelector("Empty", []string{})

	if s.SelectedItem() != "" {
		t.Errorf("SelectedItem() on empty selector = %q, want empty", s.SelectedItem())
	}
}
