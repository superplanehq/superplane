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

//go:embed example_output_get_project.json
var exampleOutputGetProjectBytes []byte

//go:embed example_output_create_project.json
var exampleOutputCreateProjectBytes []byte

//go:embed example_output_upsert_env_var.json
var exampleOutputUpsertEnvVarBytes []byte

var exampleOutputGetProjectOnce sync.Once
var exampleOutputGetProject map[string]any

var exampleOutputCreateProjectOnce sync.Once
var exampleOutputCreateProject map[string]any

var exampleOutputUpsertEnvVarOnce sync.Once
var exampleOutputUpsertEnvVar map[string]any

func (c *GetProject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetProjectOnce,
		exampleOutputGetProjectBytes,
		&exampleOutputGetProject,
	)
}

func (c *CreateProject) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCreateProjectOnce,
		exampleOutputCreateProjectBytes,
		&exampleOutputCreateProject,
	)
}

func (c *UpsertEnvVar) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpsertEnvVarOnce,
		exampleOutputUpsertEnvVarBytes,
		&exampleOutputUpsertEnvVar,
	)
}

//go:embed example_output_add_domain.json
var exampleOutputAddDomainBytes []byte

//go:embed example_output_remove_domain.json
var exampleOutputRemoveDomainBytes []byte

var exampleOutputAddDomainOnce sync.Once
var exampleOutputAddDomain map[string]any

var exampleOutputRemoveDomainOnce sync.Once
var exampleOutputRemoveDomain map[string]any

func (c *AddDomain) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputAddDomainOnce,
		exampleOutputAddDomainBytes,
		&exampleOutputAddDomain,
	)
}

func (c *RemoveDomain) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRemoveDomainOnce,
		exampleOutputRemoveDomainBytes,
		&exampleOutputRemoveDomain,
	)
}
