package jenkins

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_trigger_build.json
var exampleOutputTriggerBuildBytes []byte

//go:embed example_data_on_build_finished.json
var exampleDataOnBuildFinishedBytes []byte

var exampleOutputTriggerBuildOnce sync.Once
var exampleOutputTriggerBuild map[string]any

var exampleDataOnBuildFinishedOnce sync.Once
var exampleDataOnBuildFinished map[string]any

func (t *TriggerBuild) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputTriggerBuildOnce, exampleOutputTriggerBuildBytes, &exampleOutputTriggerBuild)
}

func (t *OnBuildFinished) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnBuildFinishedOnce, exampleDataOnBuildFinishedBytes, &exampleDataOnBuildFinished)
}
