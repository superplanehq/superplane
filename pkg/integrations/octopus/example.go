package octopus

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_deployment_event.json
var exampleDataOnDeploymentEventBytes []byte

//go:embed example_output_deploy_release.json
var exampleOutputDeployReleaseBytes []byte
var exampleDataOnDeploymentEvent = utils.NewEmbeddedJSON(exampleDataOnDeploymentEventBytes)
var exampleOutputDeployRelease = utils.NewEmbeddedJSON(exampleOutputDeployReleaseBytes)

func (t *OnDeploymentEvent) ExampleData() map[string]any {
	return exampleDataOnDeploymentEvent.Value()
}

func (c *DeployRelease) ExampleOutput() map[string]any {
	return exampleOutputDeployRelease.Value()
}
