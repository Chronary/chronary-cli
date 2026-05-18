package main

import (
	"fmt"
	"os"

	"github.com/Chronary/chronary-cli/pkg/client"
	"github.com/Chronary/chronary-cli/pkg/cmd"
)

// version is set at build time via -ldflags. Propagated into pkg/client.Version
// so the API can attribute traffic to the CLI surface (chronary-cli/<version>
// User-Agent), without the release pipeline needing to set two ldflags targets.
var version = "dev"

func main() {
	client.Version = version
	rootCmd := cmd.NewRootCmd(version)
	if err := rootCmd.Execute(); err != nil {
		// SilenceErrors is set on the root command so Cobra doesn't print its
		// own "Error: …" + usage banner; we print here so users (and the
		// validate-surfaces parity runner that greps stderr) actually see why
		// the command failed.
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
