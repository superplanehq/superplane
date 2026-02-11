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
	return "Retrieve a single incident from Rootly by ID"
}

func (c *GetIncident) Documentation() string {
	return `The Get Incident component retrieves a single incident from Rootly by its ID.

## Use Cases

- **Fetch incident details**: Read incident details to post a summary to Slack or update a Jira ticket
- **Branch on status**: Branch workflow on incident status or severity after a trigger fires
- **Enrich downstream steps**: Enrich a downstream step with full incident data

## Configuration

- **Incident ID**: Rootly incident UUID (required, supports expressions). Can come from trigger payload or a previous step.

## Output

Returns the incident object including:
- **id**: Incident ID
- **title**: Incident title
- **summary**: Incident summary
- **status**: Current incident status
- **severity**: Incident severity
- **started_at**: Incident creation timestamp
- **mitigated_at**: Incident mitigation timestamp
- **resolved_at**: Incident resolution timestamp
- **url**: Link to the incident in Rootly`
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
			Description: "The Rootly incident ID to retrieve",
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.GetIncident(spec.IncidentID)
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
