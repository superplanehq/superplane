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

//go:embed example_output_rollback_deploy.json
var exampleOutputRollbackDeployBytes []byte

//go:embed example_output_purge_cache.json
var exampleOutputPurgeCacheBytes []byte

//go:embed example_output_scale_service.json
var exampleOutputScaleServiceBytes []byte

//go:embed example_output_update_env_var.json
var exampleOutputUpdateEnvVarBytes []byte

//go:embed example_output_add_custom_domain.json
var exampleOutputAddCustomDomainBytes []byte

//go:embed example_output_remove_custom_domain.json
var exampleOutputRemoveCustomDomainBytes []byte

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

var exampleOutputRollbackDeployOnce sync.Once
var exampleOutputRollbackDeploy map[string]any

var exampleOutputPurgeCacheOnce sync.Once
var exampleOutputPurgeCache map[string]any

var exampleOutputScaleServiceOnce sync.Once
var exampleOutputScaleService map[string]any

var exampleOutputUpdateEnvVarOnce sync.Once
var exampleOutputUpdateEnvVar map[string]any

var exampleOutputAddCustomDomainOnce sync.Once
var exampleOutputAddCustomDomain map[string]any

var exampleOutputRemoveCustomDomainOnce sync.Once
var exampleOutputRemoveCustomDomain map[string]any

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

func (c *RollbackDeploy) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRollbackDeployOnce,
		exampleOutputRollbackDeployBytes,
		&exampleOutputRollbackDeploy,
	)
}

func (c *PurgeCache) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputPurgeCacheOnce,
		exampleOutputPurgeCacheBytes,
		&exampleOutputPurgeCache,
	)
}

func (c *ScaleService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputScaleServiceOnce,
		exampleOutputScaleServiceBytes,
		&exampleOutputScaleService,
	)
}

func (c *UpdateEnvVar) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputUpdateEnvVarOnce,
		exampleOutputUpdateEnvVarBytes,
		&exampleOutputUpdateEnvVar,
	)
}

func (c *AddCustomDomain) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputAddCustomDomainOnce,
		exampleOutputAddCustomDomainBytes,
		&exampleOutputAddCustomDomain,
	)
}

func (c *RemoveCustomDomain) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(
		&exampleOutputRemoveCustomDomainOnce,
		exampleOutputRemoveCustomDomainBytes,
		&exampleOutputRemoveCustomDomain,
	)
}

func (c *ListDeploys) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "render.deploys.listed",
		"data": map[string]any{
			"serviceId": "srv-cukouhrtq21c73e9scng",
			"count":     1,
			"deploys": []map[string]any{
				{"deployId": "dep-cukp0k3tq21c73e9sct0", "serviceId": "srv-cukouhrtq21c73e9scng", "status": "live"},
			},
			"latestSuccessful": map[string]any{"deployId": "dep-cukp0k3tq21c73e9sct0", "status": "live"},
		},
	}
}

func (c *GetMetrics) ExampleOutput() map[string]any {
	return map[string]any{
		"type": "render.metrics",
		"data": map[string]any{
			"resources": []string{"srv-cukouhrtq21c73e9scng"},
			"summaries": map[string]any{
				"cpu":    map[string]any{"latest": 72.1, "avg": 58.4, "max": 84.2, "unit": "%"},
				"memory": map[string]any{"latest": 68.9, "avg": 61.3, "max": 79.7, "unit": "%"},
			},
		},
	}
}
