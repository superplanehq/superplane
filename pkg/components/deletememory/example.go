package deletememory

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output.json
var exampleOutputBytes []byte

var exampleOutputOnce sync.Once
var parsedExampleOutput map[string]any

func exampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputBytes, &parsedExampleOutput)
}
