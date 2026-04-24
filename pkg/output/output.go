package output

import (
	"encoding/json"
	"os"

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

// PrintYAML prints data as YAML to stdout.
func PrintYAML(data any) {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	enc.Encode(data)
}
