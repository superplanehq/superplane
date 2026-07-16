package ifp

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output.json
var exampleOutputBytes []byte
var exampleOutput = utils.NewEmbeddedJSON(exampleOutputBytes)

func (f *If) ExampleOutput() map[string]any {
	return exampleOutput.Value()
}
