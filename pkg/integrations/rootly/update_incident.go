package rootly

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIncident struct{}

type UpdateIncidentSpec struct {
	IncidentID string `json:"incident_id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Severity   string `json:"severity"`
	Status     string `json:"status"`
}

func (c *UpdateIncident) Name() string {
	return "rootly.updateIncident"
}

func (c *UpdateIncident) Label() string {
	return "Update Incident"
}

func (c *UpdateIncident) Description() string {
	return "Update an existing incident in Rootly"
}

func (c *UpdateIncident) Documentation() string {
	return `The Update Incident component updates an existing incident in Rootly.

Use Cases
Sync status: Automatically update incident status from Jira or ServiceNow
Enrichment: Append new summary information from monitoring tools
Severity adjustment: Escalate severity based on new alerts

Configuration
Incident ID: The UUID of the incident to update (required)
Title: New title (optional)
Summary: New summary details (optional)
Severity: New severity level (optional)
Status: New incident status (optional)

Output
Returns the updated incident object.`
}

func (c *UpdateIncident) Icon() string {
	return "edit-3" // 用編輯圖示
}

func (c *UpdateIncident) Color() string {
	return "blue" // 改個顏色區分
}

func (c *UpdateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incident_id",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The UUID of the incident to update",
		},
		{
			Name:        "title",
			Label:       "New Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "New incident title",
		},
		{
			Name:        "summary",
			Label:       "New Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "New summary details",
		},
		{
			Name:        "status",
			Label:       "New Status",
			Type:        configuration.FieldTypeIntegrationResource, // 使用選單
			Required:    false,
			Description: "New incident status",
			Placeholder: "Select a status",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "status", // Rootly 的狀態資源
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "severity",
			Label:       "New Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "New severity level",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
	}
}

func (c *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incident_id is required")
	}

	return nil
}

func (c *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// 關鍵：這裡呼叫 UpdateIncident，但這個方法在 client.go 還沒寫！
	incident, err := client.UpdateIncident(spec.IncidentID, spec.Title, spec.Summary, spec.Severity, spec.Status)
	if err != nil {
		return fmt.Errorf("failed to update incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
		[]any{incident},
	)
}

func (c *UpdateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
