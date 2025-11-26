package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// TableWriter provides simple table formatting for CLI output
type TableWriter struct {
	w       io.Writer
	headers []string
	widths  []int
	rows    [][]string
}

// NewTableWriter creates a new table writer
func NewTableWriter(w io.Writer, headers ...string) *TableWriter {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &TableWriter{
		w:       w,
		headers: headers,
		widths:  widths,
	}
}

// AddRow adds a row to the table
func (t *TableWriter) AddRow(cells ...string) {
	// Pad or truncate to match header count
	row := make([]string, len(t.headers))
	for i := range row {
		if i < len(cells) {
			row[i] = cells[i]
		}
		if len(row[i]) > t.widths[i] {
			t.widths[i] = len(row[i])
		}
	}
	t.rows = append(t.rows, row)
}

// Render outputs the table
func (t *TableWriter) Render() {
	// Print headers
	for i, h := range t.headers {
		fmt.Fprintf(t.w, "%-*s", t.widths[i]+2, h)
	}
	fmt.Fprintln(t.w)

	// Print rows
	for _, row := range t.rows {
		for i, cell := range row {
			fmt.Fprintf(t.w, "%-*s", t.widths[i]+2, cell)
		}
		fmt.Fprintln(t.w)
	}
}

// FormatTimeAgo formats a time as "X ago" string
func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// TruncateID truncates an ID to 12 characters (like Docker)
func TruncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// TruncateString truncates a string to max length with ellipsis
func TruncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// GenerateInstanceName generates a name from image reference
func GenerateInstanceName(image string) string {
	// Extract image name without registry/tag
	name := image

	// Remove registry prefix
	if idx := strings.LastIndex(name, "/"); idx != -1 {
		name = name[idx+1:]
	}

	// Remove tag/digest
	if idx := strings.Index(name, ":"); idx != -1 {
		name = name[:idx]
	}
	if idx := strings.Index(name, "@"); idx != -1 {
		name = name[:idx]
	}

	// Add random suffix
	suffix := randomSuffix(4)
	return fmt.Sprintf("%s-%s", name, suffix)
}

// randomSuffix generates a random alphanumeric suffix
func randomSuffix(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		// Simple pseudo-random using time
		b[i] = chars[(time.Now().UnixNano()+int64(i))%int64(len(chars))]
	}
	return string(b)
}


