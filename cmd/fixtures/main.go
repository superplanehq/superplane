package main

import (
	"log"
	"os"

	"github.com/superplanehq/superplane/fixtures"
)

func main() {
	// Check if environment is development
	env := os.Getenv("APP_ENV")
	if env != "development" && env != "" {
		log.Fatalf("Fixtures operations are only allowed in development environment. Current environment: %s", env)
	}

	// Check if we should only clear the data
	clearOnly := os.Getenv("CLEAR_ONLY") == "true"

	if clearOnly {
		log.Println("Clearing all fixture data from database...")
		
		// Clear the database data
		if err := fixtures.ClearTestData(); err != nil {
			log.Fatalf("Failed to clear test data: %v", err)
		}
		
		log.Println("Successfully cleared all fixture data!")
		return
	}

	log.Println("Seeding database with test data...")

	// Load the test data
	if err := fixtures.SeedTestData(); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	log.Println("Successfully seeded database with test data!")
}
