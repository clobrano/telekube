package tui

import (
	"testing"

	"github.com/clobrano/telekube/internal/kubectl"
	"github.com/clobrano/telekube/internal/tui/components/list"
)

// newModelForTest builds a minimal Model wired with the given table data so the
// describe-identifier helpers can be exercised without kubectl.
func newModelForTest(data *kubectl.TableData) *Model {
	m := &Model{
		resources: data,
		list:      list.New(data.Headers, data.Rows),
	}
	return m
}

func TestCurrentTabHasIDAndName(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    bool
	}{
		{"both id and name", []string{"ID", "NAME", "AGE"}, true},
		{"name only", []string{"NAME", "READY", "AGE"}, false},
		{"id only", []string{"ID", "STATUS"}, false},
		{"neither", []string{"READY", "STATUS"}, false},
		{"case insensitive", []string{"id", "Name"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModelForTest(&kubectl.TableData{Headers: tt.headers})
			if got := m.currentTabHasIDAndName(); got != tt.want {
				t.Errorf("currentTabHasIDAndName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCurrentTabHasIDAndNameNilResources(t *testing.T) {
	m := &Model{}
	if m.currentTabHasIDAndName() {
		t.Error("currentTabHasIDAndName() = true for nil resources, want false")
	}
}

func TestGetSelectedResourceInfoByColumn(t *testing.T) {
	data := &kubectl.TableData{
		Headers: []string{"ID", "NAME", "AGE"},
		Rows: [][]string{
			{"id-123", "web-pod", "5m"},
		},
	}

	tests := []struct {
		name      string
		idColumn  string
		wantNames []string
	}{
		{"default uses NAME", "", []string{"web-pod"}},
		{"explicit NAME", "NAME", []string{"web-pod"}},
		{"explicit ID", "ID", []string{"id-123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModelForTest(data)
			names, _ := m.getSelectedResourceInfoBy(tt.idColumn)
			if len(names) != len(tt.wantNames) {
				t.Fatalf("got %d names, want %d (%v)", len(names), len(tt.wantNames), names)
			}
			for i := range names {
				if names[i] != tt.wantNames[i] {
					t.Errorf("names[%d] = %q, want %q", i, names[i], tt.wantNames[i])
				}
			}
		})
	}
}

func TestGetSelectedResourceInfoByNamespaceColumn(t *testing.T) {
	data := &kubectl.TableData{
		Headers: []string{"NAMESPACE", "ID", "NAME", "AGE"},
		Rows: [][]string{
			{"prod", "id-1", "pod-a", "1h"},
		},
	}

	m := newModelForTest(data)
	names, namespace := m.getSelectedResourceInfoBy("ID")
	if len(names) != 1 || names[0] != "id-1" {
		t.Errorf("names = %v, want [id-1]", names)
	}
	if namespace != "prod" {
		t.Errorf("namespace = %q, want %q", namespace, "prod")
	}
}
