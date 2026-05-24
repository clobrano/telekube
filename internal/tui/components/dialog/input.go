package dialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// InputResult represents the result of an input dialog
type InputResult int

const (
	InputPending InputResult = iota
	InputSubmitted
	InputCancelled
)

// InputModel represents an input dialog
type InputModel struct {
	title       string
	placeholder string
	hint        string
	extraHints  string // appended to the key-hint line at the bottom
	textInput   textinput.Model
	result      InputResult
	width       int
	height      int
	styles      *InputStyles
	actionID    string

	// History navigation
	history      []string // oldest → newest
	historyIndex int      // -1 = editing draft, 0..n-1 = position in history
	draft        string   // saved draft before navigating history
	suggestion   string   // autosuggestion suffix from history
}

// InputStyles defines the styles for the input dialog
type InputStyles struct {
	Dialog     lipgloss.Style
	Title      lipgloss.Style
	Input      lipgloss.Style
	Hint       lipgloss.Style
	Suggestion lipgloss.Style
}

// DefaultInputStyles returns the default input styles
func DefaultInputStyles() *InputStyles {
	return &InputStyles{
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")),
		Input: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		Suggestion: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// NewInput creates a new input dialog
func NewInput(title, placeholder string) *InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 30

	return &InputModel{
		title:        title,
		placeholder:  placeholder,
		textInput:    ti,
		result:       InputPending,
		styles:       DefaultInputStyles(),
		historyIndex: -1,
	}
}

// WithHint sets a descriptive hint shown below the title
func (m *InputModel) WithHint(hint string) *InputModel {
	m.hint = hint
	return m
}

// WithActionID sets an identifier for the action
func (m *InputModel) WithActionID(id string) *InputModel {
	m.actionID = id
	return m
}

// WithExtraHints appends extra key hints to the bottom hint line of the dialog.
func (m *InputModel) WithExtraHints(hints string) *InputModel {
	m.extraHints = hints
	return m
}

// WithValue sets the initial value (builder pattern)
func (m *InputModel) WithValue(value string) *InputModel {
	m.textInput.SetValue(value)
	return m
}

// SetValue sets the input value and places the cursor at the end
func (m *InputModel) SetValue(value string) {
	m.textInput.SetValue(value)
	m.textInput.CursorEnd()
	m.historyIndex = -1
	m.draft = ""
	m.suggestion = ""
	m.computeSuggestion(value)
}

// SetSize sets the dialog dimensions and adjusts the input width to 70% of available space
func (m *InputModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Dialog is 70% of terminal width, minus border/padding (2 border + 4 padding = 6)
	dialogInner := width*70/100 - 6
	if dialogInner > 0 {
		m.textInput.Width = dialogInner
	}
}

// Result returns the current result
func (m *InputModel) Result() InputResult {
	return m.result
}

// Value returns the input value
func (m *InputModel) Value() string {
	return m.textInput.Value()
}

// ActionID returns the action identifier
func (m *InputModel) ActionID() string {
	return m.actionID
}

// Reset resets the dialog to pending state
func (m *InputModel) Reset() {
	m.result = InputPending
	m.textInput.SetValue("")
	m.textInput.Focus()
	m.historyIndex = -1
	m.draft = ""
	m.suggestion = ""
	m.actionID = ""
	m.extraHints = ""
}

// HasSuggestion returns true when an autosuggestion is available.
func (m *InputModel) HasSuggestion() bool {
	return m.suggestion != ""
}

// SuggestionFull returns the complete string that Tab would fill in.
func (m *InputModel) SuggestionFull() string {
	if m.suggestion == "" {
		return ""
	}
	return m.textInput.Value() + m.suggestion
}

// HistoryLen returns the number of entries in the command history.
func (m *InputModel) HistoryLen() int {
	return len(m.history)
}

func (m *InputModel) addToHistory(value string) {
	if value == "" {
		return
	}
	if len(m.history) > 0 && m.history[len(m.history)-1] == value {
		return
	}
	m.history = append(m.history, value)
}

func (m *InputModel) computeSuggestion(input string) {
	if input == "" {
		m.suggestion = ""
		return
	}
	for i := len(m.history) - 1; i >= 0; i-- {
		if strings.HasPrefix(m.history[i], input) && m.history[i] != input {
			m.suggestion = m.history[i][len(input):]
			return
		}
	}
	m.suggestion = ""
}

// Update handles messages
func (m *InputModel) Update(msg tea.Msg) (*InputModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.textInput.Value() != "" {
				m.addToHistory(m.textInput.Value())
			}
			m.result = InputSubmitted
			m.historyIndex = -1
			m.draft = ""
			m.suggestion = ""
			return m, nil

		case "esc":
			m.result = InputCancelled
			m.historyIndex = -1
			m.draft = ""
			m.suggestion = ""
			return m, nil

		case "up":
			if len(m.history) == 0 {
				return m, nil
			}
			if m.historyIndex == -1 {
				m.draft = m.textInput.Value()
				m.historyIndex = len(m.history) - 1
			} else if m.historyIndex > 0 {
				m.historyIndex--
			}
			val := m.history[m.historyIndex]
			m.textInput.SetValue(val)
			m.textInput.CursorEnd()
			m.suggestion = ""
			return m, nil

		case "down":
			if m.historyIndex == -1 {
				return m, nil
			}
			if m.historyIndex < len(m.history)-1 {
				m.historyIndex++
				val := m.history[m.historyIndex]
				m.textInput.SetValue(val)
				m.textInput.CursorEnd()
			} else {
				m.historyIndex = -1
				m.textInput.SetValue(m.draft)
				m.textInput.CursorEnd()
				m.computeSuggestion(m.draft)
			}
			m.suggestion = ""
			return m, nil

		case "tab":
			if m.suggestion != "" {
				full := m.textInput.Value() + m.suggestion
				m.textInput.SetValue(full)
				m.textInput.CursorEnd()
				m.suggestion = ""
				m.historyIndex = -1
			}
			return m, nil

		case "left", "right", "ctrl+a", "ctrl+e", "ctrl+b", "ctrl+f", "home", "end":
			m.suggestion = ""
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	if _, ok := msg.(tea.KeyMsg); ok {
		m.historyIndex = -1
		m.computeSuggestion(m.textInput.Value())
	}
	return m, cmd
}

// View renders the input dialog
func (m *InputModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Title.Render(m.title))
	b.WriteString("\n")

	// Hint (if set)
	if m.hint != "" {
		b.WriteString(m.styles.Hint.Render(m.hint))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Input field with optional inline ghost text
	if m.suggestion != "" {
		// textinput.View() pads to its full width; render manually to keep the
		// ghost text adjacent to the cursor.
		cursor := lipgloss.NewStyle().Reverse(true).Render(" ")
		ghost := m.styles.Suggestion.Render(m.suggestion)
		b.WriteString(m.textInput.Value() + cursor + ghost)
	} else {
		b.WriteString(m.textInput.View())
	}
	b.WriteString("\n\n")

	// Dynamic key hints
	hints := "[Enter] submit  [Esc] cancel"
	if len(m.history) > 0 {
		hints += "  [↑↓] history"
	}
	if m.suggestion != "" {
		hints += "  [Tab] → " + m.SuggestionFull()
	}
	if m.extraHints != "" {
		hints += "  " + m.extraHints
	}
	b.WriteString(m.styles.Hint.Render(hints))

	// Use 70% of terminal width for the dialog
	dialogWidth := m.width * 70 / 100
	if dialogWidth < 40 {
		dialogWidth = 40
	}
	dialogStyle := m.styles.Dialog.Width(dialogWidth)
	content := dialogStyle.Render(b.String())

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
func (m *InputModel) String() string {
	return fmt.Sprintf("Input{title=%q, value=%q}", m.title, m.textInput.Value())
}
