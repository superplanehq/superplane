package lambda

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_run_function.json
var exampleOutputRunFunctionBytes []byte

var exampleOutputRunFunctionOnce sync.Once
var exampleOutputRunFunction map[string]any

func (c *RunFunction) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRunFunctionOnce, exampleOutputRunFunctionBytes, &exampleOutputRunFunction)
}
