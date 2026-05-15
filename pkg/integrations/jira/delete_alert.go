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
	AlertID string `json:"alertId" mapstructure:"alertId"`
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
	return `The Delete Alert component removes an alert via the [Jira Service Management Ops Alerts REST API](https://developer.atlassian.com/cloud/jira/service-desk-ops/rest/v2/api-group-alerts/).

Deletion is processed asynchronously like other mutating Ops operations.

## Configuration

- **Alert id**: The Ops alert id to delete.

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
			Name:        "alertId",
			Label:       "Alert ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jira Ops alert id to delete",
		},
	}
}

func (c *DeleteAlert) Setup(ctx core.SetupContext) error {
	spec := DeleteAlertSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if _, err := cloudIDFromIntegration(ctx.Integration); err != nil {
		return err
	}
	if strings.TrimSpace(spec.AlertID) == "" {
		return fmt.Errorf("alertId is required")
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
	apiResp, err := client.DeleteOpsAlert(cloudID, spec.AlertID)
	if err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
	}
	payload := map[string]any{
		"deleted":   true,
		"alertId":   strings.TrimSpace(spec.AlertID),
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
