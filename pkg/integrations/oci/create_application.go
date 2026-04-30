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
	Compartment  string    `json:"compartment" mapstructure:"compartment"`
	DisplayName  string    `json:"displayName" mapstructure:"displayName"`
	Vcn          string    `json:"vcn" mapstructure:"vcn"`
	Subnet       string    `json:"subnet" mapstructure:"subnet"`
	Shape        string    `json:"shape" mapstructure:"shape"`
	FreeformTags []TagItem `json:"freeformTags" mapstructure:"freeformTags"`
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
			Name:        "compartment",
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
			Name:        "vcn",
			Label:       "VCN",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Virtual Cloud Network for the application",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeVCN,
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
		{
			Name:        "subnet",
			Label:       "Subnet",
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
								Field: "compartment",
							},
						},
						{
							Name: "vcnId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "vcn",
							},
						},
					},
				},
			},
		},
		{
			Name:        "shape",
			Label:       "Shape",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "The processor architecture for the application (defaults to GENERIC_X86)",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Generic x86", Value: "GENERIC_X86"},
						{Label: "Generic ARM", Value: "GENERIC_ARM"},
						{Label: "Generic x86 & ARM", Value: "GENERIC_X86_ARM"},
					},
				},
			},
		},
		{
			Name:        "freeformTags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Free-form tags (key-value pairs) to apply to the application",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "key", Label: "Key", Type: configuration.FieldTypeString, Required: true},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true},
						},
					},
				},
			},
		},
	}
}

type CreateApplicationNodeMetadata struct {
	SubnetID   string `json:"subnetId" mapstructure:"subnetId"`
	SubnetName string `json:"subnetName" mapstructure:"subnetName"`
	Shape      string `json:"shape" mapstructure:"shape"`
}

func (c *CreateApplication) Setup(ctx core.SetupContext) error {
	spec := CreateApplicationSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(spec.Compartment) == "" {
		return errors.New("compartment is required")
	}
	if strings.TrimSpace(spec.DisplayName) == "" {
		return errors.New("displayName is required")
	}
	if strings.TrimSpace(spec.Vcn) == "" {
		return errors.New("vcn is required")
	}
	if strings.TrimSpace(spec.Subnet) == "" {
		return errors.New("subnet is required")
	}

	return resolveCreateApplicationMetadata(ctx, spec)
}

func resolveSubnetName(ctx core.SetupContext, spec CreateApplicationSpec) string {
	if strings.Contains(spec.Subnet, "{{") {
		return spec.Subnet
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return spec.Subnet
	}

	subnets, err := client.ListSubnets(spec.Compartment, spec.Vcn)
	if err != nil {
		return spec.Subnet
	}

	for _, sn := range subnets {
		if sn.ID == spec.Subnet {
			return sn.DisplayName
		}
	}

	return spec.Subnet
}

func resolveCreateApplicationMetadata(ctx core.SetupContext, spec CreateApplicationSpec) error {
	// Return early if the subnet name is already cached for this subnet.
	var existing CreateApplicationNodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil &&
		existing.SubnetID == spec.Subnet && existing.SubnetName != "" {
		// Update shape in case it changed.
		existing.Shape = spec.Shape
		return ctx.Metadata.Set(existing)
	}

	subnetName := resolveSubnetName(ctx, spec)

	return ctx.Metadata.Set(CreateApplicationNodeMetadata{
		SubnetID:   spec.Subnet,
		SubnetName: subnetName,
		Shape:      spec.Shape,
	})
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
	subnetIDs := []string{spec.Subnet}

	app, err := client.CreateApplication(spec.Compartment, spec.DisplayName, spec.Shape, subnetIDs, tagItemsToMap(spec.FreeformTags))
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
