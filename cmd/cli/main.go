package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli"
)

func main() {
	cli.MaybeStartUpdateCheck(os.Args[1:])
	err := cli.RootCmd.Execute()
	cli.PrintUpdateNotice()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
