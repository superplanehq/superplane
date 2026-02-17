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

var exampleDataOnDeployOnce sync.Once
var exampleDataOnDeploy map[string]any

var exampleDataOnBuildOnce sync.Once
var exampleDataOnBuild map[string]any

var exampleOutputDeployOnce sync.Once
var exampleOutputDeploy map[string]any

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
