package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/clobrano/telekube/internal/config"
	"github.com/clobrano/telekube/internal/kubectl"
	"github.com/clobrano/telekube/internal/tui/components/detail"
)

func newTestModel() *Model {
	return NewModel(config.DefaultConfig(), kubectl.NewKubectl("kubectl"), "")
}

// TestDetailEscExitsAfterSearch reproduces the regression where a result view
// (e.g. "Delete Result") would not respond to Esc because search state left over
// from a previous detail view was absorbing the key. Esc must reliably return to
// the list in a single press.
func TestDetailEscExitsAfterSearch(t *testing.T) {
	m := newTestModel()

	// View a YAML detail and run a search that leaves a confirmed query.
	m.detail.SetContent("pod-x", "apiVersion: v1\nkind: Pod", detail.FormatYAML)
	m.viewState = ViewDetail
	m.detail.StartSearch()
	for _, r := range "pod" {
		m.detail, _ = m.detail.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.detail, _ = m.detail.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Now show a delete result on the same detail model, as executeDelete does.
	m.detail.SetContent("Delete Result", "Deleted computeinstance 'x'.", detail.FormatTable)
	m.viewState = ViewDetail
	m.loading = true

	// A single Esc must exit back to the list.
	upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = upd.(*Model)

	if m.viewState != ViewList {
		t.Fatalf("viewState after Esc = %v, want ViewList", m.viewState)
	}
}

// TestDetailEscExitsSinglePress verifies a plain detail view exits on the first
// Esc even when a search highlight is active within that same view.
func TestDetailEscExitsSinglePress(t *testing.T) {
	m := newTestModel()
	m.detail.SetContent("pod-x", "line one\nline two", detail.FormatYAML)
	m.viewState = ViewDetail

	// Confirm a search query within this view.
	m.detail.StartSearch()
	for _, r := range "line" {
		m.detail, _ = m.detail.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.detail, _ = m.detail.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !m.detail.HasSearchQuery() {
		t.Fatal("precondition: expected an active search query")
	}

	// One Esc exits; it does not merely clear the highlight and stay.
	upd, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = upd.(*Model)

	if m.viewState != ViewList {
		t.Fatalf("viewState after Esc = %v, want ViewList", m.viewState)
	}
}

// TestDetailQExits verifies q also returns to the list from a detail/result
// view (a reliable exit where a lone Esc is delayed, e.g. tmux) and that it
// does NOT quit the application.
func TestDetailQExits(t *testing.T) {
	m := newTestModel()
	m.detail.SetContent("Delete Result", "Deleted computeinstance 'x'.", detail.FormatTable)
	m.viewState = ViewDetail

	upd, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = upd.(*Model)

	if m.viewState != ViewList {
		t.Fatalf("viewState after q = %v, want ViewList", m.viewState)
	}
	if cmd != nil {
		// tea.Quit is a non-nil command; q must not quit from the detail view.
		if msg := cmd(); msg != nil {
			if _, isQuit := msg.(tea.QuitMsg); isQuit {
				t.Fatal("q quit the application from the detail view; it must only go back")
			}
		}
	}
}

// TestListCapitalQQuits verifies capital Q quits from the list view.
func TestListCapitalQQuits(t *testing.T) {
	m := newTestModel()
	m.viewState = ViewList

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	if cmd == nil {
		t.Fatal("Q in list view returned nil cmd, want tea.Quit")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("Q in list view produced nil msg, want tea.QuitMsg")
	} else if _, isQuit := msg.(tea.QuitMsg); !isQuit {
		t.Fatalf("Q in list view produced %T, want tea.QuitMsg", msg)
	}
}

// TestListLowerQDoesNotQuit verifies lowercase q never quits the application.
func TestListLowerQDoesNotQuit(t *testing.T) {
	m := newTestModel()
	m.viewState = ViewList

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isQuit := msg.(tea.QuitMsg); isQuit {
				t.Fatal("lowercase q quit the application from the list view; only Q must quit")
			}
		}
	}
}

// TestDetailCapitalQQuits verifies capital Q quits directly from a detail view.
func TestDetailCapitalQQuits(t *testing.T) {
	m := newTestModel()
	m.detail.SetContent("pod-x", "apiVersion: v1", detail.FormatYAML)
	m.viewState = ViewDetail

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	if cmd == nil {
		t.Fatal("Q in detail view returned nil cmd, want tea.Quit")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("Q in detail view produced nil msg, want tea.QuitMsg")
	} else if _, isQuit := msg.(tea.QuitMsg); !isQuit {
		t.Fatalf("Q in detail view produced %T, want tea.QuitMsg", msg)
	}
}
