package cloudfunctions

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_invoke_function.json
var exampleOutputInvokeFunctionBytes []byte
var exampleOutputInvokeFunction = utils.NewEmbeddedJSON(exampleOutputInvokeFunctionBytes)

func (c *InvokeFunction) ExampleOutput() map[string]any {
	return exampleOutputInvokeFunction.Value()
}
