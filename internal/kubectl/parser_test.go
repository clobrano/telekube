package kubectl

import (
	"reflect"
	"testing"
)

func TestParseTableOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantCols []string
		wantRows int
	}{
		{
			name: "pods output",
			input: `NAME                     READY   STATUS    RESTARTS   AGE
nginx-7c6b7d8b9-abc12    1/1     Running   0          2d
redis-5d8f7c9b1-xyz99    1/1     Running   0          5h`,
			wantCols: []string{"NAME", "READY", "STATUS", "RESTARTS", "AGE"},
			wantRows: 2,
		},
		{
			name: "pods with namespace output",
			input: `NAMESPACE     NAME                     READY   STATUS    RESTARTS   AGE
default       nginx-7c6b7d8b9-abc12    1/1     Running   0          2d
kube-system   coredns-5644d8b4d-abcde  1/1     Running   0          10d`,
			wantCols: []string{"NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE"},
			wantRows: 2,
		},
		{
			name: "deployments output",
			input: `NAME    READY   UP-TO-DATE   AVAILABLE   AGE
nginx   3/3     3            3           5d`,
			wantCols: []string{"NAME", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"},
			wantRows: 1,
		},
		{
			name: "services output",
			input: `NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   10.96.0.1       <none>        443/TCP   30d
nginx        ClusterIP   10.96.100.50    <none>        80/TCP    5d`,
			wantCols: []string{"NAME", "TYPE", "CLUSTER-IP", "EXTERNAL-IP", "PORT(S)", "AGE"},
			wantRows: 2,
		},
		{
			name:     "empty output",
			input:    "",
			wantCols: nil,
			wantRows: 0,
		},
		{
			name: "header only",
			input: `NAME    READY   STATUS
`,
			wantCols: []string{"NAME", "READY", "STATUS"},
			wantRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTableOutput(tt.input)

			if !reflect.DeepEqual(result.Headers, tt.wantCols) && !(len(result.Headers) == 0 && tt.wantCols == nil) {
				t.Errorf("Headers = %v, want %v", result.Headers, tt.wantCols)
			}

			if len(result.Rows) != tt.wantRows {
				t.Errorf("Rows count = %d, want %d", len(result.Rows), tt.wantRows)
			}
		})
	}
}

func TestParseTableOutputValues(t *testing.T) {
	input := `NAME                     READY   STATUS    RESTARTS   AGE
nginx-7c6b7d8b9-abc12    1/1     Running   0          2d
redis-5d8f7c9b1-xyz99    0/1     Pending   5          5h`

	result := ParseTableOutput(input)

	// Check first row values
	if result.Rows[0][0] != "nginx-7c6b7d8b9-abc12" {
		t.Errorf("Row 0, Col 0 = %q, want %q", result.Rows[0][0], "nginx-7c6b7d8b9-abc12")
	}
	if result.Rows[0][1] != "1/1" {
		t.Errorf("Row 0, Col 1 = %q, want %q", result.Rows[0][1], "1/1")
	}
	if result.Rows[0][2] != "Running" {
		t.Errorf("Row 0, Col 2 = %q, want %q", result.Rows[0][2], "Running")
	}

	// Check second row values
	if result.Rows[1][0] != "redis-5d8f7c9b1-xyz99" {
		t.Errorf("Row 1, Col 0 = %q, want %q", result.Rows[1][0], "redis-5d8f7c9b1-xyz99")
	}
	if result.Rows[1][2] != "Pending" {
		t.Errorf("Row 1, Col 2 = %q, want %q", result.Rows[1][2], "Pending")
	}
}

func TestGetColumnIndex(t *testing.T) {
	data := &TableData{
		Headers: []string{"NAME", "READY", "STATUS"},
		Rows:    [][]string{{"nginx", "1/1", "Running"}},
	}

	tests := []struct {
		name    string
		col     string
		wantIdx int
	}{
		{"exact match", "NAME", 0},
		{"case insensitive", "name", 0},
		{"mixed case", "Status", 2},
		{"not found", "NAMESPACE", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := data.GetColumnIndex(tt.col)
			if idx != tt.wantIdx {
				t.Errorf("GetColumnIndex(%q) = %d, want %d", tt.col, idx, tt.wantIdx)
			}
		})
	}
}

func TestGetColumn(t *testing.T) {
	data := &TableData{
		Headers: []string{"NAME", "READY", "STATUS"},
		Rows: [][]string{
			{"nginx", "1/1", "Running"},
			{"redis", "0/1", "Pending"},
		},
	}

	names := data.GetColumn("NAME")
	expected := []string{"nginx", "redis"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("GetColumn(NAME) = %v, want %v", names, expected)
	}

	statuses := data.GetColumn("status")
	expectedStatuses := []string{"Running", "Pending"}
	if !reflect.DeepEqual(statuses, expectedStatuses) {
		t.Errorf("GetColumn(status) = %v, want %v", statuses, expectedStatuses)
	}

	// Non-existent column
	missing := data.GetColumn("NAMESPACE")
	if missing != nil {
		t.Errorf("GetColumn(NAMESPACE) = %v, want nil", missing)
	}
}

func TestParseTableOutputMultiWordHeader(t *testing.T) {
	// "RELEASE STATUS" is a single column with a space in its name (kubectl CRD output).
	// The parser must treat single-space gaps as part of the header word, not column boundaries.
	input := "NAME                                 SNAPSHOT              RELEASEPLAN        RELEASE STATUS   AGE\n" +
		"release-foo                          snapshot-foo          releaseplan-foo    Succeeded        3d\n" +
		"release-bar                          snapshot-bar          releaseplan-bar    Failed           1d"

	result := ParseTableOutput(input)

	wantHeaders := []string{"NAME", "SNAPSHOT", "RELEASEPLAN", "RELEASE STATUS", "AGE"}
	if !reflect.DeepEqual(result.Headers, wantHeaders) {
		t.Errorf("Headers = %v, want %v", result.Headers, wantHeaders)
	}

	if len(result.Rows) != 2 {
		t.Fatalf("Rows count = %d, want 2", len(result.Rows))
	}

	statusIdx := result.GetColumnIndex("RELEASE STATUS")
	if statusIdx < 0 {
		t.Fatal("column 'RELEASE STATUS' not found")
	}
	if result.Rows[0][statusIdx] != "Succeeded" {
		t.Errorf("Row 0 RELEASE STATUS = %q, want %q", result.Rows[0][statusIdx], "Succeeded")
	}
	if result.Rows[1][statusIdx] != "Failed" {
		t.Errorf("Row 1 RELEASE STATUS = %q, want %q", result.Rows[1][statusIdx], "Failed")
	}
}

func TestKubectlError(t *testing.T) {
	tests := []struct {
		name     string
		err      *KubectlError
		wantMsg  string
	}{
		{
			name:    "with stderr",
			err:     &KubectlError{Stderr: "error: resource not found\n", Err: nil},
			wantMsg: "error: resource not found",
		},
		{
			name:    "without stderr",
			err:     &KubectlError{Stderr: "", Err: &testError{msg: "exec failed"}},
			wantMsg: "exec failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
