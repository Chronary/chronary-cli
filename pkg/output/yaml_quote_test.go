package output

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func rawYAML(t *testing.T, body string) string {
	t.Helper()
	var buf bytes.Buffer
	writeRawYAML(&buf, []byte(body))
	return buf.String()
}

// The YAML 1.1 type regexps from yaml.org/type (bool, null, int, float,
// timestamp, merge, value), used as an independent YAML 1.1 reading: a plain
// (unquoted) scalar that matches one would not be read back as a string.
var yaml11Resolvers = []*regexp.Regexp{
	regexp.MustCompile(`^(y|Y|yes|Yes|YES|n|N|no|No|NO|true|True|TRUE|false|False|FALSE|on|On|ON|off|Off|OFF)$`),
	regexp.MustCompile(`^(~|null|Null|NULL|)$`),
	regexp.MustCompile(`^[-+]?(0b[0-1_]+|0[0-7_]+|(0|[1-9][0-9_]*)|0x[0-9a-fA-F_]+|[1-9][0-9_]*(:[0-5]?[0-9])+)$`),
	regexp.MustCompile(`^([-+]?([0-9][0-9_]*)?\.[0-9.]*([eE][-+][0-9]+)?|[-+]?[0-9][0-9_]*(:[0-5]?[0-9])+\.[0-9_]*|[-+]?\.(inf|Inf|INF)|\.(nan|NaN|NAN))$`),
	regexp.MustCompile(`^[0-9][0-9][0-9][0-9]-[0-9][0-9]?-[0-9][0-9]?`),
	regexp.MustCompile(`^(<<|=)$`),
}

// plainScalars returns every plain-style scalar (keys and values) in a YAML
// document, as yaml.v3 parsed it.
func plainScalars(t *testing.T, doc string) []string {
	t.Helper()
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &root), doc)
	var out []string
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		if n.Kind == yaml.ScalarNode && n.Style == 0 {
			out = append(out, n.Value)
		}
		for _, c := range n.Content {
			walk(c)
		}
	}
	walk(&root)
	return out
}

// #2738 review: strings a YAML 1.1 parser (PyYAML and others) would read as
// a bool, number, null or date are double-quoted.
func TestRawYAMLQuotesYAML11NonStrings(t *testing.T) {
	cases := []string{
		"y", "Y", "n", "N", "yes", "Yes", "YES", "no", "No", "NO", "on", "ON", "off", "Off",
		"true", "False", "null", "NULL", "~", "",
		"17:00", "-10:00", "+05:30", "1:30:00.5",
		"0", "42", "-7", "1_000", "0x1F", "0o17", "017", "0b101",
		"1.5", "-0.5", ".5", "1e5", "1.0e+3", ".inf", "-.Inf", ".nan", ".NaN",
		"2026-11-24", "2026-11-24T09:00:00.000Z",
		"<<", "=",
	}
	for _, s := range cases {
		t.Run(s, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"v": s})
			require.NoError(t, err)
			assert.Equal(t, "v: "+strconv.Quote(s)+"\n", rawYAML(t, string(body)))
		})
	}
}

func TestRawYAMLLeavesOrdinaryStringsPlain(t *testing.T) {
	assert.Equal(t, "status: confirmed\nid: evt_1\ntitle: Weekly sync\n",
		rawYAML(t, `{"status":"confirmed","id":"evt_1","title":"Weekly sync"}`))
}

// Line breaks and a leading tab go in double quotes, escaped: a literal
// block drops a leading newline and rejects a tab-led first line, and YAML
// 1.1 folds U+0085 / U+2028 / U+2029 in other styles.
func TestRawYAMLDoubleQuotesLineBreaks(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"leading newline", "\nAgenda:\n- item", `d: "\nAgenda:\n- item"` + "\n"},
		{"inner newline", "a\nb", `d: "a\nb"` + "\n"},
		{"carriage return", "a\rb", `d: "a\rb"` + "\n"},
		{"leading tab", "\tTabbed", `d: "\tTabbed"` + "\n"},
		{"leading tab, multi-line", "\tTabbed\nsecond", `d: "\tTabbed\nsecond"` + "\n"},
		{"U+2028", "a b", `d: "a\Lb"` + "\n"},
		{"U+2029", "a b", `d: "a\Pb"` + "\n"},
		{"U+0085", "a\u0085b", `d: "a\Nb"` + "\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"d": tc.in})
			require.NoError(t, err)
			out := rawYAML(t, string(body))
			assert.Equal(t, tc.want, out)
			var got map[string]string
			require.NoError(t, yaml.Unmarshal([]byte(out), &got), out)
			assert.Equal(t, tc.in, got["d"])
		})
	}
}

// Keys follow the same rule; `<<` bare would be a merge key, which made the
// whole document unreadable.
func TestRawYAMLQuotesKeys(t *testing.T) {
	out := rawYAML(t, `{"metadata":{"<<":"x","yes":1,"on":"off","17:00":"a","plain":"b"}}`)
	assert.Equal(t, "metadata:\n  \"<<\": x\n  \"yes\": 1\n  \"on\": \"off\"\n  \"17:00\": a\n  plain: b\n", out)
	var got map[string]map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(out), &got), out)
	assert.Equal(t, "x", got["metadata"]["<<"])
	assert.Equal(t, "off", got["metadata"]["on"])
}

// Under the YAML 1.1 reading, no plain scalar in a realistic body resolves to
// a non-string unless the JSON value was that type.
func TestRawYAMLReadsAsStringsUnderYAML11(t *testing.T) {
	body := `{"working_hours":{"mon":[["09:00","17:00"]],"tue":[]},"utcOffset":"-10:00","country":"NO","flag":"on","n":3,"ok":true,"none":null,"title":"Réunion: \"x\"","<<":"merge","notes":"\nline"}`
	out := rawYAML(t, body)
	nonStrings := map[string]bool{"3": true, "true": true, "null": true}
	for _, s := range plainScalars(t, out) {
		if nonStrings[s] {
			continue
		}
		for _, re := range yaml11Resolvers {
			assert.False(t, re.MatchString(s), "plain scalar %q would not read as a string under YAML 1.1:\n%s", s, out)
		}
	}
	assert.True(t, strings.Contains(out, `- "17:00"`), out)
}
