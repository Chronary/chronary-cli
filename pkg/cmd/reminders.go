package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// intListValue is a comma-separated list of whole minutes for --reminders and
// --default-reminders. Unlike pflag's IntSlice it accepts an empty value, so
// `--reminders=` means "no reminders" and is sent as [] (#1994). A repeated
// flag appends, as IntSlice did.
type intListValue struct {
	values  []int
	changed bool
}

func (v *intListValue) Set(raw string) error {
	parsed := []int{}
	if strings.TrimSpace(raw) != "" {
		for _, item := range strings.Split(raw, ",") {
			item = strings.TrimSpace(item)
			n, err := strconv.Atoi(item)
			if err != nil {
				return fmt.Errorf("invalid reminder offset %q: must be a whole number of minutes", item)
			}
			parsed = append(parsed, n)
		}
	}
	if v.changed {
		v.values = append(v.values, parsed...)
	} else {
		v.values = parsed
		v.changed = true
	}
	return nil
}

func (v *intListValue) String() string {
	parts := make([]string, len(v.values))
	for i, n := range v.values {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

func (v *intListValue) Type() string { return "ints" }

// addRemindersFlag registers a reminders list flag on cmd.
func addRemindersFlag(cmd *cobra.Command, name, usage string) {
	cmd.Flags().Var(&intListValue{}, name, usage)
}

// remindersFlag returns the parsed list and whether the flag was given. An
// empty value yields a non-nil empty slice, which encodes as [].
func remindersFlag(cmd *cobra.Command, name string) ([]int, bool) {
	f := cmd.Flags().Lookup(name)
	if f == nil || !f.Changed {
		return nil, false
	}
	v, ok := f.Value.(*intListValue)
	if !ok {
		return nil, false
	}
	if v.values == nil {
		return []int{}, true
	}
	return v.values, true
}
