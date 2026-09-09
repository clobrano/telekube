package detail

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Format represents the display format for the detail view
type Format int

const (
	FormatTable Format = iota
	FormatYAML
	FormatJSON
)

// String returns the string representation of the format
func (f Format) String() string {
	switch f {
	case FormatYAML:
		return "YAML"
	case FormatJSON:
		return "JSON"
	default:
		return "Table"
	}
}

// Model represents the detail view component
type Model struct {
	resourceName string
	content      string
	format       Format
	scrollOffset int
	width        int
	height       int
	styles       *Styles

	// Search state
	searching    bool
	searchInput  textinput.Model
	searchQuery  string // confirmed query (persists after closing search bar)
	matchLines   []int  // line indices that contain a match
	currentMatch int    // index into matchLines for the current match
}

// Styles defines the styles for the detail view component
type Styles struct {
	Header      lipgloss.Style
	Content     lipgloss.Style
	LineNum     lipgloss.Style
	KeyWord     lipgloss.Style
	String      lipgloss.Style
	Number      lipgloss.Style
	Hint        lipgloss.Style
	SearchMatch lipgloss.Style
}

// DefaultStyles returns the default detail view styles
func DefaultStyles() *Styles {
	return &Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("237")).
			Padding(0, 1),
		Content: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		LineNum: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(5).
			Align(lipgloss.Right),
		KeyWord: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		String: lipgloss.NewStyle().
			Foreground(lipgloss.Color("78")),
		Number: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")),
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		SearchMatch: lipgloss.NewStyle().
			Background(lipgloss.Color("214")).
			Foreground(lipgloss.Color("0")).
			Bold(true),
	}
}

// newSearchInput creates a configured text input for searching
func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "search..."
	ti.CharLimit = 256
	return ti
}

// New creates a new detail view component
func New(resourceName, content string, format Format) *Model {
	return &Model{
		resourceName: resourceName,
		content:      content,
		format:       format,
		styles:       DefaultStyles(),
		searchInput:  newSearchInput(),
	}
}

// SetContent updates the content and format. It also resets any search state
// so a freshly opened view never inherits a stale search from a previous one
// (a lingering search query or an open search bar would otherwise absorb the
// Esc key and prevent the view from closing).
func (m *Model) SetContent(resourceName, content string, format Format) {
	m.resourceName = resourceName
	m.content = content
	m.format = format
	m.scrollOffset = 0
	m.ClearSearch()
}

// StartSearch activates the search input bar
func (m *Model) StartSearch() {
	m.searching = true
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	m.matchLines = nil
	m.currentMatch = 0
}

// StopSearch closes the search bar, keeping the current query highlighted
func (m *Model) StopSearch() {
	m.searching = false
	m.searchInput.Blur()
}

// ClearSearch closes the search bar and clears all highlights
func (m *Model) ClearSearch() {
	m.searching = false
	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.searchQuery = ""
	m.matchLines = nil
	m.currentMatch = 0
}

// IsSearching returns whether the search input is active
func (m *Model) IsSearching() bool {
	return m.searching
}

// HasSearchQuery returns whether a search query is active (highlighted)
func (m *Model) HasSearchQuery() bool {
	return m.searchQuery != ""
}

// updateMatches recalculates match lines for the current query
func (m *Model) updateMatches() {
	m.matchLines = nil
	m.currentMatch = 0
	if m.searchQuery == "" {
		return
	}
	lines := m.wrapLines()
	lowerQuery := strings.ToLower(m.searchQuery)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), lowerQuery) {
			m.matchLines = append(m.matchLines, i)
		}
	}
}

// scrollToCurrentMatch scrolls to bring the current match into view
func (m *Model) scrollToCurrentMatch() {
	if len(m.matchLines) == 0 {
		return
	}
	lineIdx := m.matchLines[m.currentMatch]
	ch := m.contentHeight()
	// Scroll so the match line is visible (roughly centered)
	m.scrollOffset = lineIdx - ch/2
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
	m.ensureScrollBounds()
}

// NextMatch jumps to the next search match
func (m *Model) NextMatch() {
	if len(m.matchLines) == 0 {
		return
	}
	m.currentMatch = (m.currentMatch + 1) % len(m.matchLines)
	m.scrollToCurrentMatch()
}

// PrevMatch jumps to the previous search match
func (m *Model) PrevMatch() {
	if len(m.matchLines) == 0 {
		return
	}
	m.currentMatch--
	if m.currentMatch < 0 {
		m.currentMatch = len(m.matchLines) - 1
	}
	m.scrollToCurrentMatch()
}

// Update handles messages when the detail view has an active search input.
// Returns an updated model and an optional command.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if !m.searching {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Confirm search query
			m.searchQuery = m.searchInput.Value()
			m.updateMatches()
			m.StopSearch()
			if len(m.matchLines) > 0 {
				// Jump to first match at or after current scroll position
				m.currentMatch = 0
				for i, lineIdx := range m.matchLines {
					if lineIdx >= m.scrollOffset {
						m.currentMatch = i
						break
					}
				}
				m.scrollToCurrentMatch()
			}
			return m, nil
		case tea.KeyEsc:
			m.StopSearch()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// SetSize sets the dimensions for the detail view
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.ensureScrollBounds()
}

// ScrollUp scrolls up by one line
func (m *Model) ScrollUp() {
	if m.scrollOffset > 0 {
		m.scrollOffset--
	}
}

// ScrollDown scrolls down by one line
func (m *Model) ScrollDown() {
	m.scrollOffset++
	m.ensureScrollBounds()
}

// ScrollToTop scrolls to the top
func (m *Model) ScrollToTop() {
	m.scrollOffset = 0
}

// ScrollToBottom scrolls to the bottom
func (m *Model) ScrollToBottom() {
	lines := m.wrapLines()
	m.scrollOffset = len(lines) - m.contentHeight()
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// PageUp scrolls up by one page
func (m *Model) PageUp() {
	m.scrollOffset -= m.contentHeight()
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// PageDown scrolls down by one page
func (m *Model) PageDown() {
	m.scrollOffset += m.contentHeight()
	m.ensureScrollBounds()
}

// contentHeight returns the available height for content
func (m *Model) contentHeight() int {
	// Reserve 2 lines for header and scroll indicator
	h := m.height - 2
	// Reserve 1 more line when search bar is visible
	if m.searching {
		h--
	}
	if h < 1 {
		h = 1
	}
	return h
}

// wrapLines returns lines wrapped to fit the current width
func (m *Model) wrapLines() []string {
	rawLines := strings.Split(m.content, "\n")
	maxWidth := m.width - 2
	if maxWidth < 1 {
		maxWidth = 1
	}
	var lines []string
	for _, raw := range rawLines {
		if len(raw) <= maxWidth {
			lines = append(lines, raw)
		} else {
			for len(raw) > maxWidth {
				lines = append(lines, raw[:maxWidth])
				raw = raw[maxWidth:]
			}
			lines = append(lines, raw)
		}
	}
	return lines
}

// ensureScrollBounds ensures scroll offset is within valid bounds
func (m *Model) ensureScrollBounds() {
	lines := m.wrapLines()
	maxOffset := len(lines) - m.contentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// Content returns the current content
func (m *Model) Content() string {
	return m.content
}

// Format returns the current format
func (m *Model) Format() Format {
	return m.format
}

// ResourceName returns the resource name
func (m *Model) ResourceName() string {
	return m.resourceName
}

// highlightLine returns the line with search matches highlighted
func (m *Model) highlightLine(line string, lineIdx int) string {
	if m.searchQuery == "" {
		return m.styles.Content.Render(line)
	}

	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(m.searchQuery)
	queryLen := len(m.searchQuery)

	idx := strings.Index(lowerLine, lowerQuery)
	if idx < 0 {
		return m.styles.Content.Render(line)
	}

	// Build the line with highlighted matches
	var result strings.Builder
	pos := 0
	for idx >= 0 {
		// Render text before the match
		if idx > pos {
			result.WriteString(m.styles.Content.Render(line[pos:idx]))
		}
		// Render the match highlighted
		result.WriteString(m.styles.SearchMatch.Render(line[idx : idx+queryLen]))
		pos = idx + queryLen
		next := strings.Index(lowerLine[pos:], lowerQuery)
		if next < 0 {
			break
		}
		idx = pos + next
	}
	// Render remaining text
	if pos < len(line) {
		result.WriteString(m.styles.Content.Render(line[pos:]))
	}
	return result.String()
}

// View renders the detail view
func (m *Model) View() string {
	var b strings.Builder

	// Header with resource name and format
	headerText := fmt.Sprintf("%s - %s", m.resourceName, m.format.String())
	hintText := "[Esc] Back  [/] Search"
	if m.searchQuery != "" {
		hintText = "[Esc] Back  [/] Search  [n/N] Next/Prev"
	}
	hint := m.styles.Hint.Render(hintText)
	headerPadding := m.width - lipgloss.Width(headerText) - lipgloss.Width(hint) - 4
	if headerPadding < 1 {
		headerPadding = 1
	}
	header := headerText + strings.Repeat(" ", headerPadding) + hint
	b.WriteString(m.styles.Header.Width(m.width).Render(header))
	b.WriteString("\n")

	// Search bar (when active)
	if m.searching {
		b.WriteString(m.searchInput.View())
		b.WriteString("\n")
	}

	// Content - wrap long lines to fit width
	lines := m.wrapLines()

	contentHeight := m.contentHeight()

	endIdx := m.scrollOffset + contentHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		b.WriteString(m.highlightLine(lines[i], i))
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Fill remaining space
	renderedLines := endIdx - m.scrollOffset
	for i := renderedLines; i < contentHeight; i++ {
		b.WriteString("\n")
	}

	// Scroll indicator
	b.WriteString("\n")
	scrollInfo := fmt.Sprintf("Line %d-%d of %d", m.scrollOffset+1, endIdx, len(lines))
	if m.scrollOffset > 0 {
		scrollInfo = "↑ " + scrollInfo
	}
	if endIdx < len(lines) {
		scrollInfo = scrollInfo + " ↓"
	}
	if m.searchQuery != "" {
		if len(m.matchLines) > 0 {
			scrollInfo += fmt.Sprintf("  [%d/%d matches]", m.currentMatch+1, len(m.matchLines))
		} else {
			scrollInfo += "  [no matches]"
		}
	}
	b.WriteString(m.styles.Hint.Render(scrollInfo))

	return b.String()
}

// String implements fmt.Stringer
func (m *Model) String() string {
	return fmt.Sprintf("Detail{resource=%q, format=%s, lines=%d}", m.resourceName, m.format, len(strings.Split(m.content, "\n")))
}
