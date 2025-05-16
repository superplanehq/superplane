package fixtures

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/go-testfixtures/testfixtures/v3"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
)

var fixtures *testfixtures.Loader

// Setup initializes the testfixtures loader with all the fixture files
func Setup(db *gorm.DB) error {
	// Get the SQL DB from the GORM DB
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB from GORM DB: %w", err)
	}

	// Find the root directory of the project
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	// Initialize the fixtures loader
	fixtures, err = testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory(filepath.Join(basepath, "yaml")),
	)
	if err != nil {
		return fmt.Errorf("failed to create fixtures loader: %w", err)
	}

	return nil
}

// Load loads all fixture data into the database
func Load() error {
	// If fixtures haven't been set up yet, set them up
	if fixtures == nil {
		if err := Setup(database.Conn()); err != nil {
			return err
		}
	}

	// Load the fixtures
	err := fixtures.Load()
	if err != nil {
		return fmt.Errorf("failed to load fixtures: %w", err)
	}

	return nil
}

// SeedTestData is a helper function to easily load test data in development or test environments
func SeedTestData() {
	// Truncate all tables before loading fixtures
	if err := database.TruncateTables(); err != nil {
		log.Fatalf("Failed to truncate tables: %v", err)
	}

	// Load the fixtures
	if err := Load(); err != nil {
		log.Fatalf("Failed to load fixtures: %v", err)
	}

	log.Println("Seed data has been successfully loaded into the database")
}
