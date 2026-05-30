package plugin

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output.json
var exampleOutputBytes []byte

var exampleOutputOnce sync.Once
var exampleOutput map[string]any

func (r *RunAction) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputOnce, exampleOutputBytes, &exampleOutput)
}
