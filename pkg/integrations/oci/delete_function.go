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

const DeleteFunctionPayloadType = "oci.functionDeleted"

type DeleteFunction struct{}

type DeleteFunctionSpec struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	ApplicationID string `json:"applicationId" mapstructure:"applicationId"`
	FunctionID    string `json:"functionId" mapstructure:"functionId"`
}

func (d *DeleteFunction) Name() string {
	return "oci.deleteFunction"
}

func (d *DeleteFunction) Label() string {
	return "Delete Function"
}

func (d *DeleteFunction) Description() string {
	return "Remove a function from an OCI Functions application"
}

func (d *DeleteFunction) Documentation() string {
	return `The Delete Function component removes a function from an Oracle Cloud Infrastructure Functions application.

## Configuration

- **Compartment**: The compartment containing the application
- **Application**: The application that owns the function
- **Function**: The function to delete

## Output

Emits the deleted function ID on the default output channel:
- ` + "`functionId`" + ` — OCID of the deleted function
`
}

func (d *DeleteFunction) Icon() string {
	return "oci"
}

func (d *DeleteFunction) Color() string {
	return "red"
}

func (d *DeleteFunction) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteFunction) ExampleOutput() map[string]any {
	return map[string]any{
		"type": DeleteFunctionPayloadType,
		"data": map[string]any{
			"functionId": "ocid1.fnfunc.oc1.eu-frankfurt-1.aaaaExample1234567890abcdefghijklmnopqrstuvwxyz",
		},
	}
}

func (d *DeleteFunction) Configuration() []configuration.Field {
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
			Description: "The function to delete",
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
	}
}

func (d *DeleteFunction) Setup(ctx core.SetupContext) error {
	spec := DeleteFunctionSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.FunctionID) == "" {
		return errors.New("functionId is required")
	}

	return nil
}

func (d *DeleteFunction) Execute(ctx core.ExecutionContext) error {
	spec := DeleteFunctionSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	if err := client.DeleteFunction(spec.FunctionID); err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}

	payload := map[string]any{
		"functionId": spec.FunctionID,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteFunctionPayloadType, []any{payload})
}

func (d *DeleteFunction) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteFunction) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (d *DeleteFunction) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (d *DeleteFunction) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteFunction) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteFunction) Cleanup(_ core.SetupContext) error {
	return nil
}
