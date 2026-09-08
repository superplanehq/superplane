package main

import (
	"context"
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/registryimports"
	"github.com/superplanehq/superplane/pkg/seed"
)

var _ = registryimports.Loaded

const seedDatabaseName = "superplane_dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if name := os.Getenv("DB_NAME"); name != seedDatabaseName {
		return fmt.Errorf("make seed only writes to %s (DB_NAME=%s)", seedDatabaseName, name)
	}

	deps, err := seed.NewLiveDependencies()
	if err != nil {
		return err
	}

	result, err := seed.Run(context.Background(), deps)
	if err != nil {
		return err
	}

	seed.WriteSummary(os.Stdout, result)
	return nil
}
