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

	// Initialize the fixtures loader with minimal options
	fixtures, err = testfixtures.New(
		testfixtures.Database(sqlDB),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory(filepath.Join(basepath, "yaml")),
		testfixtures.DangerousSkipTestDatabaseCheck(),
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
func SeedTestData() error {
	// Truncate all tables before loading fixtures
	if err := database.TruncateTables(); err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	// Load the fixtures
	if err := Load(); err != nil {
		return fmt.Errorf("failed to load fixtures: %w", err)
	}

	log.Println("Seed data has been successfully loaded into the database")

	return nil
}

// ClearTestData removes all seeded data from the database by truncating all tables
func ClearTestData() error {
	log.Println("Clearing all seeded data from the database...")
	
	// Truncate all tables to remove all data
	if err := database.TruncateTables(); err != nil {
		return fmt.Errorf("failed to clear seeded data: %w", err)
	}
	
	log.Println("All seeded data has been successfully cleared from the database")
	return nil
}
