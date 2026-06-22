package cloudsql

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_database.json
var exampleOutputCreateDatabaseBytes []byte

//go:embed example_output_get_database.json
var exampleOutputGetDatabaseBytes []byte

//go:embed example_output_delete_database.json
var exampleOutputDeleteDatabaseBytes []byte

//go:embed example_output_create_instance.json
var exampleOutputCreateInstanceBytes []byte

//go:embed example_output_get_instance.json
var exampleOutputGetInstanceBytes []byte

//go:embed example_output_delete_instance.json
var exampleOutputDeleteInstanceBytes []byte

var (
	exampleOutputCreateDatabaseOnce sync.Once
	exampleOutputCreateDatabase     map[string]any

	exampleOutputGetDatabaseOnce sync.Once
	exampleOutputGetDatabase     map[string]any

	exampleOutputDeleteDatabaseOnce sync.Once
	exampleOutputDeleteDatabase     map[string]any

	exampleOutputCreateInstanceOnce sync.Once
	exampleOutputCreateInstance     map[string]any

	exampleOutputGetInstanceOnce sync.Once
	exampleOutputGetInstance     map[string]any

	exampleOutputDeleteInstanceOnce sync.Once
	exampleOutputDeleteInstance     map[string]any
)

func (c *CreateDatabase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateDatabaseOnce, exampleOutputCreateDatabaseBytes, &exampleOutputCreateDatabase)
}

func (g *GetDatabase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetDatabaseOnce, exampleOutputGetDatabaseBytes, &exampleOutputGetDatabase)
}

func (d *DeleteDatabase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteDatabaseOnce, exampleOutputDeleteDatabaseBytes, &exampleOutputDeleteDatabase)
}

func (c *CreateInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateInstanceOnce, exampleOutputCreateInstanceBytes, &exampleOutputCreateInstance)
}

func (g *GetInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetInstanceOnce, exampleOutputGetInstanceBytes, &exampleOutputGetInstance)
}

func (d *DeleteInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteInstanceOnce, exampleOutputDeleteInstanceBytes, &exampleOutputDeleteInstance)
}
