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
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	ApplicationID string `json:"applicationId" mapstructure:"applicationId"`
}

func (d *DeleteApplication) Name() string {
	return "oci.deleteApplication"
}

func (d *DeleteApplication) Label() string {
	return "Delete Application"
}

func (d *DeleteApplication) Description() string {
	return "Delete an OCI Functions application and all of its functions"
}

func (d *DeleteApplication) Documentation() string {
	return `The Delete Application component removes an Oracle Cloud Infrastructure Functions application and all functions it contains.

## Configuration

- **Compartment**: The compartment containing the application
- **Application**: The application to delete

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
			Description: "The application to delete",
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

	if strings.TrimSpace(spec.CompartmentID) == "" {
		return errors.New("compartmentId is required")
	}

	if strings.TrimSpace(spec.ApplicationID) == "" {
		return errors.New("applicationId is required")
	}

	return resolveDeleteApplicationMetadata(ctx, spec.ApplicationID)
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		// Non-fatal: fall back to showing the ID.
		return ctx.Metadata.Set(DeleteApplicationNodeMetadata{
			ApplicationID:   applicationID,
			ApplicationName: applicationID,
		})
	}

	app, err := client.GetApplication(applicationID)
	if err != nil {
		// Non-fatal: fall back to showing the ID.
		return ctx.Metadata.Set(DeleteApplicationNodeMetadata{
			ApplicationID:   applicationID,
			ApplicationName: applicationID,
		})
	}

	name := app.DisplayName
	if name == "" {
		name = applicationID
	}

	return ctx.Metadata.Set(DeleteApplicationNodeMetadata{
		ApplicationID:   applicationID,
		ApplicationName: name,
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

	app, err := client.GetApplication(spec.ApplicationID)
	if err != nil {
		return fmt.Errorf("failed to get application: %w", err)
	}

	// OCI rejects application deletion when functions still exist (409 Conflict).
	// Enumerate and delete all child functions first.
	functions, err := client.ListFunctions(spec.ApplicationID)
	if err != nil {
		return fmt.Errorf("failed to list functions in application: %w", err)
	}
	for _, fn := range functions {
		if err := client.DeleteFunction(fn.ID); err != nil {
			return fmt.Errorf("failed to delete function %q: %w", fn.ID, err)
		}
	}

	if err := client.DeleteApplication(spec.ApplicationID); err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	payload := map[string]any{
		"applicationId": spec.ApplicationID,
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
