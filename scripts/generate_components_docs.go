package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/docs"
)

func main() {
	if err := docs.WriteFiles("docs/components"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
