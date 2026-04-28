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

const CreateApplicationPayloadType = "oci.applicationCreated"

type CreateApplication struct{}

type CreateApplicationSpec struct {
	CompartmentID string `json:"compartmentId" mapstructure:"compartmentId"`
	DisplayName   string `json:"displayName" mapstructure:"displayName"`
	SubnetIDs     string `json:"subnetIds" mapstructure:"subnetIds"`
}

func (c *CreateApplication) Name() string {
	return "oci.createApplication"
}

func (c *CreateApplication) Label() string {
	return "Create Application"
}

func (c *CreateApplication) Description() string {
	return "Create a new OCI Functions application"
}

func (c *CreateApplication) Documentation() string {
	return `The Create Application component creates a new Oracle Cloud Infrastructure Functions application.

## Use Cases

- **Serverless environment provisioning**: Create application containers as part of a deployment workflow
- **Environment lifecycle management**: Automatically create application environments for each release

## Configuration

- **Compartment**: The compartment where the application will be created
- **Display Name**: A human-readable name for the application
- **Subnet OCIDs**: One or more subnet OCIDs (comma-separated) for the application's VCN configuration

## Output

Emits the created application details including:
- ` + "`applicationId`" + ` — application OCID
- ` + "`displayName`" + ` — application name
- ` + "`lifecycleState`" + ` — current state (` + "`ACTIVE`" + `)
- ` + "`compartmentId`" + ` — the compartment OCID
- ` + "`timeCreated`" + ` — ISO-8601 creation timestamp
`
}

func (c *CreateApplication) Icon() string {
	return "oci"
}

func (c *CreateApplication) Color() string {
	return "red"
}

func (c *CreateApplication) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateApplication) ExampleOutput() map[string]any {
	return exampleOutputCreateApplication()
}

func (c *CreateApplication) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "compartmentId",
			Label:       "Compartment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The compartment where the application will be created",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeCompartment,
				},
			},
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A human-readable name for the application",
			Placeholder: "my-functions-app",
		},
		{
			Name:        "subnetIds",
			Label:       "Subnet OCIDs",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Subnet for the application's VCN configuration",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeSubnet,
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

func (c *CreateApplication) Setup(ctx core.SetupContext) error {
	spec := CreateApplicationSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.CompartmentID) == "" {
		return errors.New("compartmentId is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if strings.TrimSpace(spec.SubnetIDs) == "" {
		return errors.New("subnetIds is required")
	}

	return nil
}

func (c *CreateApplication) Execute(ctx core.ExecutionContext) error {
	spec := CreateApplicationSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	// Subnet IDs are stored as a single OCID from the resource picker.
	subnetIDs := []string{spec.SubnetIDs}

	app, err := client.CreateApplication(spec.CompartmentID, spec.DisplayName, subnetIDs)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	payload := map[string]any{
		"applicationId":  app.ID,
		"displayName":    app.DisplayName,
		"compartmentId":  app.CompartmentID,
		"lifecycleState": app.LifecycleState,
		"timeCreated":    app.TimeCreated,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateApplicationPayloadType, []any{payload})
}

func (c *CreateApplication) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateApplication) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *CreateApplication) Cancel(_ core.ExecutionContext) error {
	return nil
}

func (c *CreateApplication) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateApplication) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateApplication) Cleanup(_ core.SetupContext) error {
	return nil
}
