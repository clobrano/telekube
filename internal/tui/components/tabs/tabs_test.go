package tabs

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tabs := New([]string{"Pods", "Deployments", "Services"}, 0)

	if tabs.Active() != 0 {
		t.Errorf("Active() = %d, want 0", tabs.Active())
	}

	if tabs.Count() != 3 {
		t.Errorf("Count() = %d, want 3", tabs.Count())
	}
}

func TestNewWithInvalidIndex(t *testing.T) {
	// Negative index should be clamped to 0
	tabs := New([]string{"A", "B"}, -1)
	if tabs.Active() != 0 {
		t.Errorf("Active() with negative index = %d, want 0", tabs.Active())
	}

	// Index beyond range should be clamped to last
	tabs = New([]string{"A", "B"}, 10)
	if tabs.Active() != 1 {
		t.Errorf("Active() with out-of-range index = %d, want 1", tabs.Active())
	}
}

func TestSetActive(t *testing.T) {
	tabs := New([]string{"A", "B", "C"}, 0)

	tabs.SetActive(1)
	if tabs.Active() != 1 {
		t.Errorf("Active() after SetActive(1) = %d, want 1", tabs.Active())
	}

	// Invalid indices should be ignored
	tabs.SetActive(-1)
	if tabs.Active() != 1 {
		t.Errorf("Active() after SetActive(-1) = %d, want 1", tabs.Active())
	}

	tabs.SetActive(100)
	if tabs.Active() != 1 {
		t.Errorf("Active() after SetActive(100) = %d, want 1", tabs.Active())
	}
}

func TestActiveName(t *testing.T) {
	tabs := New([]string{"Pods", "Deployments"}, 0)

	if tabs.ActiveName() != "Pods" {
		t.Errorf("ActiveName() = %q, want %q", tabs.ActiveName(), "Pods")
	}

	tabs.SetActive(1)
	if tabs.ActiveName() != "Deployments" {
		t.Errorf("ActiveName() = %q, want %q", tabs.ActiveName(), "Deployments")
	}
}

func TestNext(t *testing.T) {
	tabs := New([]string{"A", "B", "C"}, 0)

	tabs.Next()
	if tabs.Active() != 1 {
		t.Errorf("Active() after Next() = %d, want 1", tabs.Active())
	}

	tabs.Next()
	if tabs.Active() != 2 {
		t.Errorf("Active() after second Next() = %d, want 2", tabs.Active())
	}

	// Should wrap around
	tabs.Next()
	if tabs.Active() != 0 {
		t.Errorf("Active() after wrap-around Next() = %d, want 0", tabs.Active())
	}
}

func TestPrevious(t *testing.T) {
	tabs := New([]string{"A", "B", "C"}, 2)

	tabs.Previous()
	if tabs.Active() != 1 {
		t.Errorf("Active() after Previous() = %d, want 1", tabs.Active())
	}

	tabs.Previous()
	if tabs.Active() != 0 {
		t.Errorf("Active() after second Previous() = %d, want 0", tabs.Active())
	}

	// Should wrap around
	tabs.Previous()
	if tabs.Active() != 2 {
		t.Errorf("Active() after wrap-around Previous() = %d, want 2", tabs.Active())
	}
}

func TestNextPreviousEmpty(t *testing.T) {
	tabs := New([]string{}, 0)

	// Should not panic
	tabs.Next()
	tabs.Previous()
}

func TestViewContainsTabs(t *testing.T) {
	tabs := New([]string{"Pods", "Services"}, 0)
	view := tabs.View()

	if !strings.Contains(view, "Pods") {
		t.Error("View should contain 'Pods'")
	}

	if !strings.Contains(view, "Services") {
		t.Error("View should contain 'Services'")
	}
}

func TestViewContainsNumbers(t *testing.T) {
	tabs := New([]string{"A", "B", "C"}, 0)
	view := tabs.View()

	if !strings.Contains(view, "1") {
		t.Error("View should contain tab number '1'")
	}

	if !strings.Contains(view, "2") {
		t.Error("View should contain tab number '2'")
	}

	if !strings.Contains(view, "3") {
		t.Error("View should contain tab number '3'")
	}
}

func TestViewEmpty(t *testing.T) {
	tabs := New([]string{}, 0)
	view := tabs.View()

	if view != "" {
		t.Errorf("View with no tabs should be empty, got %q", view)
	}
}

func TestString(t *testing.T) {
	tabs := New([]string{"A", "B"}, 1)
	s := tabs.String()

	if !strings.Contains(s, "active=1") {
		t.Errorf("String() = %q, should contain 'active=1'", s)
	}
}

func TestRemoveTabMiddleIndex(t *testing.T) {
	// Removing middle tab should keep active index consistent if not removed
	tabs := New([]string{"Search", "Pods", "Deployments", "Services"}, 1)

	tabs.RemoveTab(2) // Remove "Deployments"

	if tabs.Count() != 3 {
		t.Errorf("Count() after removing tab = %d, want 3", tabs.Count())
	}

	// Active should still be 1 (Pods) since we didn't remove it
	if tabs.Active() != 1 {
		t.Errorf("Active() after removing middle tab = %d, want 1", tabs.Active())
	}

	if tabs.ActiveName() != "Pods" {
		t.Errorf("ActiveName() = %q, want %q", tabs.ActiveName(), "Pods")
	}
}

func TestRemoveTabLastIndex(t *testing.T) {
	// Removing the last tab should set active to previous tab
	tabs := New([]string{"Search", "Pods", "Deployments", "Services"}, 3)

	tabs.RemoveTab(3) // Remove "Services"

	if tabs.Count() != 3 {
		t.Errorf("Count() after removing last tab = %d, want 3", tabs.Count())
	}

	// Active should be set to previous tab (index 2)
	if tabs.Active() != 2 {
		t.Errorf("Active() after removing last tab = %d, want 2", tabs.Active())
	}

	if tabs.ActiveName() != "Deployments" {
		t.Errorf("ActiveName() = %q, want %q", tabs.ActiveName(), "Deployments")
	}
}

func TestRemoveTabSearchTabBlocked(t *testing.T) {
	// Removing Search tab (index 0) should be prevented
	tabs := New([]string{"Search", "Pods", "Deployments"}, 0)

	tabs.RemoveTab(0) // Try to remove "Search"

	if tabs.Count() != 3 {
		t.Errorf("Count() after trying to remove Search tab = %d, want 3", tabs.Count())
	}

	if tabs.ActiveName() != "Search" {
		t.Errorf("ActiveName() = %q, want %q", tabs.ActiveName(), "Search")
	}
}

func TestRemoveTabEmptyList(t *testing.T) {
	// Removing from empty list should not panic
	tabs := New([]string{}, 0)

	tabs.RemoveTab(0) // Should not panic

	if tabs.Count() != 0 {
		t.Errorf("Count() of empty tabs = %d, want 0", tabs.Count())
	}
}

func TestRemoveTabUpdateActiveAfterRemoval(t *testing.T) {
	// If active tab is removed, active should shift to previous
	tabs := New([]string{"Search", "Pods", "Deployments"}, 2)

	tabs.RemoveTab(2) // Remove active tab "Deployments"

	if tabs.Count() != 2 {
		t.Errorf("Count() after removing active tab = %d, want 2", tabs.Count())
	}

	// Active should shift to previous (index 1)
	if tabs.Active() != 1 {
		t.Errorf("Active() after removing active tab = %d, want 1", tabs.Active())
	}
}
