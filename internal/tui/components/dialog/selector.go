package dialog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/clobrano/telekube/internal/tui/components/search"
)

// SelectorResult represents the result of a selector dialog
type SelectorResult int

const (
	SelectorPending SelectorResult = iota
	SelectorSelected
	SelectorCancelled
)

// SelectorModel represents a selection dialog
type SelectorModel struct {
	title         string
	allItems      []string // original unfiltered items
	items         []string // currently visible (filtered) items
	cursor        int
	result        SelectorResult
	width         int
	height        int
	styles        *SelectorStyles
	actionID      string
	filterInput   textinput.Model
	filterActive  bool
	filterPattern string
}

// SelectorStyles defines the styles for the selector dialog
type SelectorStyles struct {
	Dialog   lipgloss.Style
	Title    lipgloss.Style
	Item     lipgloss.Style
	Selected lipgloss.Style
	Cursor   lipgloss.Style
}

// DefaultSelectorStyles returns the default selector styles
func DefaultSelectorStyles() *SelectorStyles {
	return &SelectorStyles{
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")),
		Cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
	}
}

// NewSelector creates a new selector dialog
func NewSelector(title string, items []string) *SelectorModel {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	ti.CharLimit = 64

	return &SelectorModel{
		title:       title,
		allItems:    items,
		items:       items,
		cursor:      0,
		result:      SelectorPending,
		styles:      DefaultSelectorStyles(),
		filterInput: ti,
	}
}

// WithActionID sets an identifier for the action
func (m *SelectorModel) WithActionID(id string) *SelectorModel {
	m.actionID = id
	return m
}

// SetSize sets the dialog dimensions
func (m *SelectorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Result returns the current result
func (m *SelectorModel) Result() SelectorResult {
	return m.result
}

// SelectedItem returns the selected item
func (m *SelectorModel) SelectedItem() string {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor]
	}
	return ""
}

// SelectedIndex returns the selected index
func (m *SelectorModel) SelectedIndex() int {
	return m.cursor
}

// ActionID returns the action identifier
func (m *SelectorModel) ActionID() string {
	return m.actionID
}

// Reset resets the dialog to pending state
func (m *SelectorModel) Reset() {
	m.result = SelectorPending
	m.cursor = 0
	m.filterActive = false
	m.filterPattern = ""
	m.filterInput.SetValue("")
	m.filterInput.Blur()
	m.items = m.allItems
}

// applyFilter filters allItems using fuzzy matching and updates items
func (m *SelectorModel) applyFilter() {
	pattern := m.filterInput.Value()
	m.filterPattern = pattern

	if pattern == "" {
		m.items = m.allItems
		m.cursor = 0
		return
	}

	type scored struct {
		item  string
		score int
	}
	var matches []scored
	for _, item := range m.allItems {
		if ok, s := search.FuzzyMatch(pattern, item); ok {
			matches = append(matches, scored{item, s})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].score > matches[j].score
	})

	m.items = make([]string, len(matches))
	for i, s := range matches {
		m.items[i] = s.item
	}
	m.cursor = 0
}

// Update handles key messages
func (m *SelectorModel) Update(msg tea.Msg) (*SelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "enter":
				m.filterActive = false
				m.filterInput.Blur()
				return m, nil
			case "esc":
				m.filterActive = false
				m.filterInput.Blur()
				m.filterInput.SetValue("")
				m.filterPattern = ""
				m.items = m.allItems
				m.cursor = 0
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				m.applyFilter()
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.cursor = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if len(m.items) > 0 {
				m.result = SelectorSelected
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.filterPattern != "" {
				// Clear filter first
				m.filterPattern = ""
				m.filterInput.SetValue("")
				m.items = m.allItems
				m.cursor = 0
			} else {
				m.result = SelectorCancelled
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("/"))):
			m.filterActive = true
			m.filterInput.Focus()
		}
	}
	return m, nil
}

// View renders the selector dialog
func (m *SelectorModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render(m.title))
	b.WriteString("\n\n")

	// Items
	maxVisible := 10
	if m.height > 0 {
		maxVisible = m.height - 10 // Account for dialog chrome + filter line
		if maxVisible < 3 {
			maxVisible = 3
		}
	}

	start := 0
	if m.cursor >= maxVisible {
		start = m.cursor - maxVisible + 1
	}

	end := start + maxVisible
	if end > len(m.items) {
		end = len(m.items)
	}

	if len(m.items) == 0 {
		noMatch := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
		b.WriteString(noMatch.Render("  No matches"))
	}
	for i := start; i < end; i++ {
		item := m.items[i]
		if i == m.cursor {
			b.WriteString(m.styles.Cursor.Render("> "))
			b.WriteString(m.styles.Selected.Render(item))
		} else {
			b.WriteString("  ")
			b.WriteString(m.styles.Item.Render(item))
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicators
	if len(m.items) > maxVisible {
		b.WriteString("\n\n")
		if start > 0 {
			b.WriteString("↑ ")
		} else {
			b.WriteString("  ")
		}
		b.WriteString(fmt.Sprintf("%d/%d", m.cursor+1, len(m.items)))
		if end < len(m.items) {
			b.WriteString(" ↓")
		}
	}

	// Filter input
	if m.filterActive {
		b.WriteString("\n\n")
		b.WriteString(m.filterInput.View())
	} else if m.filterPattern != "" {
		b.WriteString("\n\n")
		filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
		b.WriteString(filterStyle.Render("/ " + m.filterPattern))
	}

	b.WriteString("\n\n")
	if m.filterActive {
		b.WriteString("[Enter] confirm filter  [Esc] clear filter")
	} else {
		hints := "[Enter] select  [/] filter  [Esc] "
		if m.filterPattern != "" {
			hints += "clear filter"
		} else {
			hints += "cancel"
		}
		b.WriteString(hints)
	}

	content := m.styles.Dialog.Render(b.String())

	// Center the dialog
	if m.width > 0 && m.height > 0 {
		dialogWidth := lipgloss.Width(content)
		dialogHeight := lipgloss.Height(content)

		horizontalPadding := (m.width - dialogWidth) / 2
		verticalPadding := (m.height - dialogHeight) / 2

		if horizontalPadding < 0 {
			horizontalPadding = 0
		}
		if verticalPadding < 0 {
			verticalPadding = 0
		}

		var centered strings.Builder
		for i := 0; i < verticalPadding; i++ {
			centered.WriteString(strings.Repeat(" ", m.width))
			centered.WriteString("\n")
		}

		lines := strings.Split(content, "\n")
		for _, line := range lines {
			centered.WriteString(strings.Repeat(" ", horizontalPadding))
			centered.WriteString(line)
			centered.WriteString("\n")
		}

		return centered.String()
	}

	return content
}

// String implements fmt.Stringer
func (m *SelectorModel) String() string {
	return fmt.Sprintf("Selector{title=%q, items=%d, cursor=%d}", m.title, len(m.items), m.cursor)
}
