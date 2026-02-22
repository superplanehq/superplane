package octopus

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_deployment_event.json
var exampleDataOnDeploymentEventBytes []byte

//go:embed example_output_deploy_release.json
var exampleOutputDeployReleaseBytes []byte

var exampleDataOnDeploymentEventOnce sync.Once
var exampleDataOnDeploymentEvent map[string]any

var exampleOutputDeployReleaseOnce sync.Once
var exampleOutputDeployRelease map[string]any

func (t *OnDeploymentEvent) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnDeploymentEventOnce,
		exampleDataOnDeploymentEventBytes,
		&exampleDataOnDeploymentEvent,
	)
}

func (c *DeployRelease) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeployReleaseOnce,
		exampleOutputDeployReleaseBytes,
		&exampleOutputDeployRelease,
	)
}
