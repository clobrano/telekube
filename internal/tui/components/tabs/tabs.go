package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Model represents the tab bar component
type Model struct {
	tabs        []string
	activeIndex int
	width       int
	styles      *Styles
}

// Styles defines the styles for the tab bar component
type Styles struct {
	Container lipgloss.Style
	Active    lipgloss.Style
	Inactive  lipgloss.Style
	Number    lipgloss.Style
}

// DefaultStyles returns the default tab bar styles
func DefaultStyles() *Styles {
	return &Styles{
		Container: lipgloss.NewStyle().
			Background(lipgloss.Color("236")),
		Active: lipgloss.NewStyle().
			Background(lipgloss.Color("39")).
			Foreground(lipgloss.Color("0")).
			Padding(0, 2).
			Bold(true),
		Inactive: lipgloss.NewStyle().
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 2),
		Number: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			PaddingRight(1),
	}
}

// New creates a new tab bar component
func New(tabs []string, activeIndex int) *Model {
	if activeIndex < 0 {
		activeIndex = 0
	}
	if activeIndex >= len(tabs) && len(tabs) > 0 {
		activeIndex = len(tabs) - 1
	}

	return &Model{
		tabs:        tabs,
		activeIndex: activeIndex,
		styles:      DefaultStyles(),
	}
}

// SetActive sets the active tab index
func (m *Model) SetActive(index int) {
	if index >= 0 && index < len(m.tabs) {
		m.activeIndex = index
	}
}

// Active returns the current active tab index
func (m *Model) Active() int {
	return m.activeIndex
}

// ActiveName returns the name of the active tab
func (m *Model) ActiveName() string {
	if m.activeIndex >= 0 && m.activeIndex < len(m.tabs) {
		return m.tabs[m.activeIndex]
	}
	return ""
}

// Next moves to the next tab
func (m *Model) Next() {
	if len(m.tabs) == 0 {
		return
	}
	m.activeIndex = (m.activeIndex + 1) % len(m.tabs)
}

// Previous moves to the previous tab
func (m *Model) Previous() {
	if len(m.tabs) == 0 {
		return
	}
	m.activeIndex--
	if m.activeIndex < 0 {
		m.activeIndex = len(m.tabs) - 1
	}
}

// SetWidth sets the width for the tab bar
func (m *Model) SetWidth(width int) {
	m.width = width
}

// Count returns the number of tabs
func (m *Model) Count() int {
	return len(m.tabs)
}

// AddTab appends a new tab with the given name
func (m *Model) AddTab(name string) {
	m.tabs = append(m.tabs, name)
}

// View renders the tab bar
func (m *Model) View() string {
	if len(m.tabs) == 0 {
		return ""
	}

	var tabViews []string

	for i, tab := range m.tabs {
		// Tab number (1-9, only show for first 9 tabs)
		var numStr string
		if i < 9 {
			numStr = m.styles.Number.Render(fmt.Sprintf("%d", i+1))
		}

		// Tab content
		var tabContent string
		if i == m.activeIndex {
			tabContent = m.styles.Active.Render(tab)
		} else {
			tabContent = m.styles.Inactive.Render(tab)
		}

		tabViews = append(tabViews, numStr+tabContent)
	}

	content := strings.Join(tabViews, " ")

	// If width is set, apply container styling
	if m.width > 0 {
		return m.styles.Container.Width(m.width).Render(content)
	}

	return m.styles.Container.Render(content)
}

// String implements fmt.Stringer
func (m *Model) String() string {
	return fmt.Sprintf("Tabs{active=%d, tabs=%v}", m.activeIndex, m.tabs)
}
