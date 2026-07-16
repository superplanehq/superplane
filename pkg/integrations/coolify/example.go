package coolify

import (
	_ "embed"

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
	exampleOutputListApplications   = utils.NewEmbeddedJSON(exampleOutputListApplicationsBytes)
	exampleOutputListServices       = utils.NewEmbeddedJSON(exampleOutputListServicesBytes)
	exampleOutputControlApplication = utils.NewEmbeddedJSON(exampleOutputControlApplicationBytes)
	exampleOutputControlService     = utils.NewEmbeddedJSON(exampleOutputControlServiceBytes)
	exampleOutputDeployApplication  = utils.NewEmbeddedJSON(exampleOutputDeployApplicationBytes)
)

func (c *ListApplications) ExampleOutput() map[string]any {
	return exampleOutputListApplications.Value()
}

func (c *ListServices) ExampleOutput() map[string]any {
	return exampleOutputListServices.Value()
}

func (c *ControlApplication) ExampleOutput() map[string]any {
	return exampleOutputControlApplication.Value()
}

func (c *ControlService) ExampleOutput() map[string]any {
	return exampleOutputControlService.Value()
}

func (c *DeployApplication) ExampleOutput() map[string]any {
	return exampleOutputDeployApplication.Value()
}
