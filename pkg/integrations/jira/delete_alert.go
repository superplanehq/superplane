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

const DeleteJiraAlertPayloadType = "jira.alert.deleted"

type DeleteAlert struct{}

type DeleteAlertSpec struct {
	Alert string `json:"alert" mapstructure:"alert"`
}

func (c *DeleteAlert) Name() string {
	return "jira.deleteAlert"
}

func (c *DeleteAlert) Label() string {
	return "Delete Alert"
}

func (c *DeleteAlert) Description() string {
	return "Delete a Jira Service Management Ops alert by id"
}

func (c *DeleteAlert) Documentation() string {
	return `The Delete Alert component removes an alert from Jira Service Management.

Deletion is processed asynchronously like other mutating Ops operations.

## Configuration

- **Alert**: Pick the Ops alert to delete from the integration resource list.

## Output

Includes the API acknowledgement (**requestId**, etc.) and **deleted**: true for workflow convenience.`
}

func (c *DeleteAlert) Icon() string {
	return "jira"
}

func (c *DeleteAlert) Color() string {
	return "red"
}

func (c *DeleteAlert) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteAlert) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "alert",
			Label:       "Alert",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Ops alert to delete (from List alerts)",
			Placeholder: "Select an alert",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "alert",
				},
			},
		},
	}
}

func (c *DeleteAlert) Setup(ctx core.SetupContext) error {
	spec := DeleteAlertSpec{}
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

func (c *DeleteAlert) Execute(ctx core.ExecutionContext) error {
	spec := DeleteAlertSpec{}
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
	apiResp, err := client.DeleteOpsAlert(cloudID, strings.TrimSpace(spec.Alert))
	if err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}
	payload := map[string]any{
		"deleted":   true,
		"alertId":   strings.TrimSpace(spec.Alert),
		"requestId": apiResp.RequestID,
		"result":    apiResp.Result,
	}
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		DeleteJiraAlertPayloadType,
		[]any{payload},
	)
}

func (c *DeleteAlert) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteAlert) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteAlert) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteAlert) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteAlert) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteAlert) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
