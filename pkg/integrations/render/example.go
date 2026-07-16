package render

import (
	_ "embed"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_data_on_deploy.json
var exampleDataOnDeployBytes []byte

//go:embed example_data_on_build.json
var exampleDataOnBuildBytes []byte

//go:embed example_output_deploy.json
var exampleOutputDeployBytes []byte

//go:embed example_output_get_service.json
var exampleOutputGetServiceBytes []byte

//go:embed example_output_get_deploy.json
var exampleOutputGetDeployBytes []byte

//go:embed example_output_cancel_deploy.json
var exampleOutputCancelDeployBytes []byte

//go:embed example_output_rollback_deploy.json
var exampleOutputRollbackDeployBytes []byte

//go:embed example_output_purge_cache.json
var exampleOutputPurgeCacheBytes []byte

//go:embed example_output_update_env_var.json
var exampleOutputUpdateEnvVarBytes []byte

//go:embed example_output_add_custom_domain.json
var exampleOutputAddCustomDomainBytes []byte

//go:embed example_output_remove_custom_domain.json
var exampleOutputRemoveCustomDomainBytes []byte
var exampleDataOnDeploy = utils.NewEmbeddedJSON(exampleDataOnDeployBytes)
var exampleDataOnBuild = utils.NewEmbeddedJSON(exampleDataOnBuildBytes)
var exampleOutputDeploy = utils.NewEmbeddedJSON(exampleOutputDeployBytes)
var exampleOutputGetService = utils.NewEmbeddedJSON(exampleOutputGetServiceBytes)
var exampleOutputGetDeploy = utils.NewEmbeddedJSON(exampleOutputGetDeployBytes)
var exampleOutputCancelDeploy = utils.NewEmbeddedJSON(exampleOutputCancelDeployBytes)
var exampleOutputRollbackDeploy = utils.NewEmbeddedJSON(exampleOutputRollbackDeployBytes)
var exampleOutputPurgeCache = utils.NewEmbeddedJSON(exampleOutputPurgeCacheBytes)
var exampleOutputUpdateEnvVar = utils.NewEmbeddedJSON(exampleOutputUpdateEnvVarBytes)
var exampleOutputAddCustomDomain = utils.NewEmbeddedJSON(exampleOutputAddCustomDomainBytes)
var exampleOutputRemoveCustomDomain = utils.NewEmbeddedJSON(exampleOutputRemoveCustomDomainBytes)

func (t *OnDeploy) ExampleData() map[string]any {
	return exampleDataOnDeploy.Value()
}

func (t *OnBuild) ExampleData() map[string]any {
	return exampleDataOnBuild.Value()
}

func (c *Deploy) ExampleOutput() map[string]any {
	return exampleOutputDeploy.Value()
}

func (c *GetService) ExampleOutput() map[string]any {
	return exampleOutputGetService.Value()
}

func (c *GetDeploy) ExampleOutput() map[string]any {
	return exampleOutputGetDeploy.Value()
}

func (c *CancelDeploy) ExampleOutput() map[string]any {
	return exampleOutputCancelDeploy.Value()
}

func (c *RollbackDeploy) ExampleOutput() map[string]any {
	return exampleOutputRollbackDeploy.Value()
}

func (c *PurgeCache) ExampleOutput() map[string]any {
	return exampleOutputPurgeCache.Value()
}

func (c *UpdateEnvVar) ExampleOutput() map[string]any {
	return exampleOutputUpdateEnvVar.Value()
}

func (c *AddCustomDomain) ExampleOutput() map[string]any {
	return exampleOutputAddCustomDomain.Value()
}

func (c *RemoveCustomDomain) ExampleOutput() map[string]any {
	return exampleOutputRemoveCustomDomain.Value()
}
