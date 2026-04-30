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

const DeleteApplicationPayloadType = "oci.applicationDeleted"

type DeleteApplication struct{}

type DeleteApplicationSpec struct {
	Compartment string `json:"compartment" mapstructure:"compartment"`
	Application string `json:"application" mapstructure:"application"`
}

func (d *DeleteApplication) Name() string {
	return "oci.deleteApplication"
}

func (d *DeleteApplication) Label() string {
	return "Delete Application"
}

func (d *DeleteApplication) Description() string {
	return "Delete an OCI Functions application"
}

func (d *DeleteApplication) Documentation() string {
	return `The Delete Application component removes an Oracle Cloud Infrastructure Functions application.

> **Important:** OCI will reject the deletion with a 409 Conflict if the application still contains functions.
> You must delete all functions inside the application (using the **Delete Function** component) before using this component.

## Configuration

- **Compartment**: The compartment containing the application
- **Application**: The application to delete. The application must have no remaining functions.

## Output

Emits the deleted application ID on the default output channel:
- ` + "`applicationId`" + ` — OCID of the deleted application
- ` + "`deleted`" + ` — boolean confirming deletion
- ` + "`displayName`" + ` — name of the deleted application
`
}

func (d *DeleteApplication) Icon() string {
	return "oci"
}

func (d *DeleteApplication) Color() string {
	return "red"
}

func (d *DeleteApplication) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteApplication) ExampleOutput() map[string]any {
	return exampleOutputDeleteApplication()
}

func (d *DeleteApplication) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartment",
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
			Name:        "application",
			Label:       "Application",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The application to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeFunctionApplication,
					Parameters: []configuration.ParameterRef{
						{
							Name: "compartmentId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "compartment",
							},
						},
					},
				},
			},
		},
	}
}

type DeleteApplicationNodeMetadata struct {
	ApplicationID   string `json:"applicationId" mapstructure:"applicationId"`
	ApplicationName string `json:"applicationName" mapstructure:"applicationName"`
}

func (d *DeleteApplication) Setup(ctx core.SetupContext) error {
	spec := DeleteApplicationSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Compartment) == "" {
		return errors.New("compartment is required")
	}

	if strings.TrimSpace(spec.Application) == "" {
		return errors.New("application is required")
	}

	return resolveDeleteApplicationMetadata(ctx, spec.Application)
}

func resolveDeleteApplicationMetadata(ctx core.SetupContext, applicationID string) error {
	// If it's an expression placeholder, store as-is.
	if strings.Contains(applicationID, "{{") {
		return ctx.Metadata.Set(DeleteApplicationNodeMetadata{ApplicationID: applicationID, ApplicationName: applicationID})
	}

	// Return early if already cached for this application.
	var existing DeleteApplicationNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil &&
		existing.ApplicationID == applicationID && existing.ApplicationName != "" {
		return nil
	}

	return ctx.Metadata.Set(DeleteApplicationNodeMetadata{
		ApplicationID:   applicationID,
		ApplicationName: resolveApplicationName(ctx, applicationID),
	})
}

func (d *DeleteApplication) Execute(ctx core.ExecutionContext) error {
	spec := DeleteApplicationSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	app, err := client.GetApplication(spec.Application)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	if err := client.DeleteApplication(spec.Application); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	payload := map[string]any{
		"applicationId": spec.Application,
		"displayName":   app.DisplayName,
		"deleted":       true,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, DeleteApplicationPayloadType, []any{payload})
}

func (d *DeleteApplication) Hooks() []core.Hook {
	return []core.Hook{}
}

func (d *DeleteApplication) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (d *DeleteApplication) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (d *DeleteApplication) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteApplication) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteApplication) Cleanup(_ core.SetupContext) error {
	return nil
}
