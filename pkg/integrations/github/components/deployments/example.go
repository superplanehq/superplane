package deployments

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/create_deployment.json
var exampleCreateDeploymentBytes []byte
var exampleCreateDeployment = utils.NewEmbeddedJSON(exampleCreateDeploymentBytes)

func (c *CreateDeployment) ExampleOutput() map[string]any {
	return exampleCreateDeployment.Value()
}

//go:embed payloads/create_deployment_status.json
var exampleCreateDeploymentStatusBytes []byte
var exampleCreateDeploymentStatus = utils.NewEmbeddedJSON(exampleCreateDeploymentStatusBytes)

func (c *CreateDeploymentStatus) ExampleOutput() map[string]any {
	return exampleCreateDeploymentStatus.Value()
}
