package dialog

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestMultiSelector() *MultiSelectorModel {
	return NewMultiSelector("Copy", []MultiSelectorItem{
		{Label: "NAME  pod-1", Value: "pod-1"},
		{Label: "NS    default", Value: "default"},
	})
}

func TestMultiSelectorEscCancels(t *testing.T) {
	m := newTestMultiSelector()

	// Esc is the uniform cancel/back key.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if m.Result() != MultiSelectorCancelled {
		t.Errorf("Result after Esc = %v, want MultiSelectorCancelled", m.Result())
	}
}

func TestMultiSelectorQDoesNotCancel(t *testing.T) {
	m := newTestMultiSelector()

	// Press 'q' - must NOT cancel. 'q' is reserved for quitting the application.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if m.Result() != MultiSelectorPending {
		t.Errorf("Result after 'q' = %v, want MultiSelectorPending", m.Result())
	}
}
