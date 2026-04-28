package restate

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_invoke_handler.json
var exampleOutputInvokeHandlerBytes []byte

var exampleOutputInvokeHandlerOnce sync.Once
var exampleOutputInvokeHandler map[string]any

func (c *InvokeHandler) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputInvokeHandlerOnce, exampleOutputInvokeHandlerBytes, &exampleOutputInvokeHandler)
}

//go:embed example_output_send_handler.json
var exampleOutputSendHandlerBytes []byte

var exampleOutputSendHandlerOnce sync.Once
var exampleOutputSendHandler map[string]any

func (c *SendHandler) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendHandlerOnce, exampleOutputSendHandlerBytes, &exampleOutputSendHandler)
}

//go:embed example_output_send_delayed_handler.json
var exampleOutputSendDelayedHandlerBytes []byte

var exampleOutputSendDelayedHandlerOnce sync.Once
var exampleOutputSendDelayedHandler map[string]any

func (c *SendDelayedHandler) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputSendDelayedHandlerOnce, exampleOutputSendDelayedHandlerBytes, &exampleOutputSendDelayedHandler)
}

//go:embed example_output_register_deployment.json
var exampleOutputRegisterDeploymentBytes []byte

var exampleOutputRegisterDeploymentOnce sync.Once
var exampleOutputRegisterDeployment map[string]any

func (c *RegisterDeployment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRegisterDeploymentOnce, exampleOutputRegisterDeploymentBytes, &exampleOutputRegisterDeployment)
}

//go:embed example_output_remove_deployment.json
var exampleOutputRemoveDeploymentBytes []byte

var exampleOutputRemoveDeploymentOnce sync.Once
var exampleOutputRemoveDeployment map[string]any

func (c *RemoveDeployment) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputRemoveDeploymentOnce, exampleOutputRemoveDeploymentBytes, &exampleOutputRemoveDeployment)
}

//go:embed example_output_get_service.json
var exampleOutputGetServiceBytes []byte

var exampleOutputGetServiceOnce sync.Once
var exampleOutputGetService map[string]any

func (c *GetService) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputGetServiceOnce, exampleOutputGetServiceBytes, &exampleOutputGetService)
}

//go:embed example_output_list_services.json
var exampleOutputListServicesBytes []byte

var exampleOutputListServicesOnce sync.Once
var exampleOutputListServices map[string]any

func (c *ListServices) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputListServicesOnce, exampleOutputListServicesBytes, &exampleOutputListServices)
}

//go:embed example_output_cancel_invocation.json
var exampleOutputCancelInvocationBytes []byte

var exampleOutputCancelInvocationOnce sync.Once
var exampleOutputCancelInvocation map[string]any

func (c *CancelInvocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputCancelInvocationOnce, exampleOutputCancelInvocationBytes, &exampleOutputCancelInvocation)
}

//go:embed example_output_kill_invocation.json
var exampleOutputKillInvocationBytes []byte

var exampleOutputKillInvocationOnce sync.Once
var exampleOutputKillInvocation map[string]any

func (c *KillInvocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputKillInvocationOnce, exampleOutputKillInvocationBytes, &exampleOutputKillInvocation)
}

//go:embed example_output_purge_invocation.json
var exampleOutputPurgeInvocationBytes []byte

var exampleOutputPurgeInvocationOnce sync.Once
var exampleOutputPurgeInvocation map[string]any

func (c *PurgeInvocation) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputPurgeInvocationOnce, exampleOutputPurgeInvocationBytes, &exampleOutputPurgeInvocation)
}

//go:embed example_output_health_check.json
var exampleOutputHealthCheckBytes []byte

var exampleOutputHealthCheckOnce sync.Once
var exampleOutputHealthCheck map[string]any

func (c *HealthCheck) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&exampleOutputHealthCheckOnce, exampleOutputHealthCheckBytes, &exampleOutputHealthCheck)
}
