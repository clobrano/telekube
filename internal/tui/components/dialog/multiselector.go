package dialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelectorResult represents the result of a multi-selector dialog
type MultiSelectorResult int

const (
	MultiSelectorPending MultiSelectorResult = iota
	MultiSelectorConfirmed
	MultiSelectorCancelled
)

// MultiSelectorItem represents a selectable item with a label and a copyable value
type MultiSelectorItem struct {
	Label string
	Value string
}

// MultiSelectorModel is a floating dialog that allows toggling multiple items
// with [space] and confirming the selection with [enter].
type MultiSelectorModel struct {
	title    string
	items    []MultiSelectorItem
	selected []bool
	cursor   int
	result   MultiSelectorResult
	width    int
	height   int
	styles   *SelectorStyles
}

// NewMultiSelector creates a new multi-select dialog.
func NewMultiSelector(title string, items []MultiSelectorItem) *MultiSelectorModel {
	return &MultiSelectorModel{
		title:    title,
		items:    items,
		selected: make([]bool, len(items)),
		result:   MultiSelectorPending,
		styles:   DefaultSelectorStyles(),
	}
}

// SetSize sets the dialog dimensions.
func (m *MultiSelectorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Result returns the current result state.
func (m *MultiSelectorModel) Result() MultiSelectorResult {
	return m.result
}

// SelectedValues returns the copy values of all toggled items.
func (m *MultiSelectorModel) SelectedValues() []string {
	var out []string
	for i, sel := range m.selected {
		if sel {
			out = append(out, m.items[i].Value)
		}
	}
	return out
}

// Update handles keyboard messages.
func (m *MultiSelectorModel) Update(msg tea.Msg) (*MultiSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
			if len(m.items) > 0 {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			// Confirm with whatever is currently toggled; if nothing toggled,
			// treat the cursor item as the selection.
			anySelected := false
			for _, s := range m.selected {
				if s {
					anySelected = true
					break
				}
			}
			if !anySelected && len(m.items) > 0 {
				m.selected[m.cursor] = true
			}
			m.result = MultiSelectorConfirmed
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			m.result = MultiSelectorCancelled
		}
	}
	return m, nil
}

// View renders the dialog centered on the terminal.
func (m *MultiSelectorModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.title))
	b.WriteString("\n\n")

	maxVisible := 10
	if m.height > 0 {
		maxVisible = m.height - 10
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

	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	uncheckedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i := start; i < end; i++ {
		item := m.items[i]
		var checkbox string
		if m.selected[i] {
			checkbox = checkStyle.Render("[x]")
		} else {
			checkbox = uncheckedStyle.Render("[ ]")
		}

		var line string
		if i == m.cursor {
			line = fmt.Sprintf("%s %s %s", m.styles.Cursor.Render(">"), checkbox, m.styles.Selected.Render(item.Label))
		} else {
			line = fmt.Sprintf("  %s %s", checkbox, m.styles.Item.Render(item.Label))
		}
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	b.WriteString("[space] toggle  [enter] copy  [esc] cancel")

	content := m.styles.Dialog.Render(b.String())

	if m.width > 0 && m.height > 0 {
		dialogWidth := lipgloss.Width(content)
		dialogHeight := lipgloss.Height(content)

		hPad := (m.width - dialogWidth) / 2
		vPad := (m.height - dialogHeight) / 2
		if hPad < 0 {
			hPad = 0
		}
		if vPad < 0 {
			vPad = 0
		}

		var centered strings.Builder
		for i := 0; i < vPad; i++ {
			centered.WriteString(strings.Repeat(" ", m.width))
			centered.WriteString("\n")
		}
		for _, line := range strings.Split(content, "\n") {
			centered.WriteString(strings.Repeat(" ", hPad))
			centered.WriteString(line)
			centered.WriteString("\n")
		}
		return centered.String()
	}

	return content
}
