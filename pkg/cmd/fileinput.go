package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// readBodyFromFile reads a JSON body from a file path prefixed with @.
// "@-" reads from stdin, "@filename" reads from the file.
// Returns nil if the arg doesn't start with "@".
func readBodyFromFile(arg string) (map[string]any, error) {
	if !strings.HasPrefix(arg, "@") {
		return nil, nil
	}

	source := arg[1:]
	var data []byte
	var err error

	if source == "-" {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading from stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("reading file %q: %w", source, err)
		}
	}

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("parsing JSON from %q: %w", arg, err)
	}
	return body, nil
}

// checkFileArg checks if the first positional arg (after subcommand args) is
// a @file reference. Used by create/update commands that accept @file input.
// Returns the parsed body or nil if no @file arg was found.
func checkFileArg(args []string, startIdx int) (map[string]any, error) {
	if startIdx >= len(args) {
		return nil, nil
	}
	return readBodyFromFile(args[startIdx])
}
