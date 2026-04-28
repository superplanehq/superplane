package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli"
)

func main() {
	if cli.ShouldStartUpdateCheck(os.Args[1:]) {
		cli.StartUpdateCheck()
	}

	if err := cli.RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	cli.PrintUpdateNotice()
}
