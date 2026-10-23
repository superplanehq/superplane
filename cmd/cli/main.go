package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli"
)

func main() {
	args := os.Args[1:]
	if cli.ShouldStartUpdateCheck(args) {
		cli.StartUpdateCheck(args)
	}
	err := cli.RootCmd.Execute()
	cli.PrintUpdateNotice()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
