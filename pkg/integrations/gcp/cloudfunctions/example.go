package cloudfunctions

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_invoke_function.json
var exampleOutputInvokeFunctionBytes []byte

var exampleOutputInvokeFunctionOnce sync.Once
var exampleOutputInvokeFunction map[string]any

func (c *InvokeFunction) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputInvokeFunctionOnce, exampleOutputInvokeFunctionBytes, &exampleOutputInvokeFunction)
}
