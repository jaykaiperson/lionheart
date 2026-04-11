// lionheart v1.3 — CLI client/server
// Uses Cobra for CLI with server/client subcommands.
package main

import (
	"fmt"
	"os"

	"github.com/lionheart-vpn/lionheart/cmd/lionheart/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
