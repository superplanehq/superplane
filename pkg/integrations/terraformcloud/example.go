package terraformcloud

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_trigger_run.json
var exampleOutputTriggerRunBytes []byte

//go:embed example_data_on_run_completed.json
var exampleDataOnRunCompletedBytes []byte

var exampleOutputTriggerRunOnce sync.Once
var exampleOutputTriggerRunData map[string]any

var exampleDataOnRunCompletedOnce sync.Once
var exampleDataOnRunCompletedData map[string]any

func (c *TriggerRun) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputTriggerRunOnce,
		exampleOutputTriggerRunBytes,
		&exampleOutputTriggerRunData,
	)
}

func exampleDataOnRunCompleted() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnRunCompletedOnce,
		exampleDataOnRunCompletedBytes,
		&exampleDataOnRunCompletedData,
	)
}
