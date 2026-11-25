package output

import (
	"strings"
	"testing"
	"unicode/utf8"

	xansi "github.com/charmbracelet/x/ansi"
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

// Truncate measures display width, not bytes, so non-ASCII text is never cut
// mid-character or too early (#2722).
func TestTruncateNonASCII(t *testing.T) {
	// 20 x "é" (40 bytes) used to come out as "ééé\xc3...".
	got := Truncate(strings.Repeat("é", 20), 10)
	assert.True(t, utf8.ValidString(got), "result must be valid UTF-8: %q", got)
	assert.Equal(t, strings.Repeat("é", 7)+"...", got)

	// A 40-character French title cut at 30 keeps 27 characters plus "..."
	// (the byte cut kept 24).
	french := "Réunion d'équipe hebdomadaire élargie àé"
	assert.Equal(t, 40, utf8.RuneCountInString(french))
	got = Truncate(french, 30)
	assert.True(t, utf8.ValidString(got))
	assert.Equal(t, 30, xansi.StringWidth(got))
	assert.Equal(t, 30, utf8.RuneCountInString(got))

	// 12 Japanese characters are 36 bytes but 24 columns: they fit in 30.
	jp := "週次チームミーティング会"
	assert.Equal(t, 12, utf8.RuneCountInString(jp))
	assert.Equal(t, jp, Truncate(jp, 30))

	// Wide text that doesn't fit stays within the width and valid.
	got = Truncate(strings.Repeat("会", 30), 30)
	assert.True(t, utf8.ValidString(got))
	assert.LessOrEqual(t, xansi.StringWidth(got), 30)

	// A ZWJ emoji is kept whole or dropped whole, never split.
	family := "\U0001F468‍\U0001F469‍\U0001F467"
	got = Truncate("abcdefg"+family+family, 10)
	assert.True(t, utf8.ValidString(got))
	assert.LessOrEqual(t, xansi.StringWidth(got), 10)
	assert.False(t, strings.Contains(got, "‍...") || strings.HasSuffix(strings.TrimSuffix(got, "..."), "‍"))
}

func TestTableDefEmpty(t *testing.T) {
	// Should not panic with empty rows
	def := TableDef{
		Headers: []string{"ID", "Name"},
		Rows:    [][]string{},
	}
	RenderTable(def, true) // prints "No results."
}
