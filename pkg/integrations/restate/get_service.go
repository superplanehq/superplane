package restate

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetService struct{}

type GetServiceSpec struct {
	Service string `json:"service"`
}

func (c *GetService) Name() string {
	return "restate.getService"
}

func (c *GetService) Label() string {
	return "Get Service"
}

func (c *GetService) Description() string {
	return "Get details about a registered Restate service"
}

func (c *GetService) Icon() string {
	return "repeat"
}

func (c *GetService) Color() string {
	return "gray"
}

func (c *GetService) Documentation() string {
	return `The Get Service component retrieves details about a specific service registered with Restate.

## Use Cases

- **Pre-deploy validation**: Check that a service is registered before invoking it
- **Service inspection**: View service handlers, revision, and deployment info
- **Conditional workflows**: Branch workflow logic based on service properties

## Outputs

The component emits the full service details from Restate, including:
- ` + "`name`" + `: The service name
- ` + "`revision`" + `: The current revision number
- ` + "`ty`" + `: The service type (Service, VirtualObject, or Workflow)
- ` + "`deployment_id`" + `: The associated deployment ID
- ` + "`public`" + `: Whether the service is publicly accessible
- ` + "`handlers`" + `: Array of handler definitions
`
}

func (c *GetService) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetService) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "service",
			Label:       "Service Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the Restate service to inspect",
		},
	}
}

func (c *GetService) Setup(ctx core.SetupContext) error {
	spec := GetServiceSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Service == "" {
		return errors.New("service is required")
	}

	return nil
}

func (c *GetService) Execute(ctx core.ExecutionContext) error {
	spec := GetServiceSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	result, err := client.GetService(spec.Service)
	if err != nil {
		return fmt.Errorf("failed to get service: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"restate.service",
		[]any{result},
	)
}

func (c *GetService) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetService) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetService) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetService) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetService) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetService) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
