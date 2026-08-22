package vercel

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_deploy.json
var exampleOutputDeployBytes []byte

//go:embed example_output_get_deployment.json
var exampleOutputGetDeploymentBytes []byte

//go:embed example_output_list_deployments.json
var exampleOutputListDeploymentsBytes []byte

//go:embed example_output_cancel_deployment.json
var exampleOutputCancelDeploymentBytes []byte

//go:embed example_output_rollback.json
var exampleOutputRollbackBytes []byte

var exampleOutputDeployOnce sync.Once
var exampleOutputDeploy map[string]any

var exampleOutputGetDeploymentOnce sync.Once
var exampleOutputGetDeployment map[string]any

var exampleOutputListDeploymentsOnce sync.Once
var exampleOutputListDeployments map[string]any

var exampleOutputCancelDeploymentOnce sync.Once
var exampleOutputCancelDeployment map[string]any

var exampleOutputRollbackOnce sync.Once
var exampleOutputRollback map[string]any

func (c *TriggerDeployment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeployOnce,
		exampleOutputDeployBytes,
		&exampleOutputDeploy,
	)
}

func (c *GetDeployment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetDeploymentOnce,
		exampleOutputGetDeploymentBytes,
		&exampleOutputGetDeployment,
	)
}

func (c *ListDeployments) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputListDeploymentsOnce,
		exampleOutputListDeploymentsBytes,
		&exampleOutputListDeployments,
	)
}

func (c *CancelDeployment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCancelDeploymentOnce,
		exampleOutputCancelDeploymentBytes,
		&exampleOutputCancelDeployment,
	)
}

func (c *RollbackProduction) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRollbackOnce,
		exampleOutputRollbackBytes,
		&exampleOutputRollback,
	)
}
