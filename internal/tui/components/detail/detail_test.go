package detail

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	d := New("nginx-pod", "apiVersion: v1", FormatYAML)

	if d.ResourceName() != "nginx-pod" {
		t.Errorf("ResourceName() = %q, want %q", d.ResourceName(), "nginx-pod")
	}

	if d.Format() != FormatYAML {
		t.Errorf("Format() = %v, want FormatYAML", d.Format())
	}

	if d.Content() != "apiVersion: v1" {
		t.Errorf("Content() = %q, want %q", d.Content(), "apiVersion: v1")
	}
}

func TestSetContent(t *testing.T) {
	d := New("old", "old content", FormatTable)
	d.SetContent("new", "new content", FormatJSON)

	if d.ResourceName() != "new" {
		t.Errorf("ResourceName() after SetContent = %q, want %q", d.ResourceName(), "new")
	}

	if d.Format() != FormatJSON {
		t.Errorf("Format() after SetContent = %v, want FormatJSON", d.Format())
	}

	if d.Content() != "new content" {
		t.Errorf("Content() after SetContent = %q, want %q", d.Content(), "new content")
	}
}

// TestSetContentResetsSearch guards against search state leaking into a freshly
// opened view. A lingering query or open search bar would otherwise absorb the
// Esc key and prevent the view from closing.
func TestSetContentResetsSearch(t *testing.T) {
	d := New("pod", "line one\nline two\nline three", FormatYAML)

	// Simulate an active search with a confirmed query.
	d.StartSearch()
	d.searchInput.SetValue("line")
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if !d.HasSearchQuery() {
		t.Fatal("precondition: expected an active search query")
	}

	// Loading new content must wipe all search state.
	d.SetContent("Delete Result", "Deleted foo.", FormatTable)

	if d.HasSearchQuery() {
		t.Error("HasSearchQuery() = true after SetContent, want false")
	}
	if d.IsSearching() {
		t.Error("IsSearching() = true after SetContent, want false")
	}
}

func TestFormatString(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatTable, "Table"},
		{FormatYAML, "YAML"},
		{FormatJSON, "JSON"},
	}

	for _, tt := range tests {
		if got := tt.format.String(); got != tt.want {
			t.Errorf("Format(%v).String() = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestScrollUp(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	d := New("test", content, FormatTable)
	d.SetSize(80, 5) // 3 visible lines

	// Move down first
	d.ScrollDown()
	d.ScrollDown()

	d.ScrollUp()
	// Should be at offset 1 now

	d.ScrollUp()
	d.ScrollUp() // Extra scroll shouldn't go negative

	// scrollOffset should be 0, not negative
	view := d.View()
	if !strings.Contains(view, "line1") {
		t.Error("After scrolling up to top, view should contain 'line1'")
	}
}

func TestScrollDown(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	d := New("test", content, FormatTable)
	d.SetSize(80, 5) // 3 visible lines

	d.ScrollDown()
	view := d.View()

	if !strings.Contains(view, "line2") {
		t.Error("After scrolling down, view should contain 'line2'")
	}
}

func TestScrollToTop(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	d := New("test", content, FormatTable)
	d.SetSize(80, 5)

	d.ScrollDown()
	d.ScrollDown()
	d.ScrollToTop()

	view := d.View()
	if !strings.Contains(view, "line1") {
		t.Error("After ScrollToTop, view should contain 'line1'")
	}
}

func TestScrollToBottom(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5"
	d := New("test", content, FormatTable)
	d.SetSize(80, 5) // 3 visible lines

	d.ScrollToBottom()

	view := d.View()
	if !strings.Contains(view, "line5") {
		t.Error("After ScrollToBottom, view should contain 'line5'")
	}
}

func TestPageUpDown(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line" + string(rune('A'+i))
	}
	content := strings.Join(lines, "\n")

	d := New("test", content, FormatTable)
	d.SetSize(80, 7) // 5 visible lines

	d.PageDown()
	view := d.View()
	// Should have scrolled down by 5 lines
	if strings.Contains(view, "lineA") {
		t.Error("After PageDown, view should not contain 'lineA'")
	}

	d.PageUp()
	view = d.View()
	if !strings.Contains(view, "lineA") {
		t.Error("After PageUp, view should contain 'lineA'")
	}
}

func TestViewContainsHeader(t *testing.T) {
	d := New("my-resource", "content", FormatYAML)
	d.SetSize(80, 10)
	view := d.View()

	if !strings.Contains(view, "my-resource") {
		t.Error("View should contain resource name")
	}

	if !strings.Contains(view, "YAML") {
		t.Error("View should contain format name")
	}

	if !strings.Contains(view, "[Esc] Back") {
		t.Error("View should contain back hint")
	}
}

func TestViewContainsContent(t *testing.T) {
	d := New("test", "apiVersion: v1\nkind: Pod", FormatYAML)
	d.SetSize(80, 10)
	view := d.View()

	if !strings.Contains(view, "apiVersion") {
		t.Error("View should contain content")
	}

	if !strings.Contains(view, "kind") {
		t.Error("View should contain content")
	}
}

func TestViewScrollIndicator(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	d := New("test", content, FormatTable)
	d.SetSize(80, 5) // Only 3 lines visible

	view := d.View()
	// Should show down arrow indicator since there's more content below
	if !strings.Contains(view, "↓") {
		t.Error("View should show scroll down indicator")
	}

	d.ScrollToBottom()
	view = d.View()
	// Should show up arrow indicator since there's more content above
	if !strings.Contains(view, "↑") {
		t.Error("View should show scroll up indicator when not at top")
	}
}

func TestString(t *testing.T) {
	d := New("test", "line1\nline2", FormatYAML)
	s := d.String()

	if !strings.Contains(s, "test") {
		t.Errorf("String() = %q, should contain resource name", s)
	}

	if !strings.Contains(s, "YAML") {
		t.Errorf("String() = %q, should contain format", s)
	}
}
