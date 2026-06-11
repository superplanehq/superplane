package cloudsql

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_create_instance.json
var exampleOutputCreateBytes []byte

//go:embed example_output_get_instance.json
var exampleOutputGetBytes []byte

//go:embed example_output_delete_instance.json
var exampleOutputDeleteBytes []byte

var (
	exampleOutputCreateOnce sync.Once
	exampleOutputCreate     map[string]any

	exampleOutputGetOnce sync.Once
	exampleOutputGet     map[string]any

	exampleOutputDeleteOnce sync.Once
	exampleOutputDelete     map[string]any
)

func (c *CreateInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCreateOnce, exampleOutputCreateBytes, &exampleOutputCreate)
}

func (g *GetInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetOnce, exampleOutputGetBytes, &exampleOutputGet)
}

func (d *DeleteInstance) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeleteOnce, exampleOutputDeleteBytes, &exampleOutputDelete)
}
