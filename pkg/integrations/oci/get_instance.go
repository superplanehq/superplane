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

const GetInstancePayloadType = "oci.getInstance"

type GetInstance struct{}

type GetInstanceSpec struct {
	InstanceID string `json:"instanceId" mapstructure:"instanceId"`
}

func (c *GetInstance) Name() string {
	return "oci.getInstance"
}

func (c *GetInstance) Label() string {
	return "Get Instance"
}

func (c *GetInstance) Description() string {
	return "Fetch details for an OCI Compute instance"
}

func (c *GetInstance) Documentation() string {
	return `The Get Instance component fetches the latest details for an Oracle Cloud Infrastructure Compute instance.

## Use Cases

- **Inventory checks**: Read instance state, shape, region, and network addresses during a workflow
- **Deployment gates**: Verify a target instance exists and is in the expected lifecycle state
- **Audit workflows**: Capture instance metadata before another operation runs

## Configuration

- **Instance**: The OCI Compute instance to fetch.

## Output

Emits the instance details on the default output channel, including:
- ` + "`instanceId`" + ` — instance OCID
- ` + "`displayName`" + ` — instance display name
- ` + "`lifecycleState`" + ` — current lifecycle state
- ` + "`publicIp`" + ` — public IP address, if assigned
- ` + "`privateIp`" + ` — primary private IP address, if available
- ` + "`shape`" + ` — the instance shape
- ` + "`availabilityDomain`" + ` — the availability domain
- ` + "`compartmentId`" + ` — the compartment OCID
- ` + "`region`" + ` — the region
- ` + "`timeCreated`" + ` — ISO-8601 creation timestamp
`
}

func (c *GetInstance) Icon() string {
	return "oci"
}

func (c *GetInstance) Color() string {
	return "red"
}

func (c *GetInstance) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetInstance) ExampleOutput() map[string]any {
	return exampleOutputGetInstance()
}

func (c *GetInstance) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "instanceId",
			Label:       "Instance",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Compute instance to fetch",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeInstance,
				},
			},
		},
	}
}

func (c *GetInstance) Setup(ctx core.SetupContext) error {
	spec := GetInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.InstanceID) == "" {
		return errors.New("instanceId is required")
	}
	return nil
}

func (c *GetInstance) Execute(ctx core.ExecutionContext) error {
	spec := GetInstanceSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	instance, err := client.GetInstance(spec.InstanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	payload := instanceToMap(instance)
	enrichInstanceWithVNICIPs(ctx.Logger, client, instance, payload)

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetInstancePayloadType, []any{payload})
}

func (c *GetInstance) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetInstance) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("unknown hook: %s", ctx.Name)
}

func (c *GetInstance) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetInstance) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetInstance) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetInstance) Cleanup(ctx core.SetupContext) error {
	return nil
}
