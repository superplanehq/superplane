package oci

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const InvokeFunctionPayloadType = "oci.functionInvoked"

type InvokeFunction struct{}

type InvokeFunctionSpec struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	ApplicationID string `json:"applicationId" mapstructure:"applicationId"`
	FunctionID    string `json:"functionId" mapstructure:"functionId"`
	Payload       string `json:"payload" mapstructure:"payload"`
}

func (i *InvokeFunction) Name() string {
	return "oci.invokeFunction"
}

func (i *InvokeFunction) Label() string {
	return "Invoke Function"
}

func (i *InvokeFunction) Description() string {
	return "Execute an OCI Function and capture its response"
}

func (i *InvokeFunction) Documentation() string {
	return `The Invoke Function component executes an Oracle Cloud Infrastructure serverless function and emits its response.

## Use Cases

- **Data transformation**: Invoke a function to process or transform data mid-workflow
- **Custom logic**: Run arbitrary serverless code as a workflow step
- **Integration hooks**: Call a function to trigger external systems

## Configuration

- **Compartment**: The compartment containing the application
- **Application**: The application that owns the function
- **Function**: The function to invoke
- **Payload**: Optional JSON or plain-text body sent to the function

## Output

Emits the function response on the default output channel:
- ` + "`functionId`" + ` — OCID of the invoked function
- ` + "`statusCode`" + ` — HTTP status code returned by the function runtime
- ` + "`response`" + ` — raw response body as a string
`
}

func (i *InvokeFunction) Icon() string {
	return "oci"
}

func (i *InvokeFunction) Color() string {
	return "red"
}

func (i *InvokeFunction) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (i *InvokeFunction) ExampleOutput() map[string]any {
	return exampleOutputInvokeFunction()
}

func (i *InvokeFunction) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment containing the application",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
		{
			Name:        "applicationId",
			Label:       "Application",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The application that owns the function",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeFunctionApplication,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartmentId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "functionId",
			Label:       "Function",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The function to invoke",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeFunction,
					Parameters: []configuration.ParameterRef{
						{
							Name: "applicationId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "applicationId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "payload",
			Label:       "Payload",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Optional JSON or plain-text body to send to the function",
			Placeholder: `{"key": "value"}`,
		},
	}
}

func (i *InvokeFunction) Setup(ctx core.SetupContext) error {
	spec := InvokeFunctionSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.FunctionID) == "" {
		return errors.New("functionId is required")
	}

	return resolveInvokeFunctionMetadata(ctx, spec.ApplicationID, spec.FunctionID)
}

type InvokeFunctionNodeMetadata struct {
	ApplicationID   string `json:"applicationId" mapstructure:"applicationId"`
	ApplicationName string `json:"applicationName" mapstructure:"applicationName"`
	FunctionID      string `json:"functionId" mapstructure:"functionId"`
	FunctionName    string `json:"functionName" mapstructure:"functionName"`
}

func resolveInvokeFunctionMetadata(ctx core.SetupContext, applicationID, functionID string) error {
	// Return early if already cached.
	var existing InvokeFunctionNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil &&
		existing.ApplicationID == applicationID && existing.FunctionID == functionID && existing.FunctionName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	meta := InvokeFunctionNodeMetadata{
		ApplicationID: applicationID,
		FunctionID:    functionID,
	}

	if app, err := client.GetApplication(applicationID); err == nil && app.DisplayName != "" {
		meta.ApplicationName = app.DisplayName
	} else {
		meta.ApplicationName = applicationID
	}

	if fn, err := client.GetFunction(functionID); err == nil && fn.DisplayName != "" {
		meta.FunctionName = fn.DisplayName
	} else {
		meta.FunctionName = functionID
	}

	return ctx.Metadata.Set(meta)
}

func (i *InvokeFunction) Execute(ctx core.ExecutionContext) error {
	spec := InvokeFunctionSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	var payloadBytes []byte
	if strings.TrimSpace(spec.Payload) != "" {
		payloadBytes = []byte(spec.Payload)
	}

	respBody, statusCode, err := client.InvokeFunction(spec.FunctionID, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to invoke function: %w", err)
	}

	payload := map[string]any{
		"functionId": spec.FunctionID,
		"statusCode": statusCode,
		"response":   string(respBody),
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, InvokeFunctionPayloadType, []any{payload})
}

func (i *InvokeFunction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (i *InvokeFunction) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (i *InvokeFunction) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (i *InvokeFunction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (i *InvokeFunction) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (i *InvokeFunction) Cleanup(_ core.SetupContext) error {
	return nil
}
