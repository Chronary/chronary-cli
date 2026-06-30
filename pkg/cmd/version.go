package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			data := struct {
				Version string `json:"version"`
			}{Version: version}

			if printStructured(cmd, data) {
				return nil
			}

			fmt.Printf("chronary %s\n", version)
			return nil
		},
	}
}
