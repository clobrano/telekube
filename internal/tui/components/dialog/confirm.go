package dialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmResult represents the result of a confirmation dialog
type ConfirmResult int

const (
	ConfirmPending ConfirmResult = iota
	ConfirmYes
	ConfirmNo
)

// ConfirmModel represents a confirmation dialog
type ConfirmModel struct {
	title    string
	message  string
	result   ConfirmResult
	width    int
	height   int
	styles   *ConfirmStyles
	actionID string // Optional identifier for the action being confirmed
}

// ConfirmStyles defines the styles for the confirmation dialog
type ConfirmStyles struct {
	Dialog     lipgloss.Style
	Title      lipgloss.Style
	Message    lipgloss.Style
	Button     lipgloss.Style
	ButtonYes  lipgloss.Style
	ButtonNo   lipgloss.Style
	Background lipgloss.Style
}

// DefaultConfirmStyles returns the default dialog styles
func DefaultConfirmStyles() *ConfirmStyles {
	return &ConfirmStyles{
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")),
		Message: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Button: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		ButtonYes: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82")),
		ButtonNo: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")),
		Background: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// NewConfirm creates a new confirmation dialog
func NewConfirm(title, message string) *ConfirmModel {
	return &ConfirmModel{
		title:   title,
		message: message,
		result:  ConfirmPending,
		styles:  DefaultConfirmStyles(),
	}
}

// WithActionID sets an identifier for the action being confirmed
func (m *ConfirmModel) WithActionID(id string) *ConfirmModel {
	m.actionID = id
	return m
}

// SetSize sets the dialog dimensions
func (m *ConfirmModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Result returns the current result
func (m *ConfirmModel) Result() ConfirmResult {
	return m.result
}

// ActionID returns the action identifier
func (m *ConfirmModel) ActionID() string {
	return m.actionID
}

// Reset resets the dialog to pending state
func (m *ConfirmModel) Reset() {
	m.result = ConfirmPending
}

// Update handles key messages
func (m *ConfirmModel) Update(msg tea.Msg) (*ConfirmModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y", "enter"))):
			m.result = ConfirmYes
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "N", "esc", "q"))):
			m.result = ConfirmNo
		}
	}
	return m, nil
}

// View renders the confirmation dialog
func (m *ConfirmModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render(m.title))
	b.WriteString("\n\n")

	// Message
	b.WriteString(m.styles.Message.Render(m.message))
	b.WriteString("\n\n")

	// Buttons
	yesBtn := m.styles.ButtonYes.Render("[Y]es")
	noBtn := m.styles.ButtonNo.Render("[N]o")
	b.WriteString(fmt.Sprintf("%s  /  %s", yesBtn, noBtn))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Button.Render("[Esc/q] cancel"))

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

		// Build centered content
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
func (m *ConfirmModel) String() string {
	return fmt.Sprintf("Confirm{title=%q, result=%d}", m.title, m.result)
}
