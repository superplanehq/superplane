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
	err := cli.RootCmd.Execute()
	cli.PrintUpdateNotice()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
