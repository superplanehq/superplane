package render

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_event.json
var exampleDataOnEventBytes []byte

//go:embed example_output_trigger_deploy.json
var exampleOutputTriggerDeployBytes []byte

var exampleDataOnEventOnce sync.Once
var exampleDataOnEvent map[string]any

var exampleOutputTriggerDeployOnce sync.Once
var exampleOutputTriggerDeploy map[string]any

func (t *OnDeploy) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnEventOnce, exampleDataOnEventBytes, &exampleDataOnEvent)
}

func (t *OnBuild) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleDataOnEventOnce, exampleDataOnEventBytes, &exampleDataOnEvent)
}

func (c *TriggerDeploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputTriggerDeployOnce,
		exampleOutputTriggerDeployBytes,
		&exampleOutputTriggerDeploy,
	)
}
