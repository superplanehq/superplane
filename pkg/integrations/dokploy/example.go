package dokploy

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_list_applications.json
var exampleOutputListApplicationsBytes []byte

//go:embed example_output_deploy_application.json
var exampleOutputDeployApplicationBytes []byte

var (
	exampleOutputListApplicationsOnce sync.Once
	exampleOutputListApplications     map[string]any

	exampleOutputDeployApplicationOnce sync.Once
	exampleOutputDeployApplication     map[string]any
)

func (d *ListApplications) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListApplicationsOnce, exampleOutputListApplicationsBytes, &exampleOutputListApplications)
}

func (d *DeployApplication) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeployApplicationOnce, exampleOutputDeployApplicationBytes, &exampleOutputDeployApplication)
}
