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

const UpdateInstancePayloadType = "oci.updateInstance"

type UpdateInstance struct{}

type UpdateInstanceSpec struct {
	InstanceID  string   `json:"instanceId" mapstructure:"instanceId"`
	DisplayName string   `json:"displayName" mapstructure:"displayName"`
	OCPUs       *float64 `json:"ocpus" mapstructure:"ocpus"`
	MemoryInGBs *float64 `json:"memoryInGBs" mapstructure:"memoryInGBs"`
}

func (c *UpdateInstance) Name() string {
	return "oci.updateInstance"
}

func (c *UpdateInstance) Label() string {
	return "Update Instance"
}

func (c *UpdateInstance) Description() string {
	return "Update mutable OCI Compute instance attributes"
}

func (c *UpdateInstance) Documentation() string {
	return `The Update Instance component updates mutable attributes on an Oracle Cloud Infrastructure Compute instance.

## Use Cases

- **Rename instances**: Update the display name for operational clarity
- **Resize flex shapes**: Adjust OCPUs or memory for supported flexible shapes
- **Post-provisioning changes**: Apply instance settings after a workflow decides the desired capacity

## Configuration

- **Instance**: The OCI Compute instance to update.
- **Display Name**: Optional new display name.
- **OCPUs**: Optional OCPU count for flexible shapes.
- **Memory (GB)**: Optional memory size for flexible shapes.

## Output

Emits the updated instance details on the default output channel, including:
- ` + "`instanceId`" + ` — instance OCID
- ` + "`displayName`" + ` — instance display name
- ` + "`lifecycleState`" + ` — current lifecycle state
- ` + "`shape`" + ` — the instance shape
- ` + "`availabilityDomain`" + ` — the availability domain
- ` + "`compartmentId`" + ` — the compartment OCID
- ` + "`region`" + ` — the region
- ` + "`timeCreated`" + ` — ISO-8601 creation timestamp
`
}

func (c *UpdateInstance) Icon() string {
	return "oci"
}

func (c *UpdateInstance) Color() string {
	return "red"
}

func (c *UpdateInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateInstance) ExampleOutput() map[string]any {
	return exampleOutputUpdateInstance()
}

func (c *UpdateInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Compute instance to update",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
		{
			Name:        "displayName",
			Label:       "Display Name",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "New display name for the instance",
			Placeholder: "my-instance",
		},
		{
			Name:        "ocpus",
			Label:       "OCPUs",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Number of OCPUs for a flexible shape",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
				},
			},
		},
		{
			Name:        "memoryInGBs",
			Label:       "Memory (GB)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Memory in GB for a flexible shape",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { v := 1; return &v }(),
				},
			},
		},
	}
}

func (c *UpdateInstance) Setup(ctx core.SetupContext) error {
	spec := UpdateInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.InstanceID) == "" {
		return errors.New("instanceId is required")
	}
	return nil
}

func (c *UpdateInstance) Execute(ctx core.ExecutionContext) error {
	spec := UpdateInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	req := UpdateInstanceRequest{}
	if strings.TrimSpace(spec.DisplayName) != "" {
		req.DisplayName = &spec.DisplayName
	}
	if spec.OCPUs != nil || spec.MemoryInGBs != nil {
		req.ShapeConfig = &InstanceShapeConfig{
			OCPUs:       spec.OCPUs,
			MemoryInGBs: spec.MemoryInGBs,
		}
	}

	instance, err := client.UpdateInstance(spec.InstanceID, req)
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UpdateInstancePayloadType, []any{instanceToMap(instance)})
}

func (c *UpdateInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateInstance) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *UpdateInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
