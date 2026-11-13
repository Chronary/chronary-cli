package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
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

// requireFlagsSet is cobra's required-flag check, run by hand. Commands that
// take a body from @file or @- can't mark body fields with MarkFlagRequired,
// because cobra checks those before RunE sees the @file arg (#1993). They call
// this on the flags path instead. The error matches cobra's text and name
// order; a flag counts as set when it was given, as in cobra.
func requireFlagsSet(cmd *cobra.Command, names ...string) error {
	var missing []string
	for _, name := range names {
		if !cmd.Flags().Changed(name) {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("required flag(s) \"%s\" not set", strings.Join(missing, `", "`))
}
