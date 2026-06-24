package modelstxdebt_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/superplanehq/superplane/pkg/lint/modelstxdebt"
)

func TestScanCountsInTransactionDefinitionsAndDatabaseConnCalls(t *testing.T) {
	rootDir := t.TempDir()
	writeFile(t, filepath.Join(rootDir, "example.go"), `package models

import "github.com/superplanehq/superplane/pkg/database"

func FindWidget(id string) (*Widget, error) {
	return FindWidgetInTransaction(database.Conn(), id)
}

func FindWidgetInTransaction(tx *gorm.DB, id string) (*Widget, error) {
	return nil, nil
}

func (w *Widget) SaveInTransaction(tx *gorm.DB) error {
	_ = database.Conn()
	return nil
}
`)

	result, err := modelstxdebt.Scan(rootDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got := result.InTransactionDefinitionCount(); got != 2 {
		t.Fatalf("InTransactionDefinitionCount() = %d, want 2", got)
	}

	if got := result.DatabaseConnCallCount(); got != 2 {
		t.Fatalf("DatabaseConnCallCount() = %d, want 2", got)
	}
}

func TestScanSkipsTestFiles(t *testing.T) {
	rootDir := t.TempDir()
	writeFile(t, filepath.Join(rootDir, "example_test.go"), `package models

import "github.com/superplanehq/superplane/pkg/database"

func TestExample(t *testing.T) {
	_ = database.Conn()
}

func FindWidgetInTransaction(tx *gorm.DB, id string) (*Widget, error) {
	return nil, nil
}
`)

	result, err := modelstxdebt.Scan(rootDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got := result.InTransactionDefinitionCount(); got != 0 {
		t.Fatalf("InTransactionDefinitionCount() = %d, want 0", got)
	}

	if got := result.DatabaseConnCallCount(); got != 0 {
		t.Fatalf("DatabaseConnCallCount() = %d, want 0", got)
	}
}

func TestScanRespectsDatabaseImportAlias(t *testing.T) {
	rootDir := t.TempDir()
	writeFile(t, filepath.Join(rootDir, "example.go"), `package models

import db "github.com/superplanehq/superplane/pkg/database"

func Example() {
	_ = db.Conn()
}
`)

	result, err := modelstxdebt.Scan(rootDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got := result.DatabaseConnCallCount(); got != 1 {
		t.Fatalf("DatabaseConnCallCount() = %d, want 1", got)
	}
}

func TestLocationKeyIgnoresLineNumber(t *testing.T) {
	rootDir := t.TempDir()
	path := filepath.Join(rootDir, "example.go")
	writeFile(t, path, `package models

import "github.com/superplanehq/superplane/pkg/database"

func FindWidget(id string) (*Widget, error) {
	return FindWidgetInTransaction(database.Conn(), id)
}

func FindWidgetInTransaction(tx *gorm.DB, id string) (*Widget, error) {
	return nil, nil
}
`)

	first, err := modelstxdebt.Scan(rootDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	writeFile(t, path, `package models

import "github.com/superplanehq/superplane/pkg/database"

// added comment shifts line numbers without changing debt

func FindWidget(id string) (*Widget, error) {
	return FindWidgetInTransaction(database.Conn(), id)
}

func FindWidgetInTransaction(tx *gorm.DB, id string) (*Widget, error) {
	return nil, nil
}
`)

	second, err := modelstxdebt.Scan(rootDir)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if first.InTransactionDefinitions[0].Key() != second.InTransactionDefinitions[0].Key() {
		t.Fatalf("InTransaction key changed after line shift: %q -> %q",
			first.InTransactionDefinitions[0].Key(),
			second.InTransactionDefinitions[0].Key(),
		)
	}

	if first.DatabaseConnCalls[0].Key() != second.DatabaseConnCalls[0].Key() {
		t.Fatalf("database.Conn() key changed after line shift: %q -> %q",
			first.DatabaseConnCalls[0].Key(),
			second.DatabaseConnCalls[0].Key(),
		)
	}
}

func TestGuidanceIsDocumented(t *testing.T) {
	if modelstxdebt.Guidance == "" {
		t.Fatal("Guidance must not be empty")
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writeFile(%q): %v", path, err)
	}
}
