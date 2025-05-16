package main

import (
	"log"
	"os"

	"github.com/superplanehq/superplane/fixtures"
)

func main() {
	log.Println("Seeding database with test data...")

	// Check if environment is development
	env := os.Getenv("APP_ENV")
	if env != "development" && env != "" {
		log.Fatalf("Seeding is only allowed in development environment. Current environment: %s", env)
	}

	// Load the test data
	if err := fixtures.SeedTestData(); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	log.Println("Successfully seeded database with test data!")
}
