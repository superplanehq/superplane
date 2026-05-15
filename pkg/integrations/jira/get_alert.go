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
	AlertID string `json:"alertId" mapstructure:"alertId"`
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
	return `The Get Alert component returns full alert details from the [Jira Service Management Ops Alerts REST API](https://developer.atlassian.com/cloud/jira/service-desk-ops/rest/v2/api-group-alerts/).

## Configuration

- **Alert id**: The Ops alert **id** (UUID) from Jira Service Management.

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
			Name:        "alertId",
			Label:       "Alert ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Jira Ops alert id",
			Placeholder: "e.g. e0caa0ce-d52f-4500-81b9-d592d06970b6",
		},
	}
}

func (c *GetAlert) Setup(ctx core.SetupContext) error {
	spec := GetAlertSpec{}
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
	alert, err := client.GetOpsAlert(cloudID, spec.AlertID)
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
