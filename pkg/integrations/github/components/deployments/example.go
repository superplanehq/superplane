package deployments

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed payloads/create_deployment.json
var exampleCreateDeploymentBytes []byte

var exampleCreateDeploymentOnce sync.Once
var exampleCreateDeployment map[string]any

func (c *CreateDeployment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleCreateDeploymentOnce,
		exampleCreateDeploymentBytes,
		&exampleCreateDeployment,
	)
}

//go:embed payloads/create_deployment_status.json
var exampleCreateDeploymentStatusBytes []byte

var exampleCreateDeploymentStatusOnce sync.Once
var exampleCreateDeploymentStatus map[string]any

func (c *CreateDeploymentStatus) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleCreateDeploymentStatusOnce,
		exampleCreateDeploymentStatusBytes,
		&exampleCreateDeploymentStatus,
	)
}
