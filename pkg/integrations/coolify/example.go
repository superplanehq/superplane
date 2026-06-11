package coolify

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_list_applications.json
var exampleOutputListApplicationsBytes []byte

//go:embed example_output_list_services.json
var exampleOutputListServicesBytes []byte

//go:embed example_output_control_application.json
var exampleOutputControlApplicationBytes []byte

//go:embed example_output_control_service.json
var exampleOutputControlServiceBytes []byte

//go:embed example_output_deploy_application.json
var exampleOutputDeployApplicationBytes []byte

var (
	exampleOutputListApplicationsOnce sync.Once
	exampleOutputListApplications     map[string]any

	exampleOutputListServicesOnce sync.Once
	exampleOutputListServices     map[string]any

	exampleOutputControlApplicationOnce sync.Once
	exampleOutputControlApplication     map[string]any

	exampleOutputControlServiceOnce sync.Once
	exampleOutputControlService     map[string]any

	exampleOutputDeployApplicationOnce sync.Once
	exampleOutputDeployApplication     map[string]any
)

func (c *ListApplications) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListApplicationsOnce, exampleOutputListApplicationsBytes, &exampleOutputListApplications)
}

func (c *ListServices) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListServicesOnce, exampleOutputListServicesBytes, &exampleOutputListServices)
}

func (c *ControlApplication) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputControlApplicationOnce, exampleOutputControlApplicationBytes, &exampleOutputControlApplication)
}

func (c *ControlService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputControlServiceOnce, exampleOutputControlServiceBytes, &exampleOutputControlService)
}

func (c *DeployApplication) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputDeployApplicationOnce, exampleOutputDeployApplicationBytes, &exampleOutputDeployApplication)
}
