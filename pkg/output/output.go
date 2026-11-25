package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Print dispatches to JSON, YAML, or table formatting.
func Print(format string, data any, tableDef *TableDef, noColor bool) {
	switch format {
	case "json":
		PrintJSON(data)
	case "yaml":
		PrintYAML(data)
	default:
		if tableDef != nil {
			RenderTable(*tableDef, noColor)
		} else {
			PrintJSON(data)
		}
	}
}

// PrintJSON prints data as pretty-printed JSON to stdout.
func PrintJSON(data any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

// PrintYAML prints data as YAML to stdout. The data goes through JSON first,
// so keys follow the `json` tags (yaml.v3 ignores them and would print
// lowercased Go field names) and a json.RawMessage prints as the value it
// holds rather than a list of byte values (#2738).
func PrintYAML(data any) {
	body, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: encoding YAML: %v\n", err)
		return
	}
	PrintRawYAML(body)
}

// PrintRawJSON pretty-prints a JSON document (an API response body) to stdout
// as it is: every key, value and key order the API sent is kept (#2738). A body
// that is not valid JSON is printed unchanged.
func PrintRawJSON(body []byte) {
	writeRawJSON(os.Stdout, body)
}

// PrintRawYAML prints a JSON document (an API response body) to stdout as
// YAML, keeping the API's keys, key order and values (#2738). A body that is
// not valid JSON is printed unchanged.
func PrintRawYAML(body []byte) {
	writeRawYAML(os.Stdout, body)
}

func writeRawJSON(w io.Writer, body []byte) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, bytes.TrimSpace(body), "", "  "); err != nil {
		_, _ = w.Write(body)
		return
	}
	buf.WriteByte('\n')
	_, _ = w.Write(buf.Bytes())
}

func writeRawYAML(w io.Writer, body []byte) {
	node, err := jsonToYAMLNode(body)
	if err != nil {
		_, _ = w.Write(body)
		return
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	_ = enc.Encode(node)
	_ = enc.Close()
}

// jsonToYAMLNode converts a JSON document into a yaml.Node tree, keeping key
// order and the exact text of every number.
func jsonToYAMLNode(body []byte) (*yaml.Node, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	node, err := decodeYAMLNode(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing data after JSON value")
	}
	return node, nil
}

// strNode is a string scalar, double-quoted whenever a bare (or block) form
// could be read back as something else.
func strNode(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if needsDoubleQuotes(s) {
		n.Style = yaml.DoubleQuotedStyle
	}
	return n
}

// YAML 1.1 resolves these plain scalars to non-strings. yaml.v3 quotes only
// what YAML 1.2 would misread, so `17:00`, `yes` or `NO` came out bare and
// PyYAML and other 1.1 parsers read them as 1020 or a bool. The patterns over-
// approximate on purpose: quoting a string that needed none is harmless.
var (
	yaml11Bools = map[string]bool{
		"y": true, "n": true, "yes": true, "no": true, "on": true, "off": true,
		"true": true, "false": true, "null": true, "~": true, "": true,
	}
	yaml11NonString = []*regexp.Regexp{
		// ints: binary, octal, decimal, hex (with _ separators)
		regexp.MustCompile(`^[-+]?(0b[0-1_]+|0o?[0-7_]+|[0-9][0-9_]*|0x[0-9a-fA-F_]+)$`),
		// floats, including exponent forms without a dot
		regexp.MustCompile(`^[-+]?([0-9][0-9_]*)?\.?[0-9_]*([eE][-+]?[0-9]+)?$`),
		// base 60 (sexagesimal) ints and floats: 17:00, -10:00, 1:30:00.5
		regexp.MustCompile(`^[-+]?[0-9][0-9_]*(:[0-5]?[0-9])+(\.[0-9_]*)?$`),
		// infinity and not-a-number
		regexp.MustCompile(`^[-+]?\.(inf|Inf|INF)$`),
		regexp.MustCompile(`^\.(nan|NaN|NAN)$`),
		// timestamps and dates
		regexp.MustCompile(`^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}`),
	}
)

func needsDoubleQuotes(s string) bool {
	// Line breaks and a leading tab don't survive a literal block or a plain
	// or single-quoted scalar; double quotes escape them.
	if strings.ContainsAny(s, "\n\r  \u0085") || strings.HasPrefix(s, "\t") {
		return true
	}
	if yaml11Bools[strings.ToLower(s)] || s == "=" || s == "<<" {
		return true
	}
	for _, re := range yaml11NonString {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

func decodeYAMLNode(dec *json.Decoder) (*yaml.Node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("unexpected object key %v", keyTok)
				}
				val, err := decodeYAMLNode(dec)
				if err != nil {
					return nil, err
				}
				// Keys follow the same quoting rule as values (`<<` included:
				// bare, it is a merge key).
				n.Content = append(n.Content, strNode(key), val)
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, err
			}
			return n, nil
		case '[':
			n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for dec.More() {
				val, err := decodeYAMLNode(dec)
				if err != nil {
					return nil, err
				}
				n.Content = append(n.Content, val)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return n, nil
		}
		return nil, fmt.Errorf("unexpected delimiter %v", v)
	case string:
		return strNode(v), nil
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(v.String(), ".eE") {
			tag = "!!float"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: v.String()}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprintf("%t", v)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}
