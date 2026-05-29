package kubectl

import (
	"strings"
)

// TableData represents parsed kubectl table output
type TableData struct {
	Headers []string
	Rows    [][]string
}

// ParseTableOutput parses kubectl table output into headers and rows
func ParseTableOutput(output string) *TableData {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return &TableData{}
	}

	// Find column boundaries from header line
	headerLine := lines[0]
	if headerLine == "" {
		return &TableData{}
	}

	columns := parseColumnBoundaries(headerLine)
	headers := extractFields(headerLine, columns)

	var rows [][]string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		row := extractFields(line, columns)
		rows = append(rows, row)
	}

	return &TableData{
		Headers: headers,
		Rows:    rows,
	}
}

// parseColumnBoundaries determines column start positions from header line
// Columns are separated by 2+ spaces; single spaces are part of a multi-word column name
func parseColumnBoundaries(headerLine string) []int {
	var columns []int
	spaceRun := 0
	inWord := false

	for i, ch := range headerLine {
		if ch != ' ' {
			if !inWord {
				// First character ever, or following a gap of 2+ spaces = new column
				if len(columns) == 0 || spaceRun >= 2 {
					columns = append(columns, i)
				}
				inWord = true
			}
			spaceRun = 0
		} else {
			spaceRun++
			inWord = false
		}
	}

	return columns
}

// extractFields extracts fields from a line based on column positions
func extractFields(line string, columns []int) []string {
	var fields []string

	for i, start := range columns {
		var end int
		if i+1 < len(columns) {
			end = columns[i+1]
		} else {
			end = len(line)
		}

		if start >= len(line) {
			fields = append(fields, "")
			continue
		}

		if end > len(line) {
			end = len(line)
		}

		field := strings.TrimSpace(line[start:end])
		fields = append(fields, field)
	}

	return fields
}

// GetColumnIndex returns the index of a column by name, or -1 if not found
func (t *TableData) GetColumnIndex(name string) int {
	name = strings.ToUpper(name)
	for i, h := range t.Headers {
		if strings.ToUpper(h) == name {
			return i
		}
	}
	return -1
}

// GetColumn returns all values in a column by name
func (t *TableData) GetColumn(name string) []string {
	idx := t.GetColumnIndex(name)
	if idx < 0 {
		return nil
	}

	var values []string
	for _, row := range t.Rows {
		if idx < len(row) {
			values = append(values, row[idx])
		} else {
			values = append(values, "")
		}
	}
	return values
}
