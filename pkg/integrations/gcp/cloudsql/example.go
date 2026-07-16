package cloudsql

import (
	_ "embed"

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
	exampleOutputCreateDatabase = utils.NewEmbeddedJSON(exampleOutputCreateDatabaseBytes)
	exampleOutputGetDatabase    = utils.NewEmbeddedJSON(exampleOutputGetDatabaseBytes)
	exampleOutputDeleteDatabase = utils.NewEmbeddedJSON(exampleOutputDeleteDatabaseBytes)
	exampleOutputCreateInstance = utils.NewEmbeddedJSON(exampleOutputCreateInstanceBytes)
	exampleOutputGetInstance    = utils.NewEmbeddedJSON(exampleOutputGetInstanceBytes)
	exampleOutputDeleteInstance = utils.NewEmbeddedJSON(exampleOutputDeleteInstanceBytes)
)

func (c *CreateDatabase) ExampleOutput() map[string]any {
	return exampleOutputCreateDatabase.Value()
}

func (g *GetDatabase) ExampleOutput() map[string]any {
	return exampleOutputGetDatabase.Value()
}

func (d *DeleteDatabase) ExampleOutput() map[string]any {
	return exampleOutputDeleteDatabase.Value()
}

func (c *CreateInstance) ExampleOutput() map[string]any {
	return exampleOutputCreateInstance.Value()
}

func (g *GetInstance) ExampleOutput() map[string]any {
	return exampleOutputGetInstance.Value()
}

func (d *DeleteInstance) ExampleOutput() map[string]any {
	return exampleOutputDeleteInstance.Value()
}
