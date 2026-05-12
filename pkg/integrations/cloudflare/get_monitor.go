package cloudflare

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

const GetMonitorPayloadType = "cloudflare.monitor.fetched"

type GetMonitor struct{}

type GetMonitorSpec struct {
	Monitor string `json:"monitor"`
}

func (c *GetMonitor) Name() string {
	return "cloudflare.getMonitor"
}

func (c *GetMonitor) Label() string {
	return "Get Monitor"
}

func (c *GetMonitor) Description() string {
	return "Retrieve a Cloudflare load balancing health monitor's configuration"
}

func (c *GetMonitor) Documentation() string {
	return `The Get Monitor component fetches the current configuration of a Cloudflare Load Balancing health monitor.

## Use Cases

- **Pre-flight validation**: Confirm a monitor exists and inspect its configuration before modifying it
- **Audit**: Capture a snapshot of monitor settings at a point in time
- **Workflow data**: Expose monitor configuration as payload data for downstream nodes

## Configuration

- **Monitor**: The health monitor to retrieve

## Output

Returns the full monitor configuration including type, path, port, intervals, and health thresholds.`
}

func (c *GetMonitor) Icon() string {
	return "activity"
}

func (c *GetMonitor) Color() string {
	return "orange"
}

func (c *GetMonitor) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetMonitor) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "monitor",
			Label:       "Monitor",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The load balancing health monitor to retrieve",
			Placeholder: "Select a monitor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "monitor",
				},
			},
		},
	}
}

func (c *GetMonitor) Setup(ctx core.SetupContext) error {
	spec := GetMonitorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	monitorID := strings.TrimSpace(spec.Monitor)
	if monitorID == "" {
		return errors.New("monitor is required")
	}

	return resolveMonitorMetadata(ctx, monitorID, nil)
}

func (c *GetMonitor) Execute(ctx core.ExecutionContext) error {
	spec := GetMonitorSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	monitorID := strings.TrimSpace(spec.Monitor)
	if monitorID == "" {
		return errors.New("monitor is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	accountID, err := accountIDForIntegration(ctx.Integration)
	if err != nil {
		return err
	}

	monitor, err := client.GetMonitor(accountID, monitorID)
	if err != nil {
		return fmt.Errorf("failed to get monitor: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetMonitorPayloadType,
		[]any{map[string]any{
			"accountId": accountID,
			"monitor":   monitor,
			"monitorId": monitorID,
		}},
	)
}

func (c *GetMonitor) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetMonitor) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetMonitor) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetMonitor) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetMonitor) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetMonitor) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
