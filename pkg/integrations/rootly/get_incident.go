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

type GetIncident struct{}

type GetIncidentSpec struct {
	IncidentID string `json:"incidentId"`
}

func (c *GetIncident) Name() string {
	return "rootly.getIncident"
}

func (c *GetIncident) Label() string {
	return "Get Incident"
}

func (c *GetIncident) Description() string {
	return "Retrieve incident details from Rootly"
}

func (c *GetIncident) Documentation() string {
	return `The Get Incident component retrieves a single incident from Rootly by ID.

## Use Cases

- **Fetch incident details**: Post a summary to Slack or update a Jira ticket
- **Branch workflow on status**: Make decisions based on incident status or severity after a trigger fires
- **Enrich downstream steps**: Provide full incident data including events, action items, and services

## Configuration

- **Incident ID**: Rootly incident UUID (required, supports expressions). Can be obtained from a trigger payload or a previous step.

## Output

Returns the complete incident object including:
- **id**: Incident UUID
- **sequential_id**: Sequential incident number
- **title**: Incident title
- **slug**: URL-friendly incident identifier
- **status**: Current status (started, mitigated, resolved, cancelled)
- **summary**: Incident description/summary
- **severity**: Incident severity level
- **url**: Direct link to the incident in Rootly
- **started_at**: When the incident was created
- **mitigated_at**: When the incident was mitigated (if applicable)
- **resolved_at**: When the incident was resolved (if applicable)
- **user**: User who created the incident
- **started_by**: User who started the incident
- **services**: Affected services
- **groups**: Associated groups
- **events**: Incident timeline events
- **action_items**: Follow-up action items`
}

func (c *GetIncident) Icon() string {
	return "alert-triangle"
}

func (c *GetIncident) Color() string {
	return "gray"
}

func (c *GetIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., abc123-def456 or {{$.data.incident.id}}",
			Description: "Rootly incident UUID (e.g. from trigger payload or previous step)",
		},
	}
}

func (c *GetIncident) Setup(ctx core.SetupContext) error {
	spec := GetIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incident ID is required")
	}

	return nil
}

func (c *GetIncident) Execute(ctx core.ExecutionContext) error {
	spec := GetIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incident ID is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.GetIncidentWithDetails(spec.IncidentID)
	if err != nil {
		return fmt.Errorf("failed to get incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
		[]any{incident},
	)
}

func (c *GetIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
