package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSnapshotListIDNotTruncated guards the snapshot list table: the ID column
// must keep its full width on a narrow terminal so the printed ID can be copied
// into `snapshot get`/`restore`. Truncatable columns (NAME, SOURCE) absorb the
// shrink instead.
func TestSnapshotListIDNotTruncated(t *testing.T) {
	os.Setenv("COLUMNS", "40")
	defer os.Unsetenv("COLUMNS")

	const fullID = "0123456789abcdef01234567" // 24-char snapshot ID

	var buf bytes.Buffer
	table := NewTableWriter(&buf, "ID", "NAME", "KIND", "SOURCE", "CREATED")
	table.TruncOrder = []int{1, 3}
	table.AddRow(fullID, "a-very-long-snapshot-name", "memory", "a-very-long-source-instance-name", "3 minutes ago")

	widths := table.renderWidths()
	assert.Equal(t, len(fullID), widths[0], "ID column should never be shrunk")

	table.Render()
	assert.True(t, strings.Contains(buf.String(), fullID), "rendered table should contain the full ID")
}
