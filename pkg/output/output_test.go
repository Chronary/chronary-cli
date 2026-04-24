package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorStatus(t *testing.T) {
	// With color enabled, output should still contain the status text
	// (may or may not have ANSI codes depending on terminal detection)
	active := ColorStatus("active", false)
	assert.Contains(t, active, "active")

	paused := ColorStatus("paused", false)
	assert.Contains(t, paused, "paused")

	decom := ColorStatus("decommissioned", false)
	assert.Contains(t, decom, "decommissioned")
}

func TestColorStatusNoColor(t *testing.T) {
	assert.Equal(t, "active", ColorStatus("active", true))
	assert.Equal(t, "paused", ColorStatus("paused", true))
	assert.Equal(t, "decommissioned", ColorStatus("decommissioned", true))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", Truncate("hello", 10))
	assert.Equal(t, "hello w...", Truncate("hello world foo", 10))
	assert.Equal(t, "", Truncate("", 10))
}

func TestTruncateExactLength(t *testing.T) {
	assert.Equal(t, "1234567890", Truncate("1234567890", 10))
}

func TestTableDefEmpty(t *testing.T) {
	// Should not panic with empty rows
	def := TableDef{
		Headers: []string{"ID", "Name"},
		Rows:    [][]string{},
	}
	RenderTable(def, true) // prints "No results."
}
