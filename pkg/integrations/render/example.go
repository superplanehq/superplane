package render

import (
	_ "embed"
	"sync"

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

//go:embed example_output_rollback.json
var exampleOutputRollbackBytes []byte

//go:embed example_output_purge_cache.json
var exampleOutputPurgeCacheBytes []byte

//go:embed example_output_update_env_vars.json
var exampleOutputUpdateEnvVarsBytes []byte

var exampleDataOnDeployOnce sync.Once
var exampleDataOnDeploy map[string]any

var exampleDataOnBuildOnce sync.Once
var exampleDataOnBuild map[string]any

var exampleOutputDeployOnce sync.Once
var exampleOutputDeploy map[string]any

var exampleOutputGetServiceOnce sync.Once
var exampleOutputGetService map[string]any

var exampleOutputGetDeployOnce sync.Once
var exampleOutputGetDeploy map[string]any

var exampleOutputCancelDeployOnce sync.Once
var exampleOutputCancelDeploy map[string]any

var exampleOutputRollbackOnce sync.Once
var exampleOutputRollback map[string]any

var exampleOutputPurgeCacheOnce sync.Once
var exampleOutputPurgeCache map[string]any

var exampleOutputUpdateEnvVarsOnce sync.Once
var exampleOutputUpdateEnvVars map[string]any

func (t *OnDeploy) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnDeployOnce,
		exampleDataOnDeployBytes,
		&exampleDataOnDeploy,
	)
}

func (t *OnBuild) ExampleData() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleDataOnBuildOnce,
		exampleDataOnBuildBytes,
		&exampleDataOnBuild,
	)
}

func (c *Deploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputDeployOnce,
		exampleOutputDeployBytes,
		&exampleOutputDeploy,
	)
}

func (c *GetService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetServiceOnce,
		exampleOutputGetServiceBytes,
		&exampleOutputGetService,
	)
}

func (c *GetDeploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputGetDeployOnce,
		exampleOutputGetDeployBytes,
		&exampleOutputGetDeploy,
	)
}

func (c *CancelDeploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputCancelDeployOnce,
		exampleOutputCancelDeployBytes,
		&exampleOutputCancelDeploy,
	)
}

func (c *Rollback) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRollbackOnce,
		exampleOutputRollbackBytes,
		&exampleOutputRollback,
	)
}

func (c *PurgeCache) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPurgeCacheOnce,
		exampleOutputPurgeCacheBytes,
		&exampleOutputPurgeCache,
	)
}

func (c *UpdateEnvVars) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateEnvVarsOnce,
		exampleOutputUpdateEnvVarsBytes,
		&exampleOutputUpdateEnvVars,
	)
}
