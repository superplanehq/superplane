package jenkins

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_trigger_build.json
var exampleOutputTriggerBuildBytes []byte

var exampleOutputTriggerBuildOnce sync.Once
var exampleOutputTriggerBuild map[string]any

func (t *TriggerBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTriggerBuildOnce, exampleOutputTriggerBuildBytes, &exampleOutputTriggerBuild)
}
