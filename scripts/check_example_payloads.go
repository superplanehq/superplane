package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/lint/examplepayloads"
)

func main() {
	issues, err := examplepayloads.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "example payload lint failed: %v\n", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		return
	}

	for _, issue := range issues {
		fmt.Fprintln(os.Stderr, issue.String())
	}

	os.Exit(1)
}
