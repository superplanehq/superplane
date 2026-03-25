package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/cli"
)

func main() {
	cli.StartUpdateCheck()
	err := cli.RootCmd.Execute()
	cli.PrintUpdateNotice()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
