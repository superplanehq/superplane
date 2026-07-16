package deletememory

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output.json
var exampleOutputBytes []byte
var parsedExampleOutput = utils.NewEmbeddedJSON(exampleOutputBytes)

func exampleOutput() map[string]any {
	return parsedExampleOutput.Value()
}
