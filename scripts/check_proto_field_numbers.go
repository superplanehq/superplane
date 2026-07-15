package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/lint/protofields"
)

func main() {
	rootDir := flag.String("root", protofields.DefaultRootDir, "directory containing .proto files")
	flag.Parse()

	issues, err := protofields.Run(*rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "proto field number lint failed: %v\n", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Printf("Proto message field numbers are contiguous in %s.\n", *rootDir)
		return
	}

	for _, issue := range issues {
		fmt.Fprintln(os.Stderr, issue.String())
	}
	fmt.Fprintf(os.Stderr, "\n%s\n", protofields.Guidance)
	os.Exit(1)
}
