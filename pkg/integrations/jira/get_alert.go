package jira

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetJiraAlertPayloadType = "jira.alert.fetched"

type GetAlert struct{}

type GetAlertSpec struct {
	Alert string `json:"alert" mapstructure:"alert"`
}

func (c *GetAlert) Name() string {
	return "jira.getAlert"
}

func (c *GetAlert) Label() string {
	return "Get Alert"
}

func (c *GetAlert) Description() string {
	return "Fetch a Jira Service Management Ops alert by alert id"
}

func (c *GetAlert) Documentation() string {
	return `The Get Alert component returns full alert details from Jira Service Management.

## Configuration

- **Alert**: Pick a recent Ops alert from the integration resource list (same as Update / Delete Alert).

## Output

Payload type **jira.alert.fetched** with the alert object (message, status, priority, responders, tags, etc.).`
}

func (c *GetAlert) Icon() string {
	return "jira"
}

func (c *GetAlert) Color() string {
	return "blue"
}

func (c *GetAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alert",
			Label:       "Alert",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Ops alerts from List alerts (refresh the picker after new alerts appear)",
			Placeholder: "Select an alert",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "alert",
				},
			},
		},
	}
}

func (c *GetAlert) Setup(ctx core.SetupContext) error {
	spec := GetAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}
	alertKey := strings.TrimSpace(spec.Alert)
	if alertKey == "" {
		return fmt.Errorf("alert is required")
	}

	if ctx.HTTP != nil {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		if row, gerr := client.GetOpsAlert(cloudID, alertKey); gerr == nil {
			label := opsAlertIntegrationResourceLabel(row, alertKey)
			if err := ctx.Metadata.Set(OpsAlertPickerMetadata{AlertLabel: label}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *GetAlert) Execute(ctx core.ExecutionContext) error {
	spec := GetAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	cloudID, err := cloudIDFromIntegration(ctx.Integration)
	if err != nil {
		return err
	}
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	alert, err := client.GetOpsAlert(cloudID, strings.TrimSpace(spec.Alert))
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		GetJiraAlertPayloadType,
		[]any{alert},
	)
}

func (c *GetAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetAlert) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
