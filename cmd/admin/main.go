package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/admincli"
)

func main() {
	if err := admincli.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
