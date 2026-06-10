package cloudsql

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_database.json
var exampleOutputCreateBytes []byte

//go:embed example_output_get_database.json
var exampleOutputGetBytes []byte

//go:embed example_output_delete_database.json
var exampleOutputDeleteBytes []byte

var (
	exampleOutputCreateOnce sync.Once
	exampleOutputCreate     map[string]any

	exampleOutputGetOnce sync.Once
	exampleOutputGet     map[string]any

	exampleOutputDeleteOnce sync.Once
	exampleOutputDelete     map[string]any
)

func (c *CreateDatabase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateOnce, exampleOutputCreateBytes, &exampleOutputCreate)
}

func (g *GetDatabase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetOnce, exampleOutputGetBytes, &exampleOutputGet)
}

func (d *DeleteDatabase) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteOnce, exampleOutputDeleteBytes, &exampleOutputDelete)
}
