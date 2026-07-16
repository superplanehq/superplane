package lambda

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_function.json
var exampleOutputRunFunctionBytes []byte
var exampleOutputRunFunction = utils.NewEmbeddedJSON(exampleOutputRunFunctionBytes)

func (c *RunFunction) ExampleOutput() map[string]any {
	return exampleOutputRunFunction.Value()
}
