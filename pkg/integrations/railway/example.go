package railway

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_trigger_deploy.json
var exampleOutputTriggerDeployBytes []byte

//go:embed example_data_on_deployment_event.json
var exampleDataOnDeploymentEventBytes []byte

var exampleOutputTriggerDeployOnce sync.Once
var exampleOutputTriggerDeploy map[string]any

var exampleDataOnDeploymentEventOnce sync.Once
var exampleDataOnDeploymentEvent map[string]any

func (c *TriggerDeploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputTriggerDeployOnce,
		exampleOutputTriggerDeployBytes,
		&exampleOutputTriggerDeploy,
	)
}

func (t *OnDeploymentEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnDeploymentEventOnce,
		exampleDataOnDeploymentEventBytes,
		&exampleDataOnDeploymentEvent,
	)
}
