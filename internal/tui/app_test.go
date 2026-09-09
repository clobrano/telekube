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
