package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/lint/grpcactionsstatus"
)

func main() {
	violations, err := grpcactionsstatus.Scan(grpcactionsstatus.DefaultRootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "grpc actions status lint failed: %v\n", err)
		os.Exit(1)
	}

	if len(violations) == 0 {
		return
	}

	for _, violation := range violations {
		fmt.Fprintln(os.Stderr, violation.String())
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", grpcactionsstatus.Guidance)
	os.Exit(1)
}
