package statuspage

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

// GetIncidentSpec is the strongly typed configuration for the Get Incident component.
type GetIncidentSpec struct {
	Page     string `mapstructure:"page"`
	Incident string `mapstructure:"incident"`
}

func (c *GetIncident) Name() string {
	return "statuspage.getIncident"
}

func (c *GetIncident) Label() string {
	return "Get Incident"
}

func (c *GetIncident) Description() string {
	return "Get the full details of an incident including its timeline and status on your Statuspage."
}

func (c *GetIncident) Documentation() string {
	return `The Get Incident component fetches the full details of an existing incident on your Atlassian Statuspage.

## Use Cases

- **Incident lookup**: Fetch incident details for processing or display
- **Workflow automation**: Get incident information to make decisions in workflows
- **Timeline enrichment**: Retrieve the incident timeline (incident_updates) for reporting or notifications
- **Status checking**: Check incident status before performing actions

## Configuration

- **Page** (required): Page ID containing the incident. Supports expressions for dynamic values from upstream nodes (e.g. Create Incident).
- **Incident** (required): Incident ID to fetch. Supports expressions for dynamic values from upstream nodes (e.g. Create Incident).

## Output

Returns the full Statuspage Incident object from the API. The payload has structure { type, timestamp, data } where data is the incident. Common expression paths (use $['Node Name'].data. as prefix):
- data.id, data.name, data.status, data.impact
- data.shortlink — link to the incident
- data.created_at, data.updated_at, data.resolved_at
- data.components — array of affected components
- data.incident_updates — array of update messages (timeline), in API order`
}

func (c *GetIncident) Icon() string {
	return "activity"
}

func (c *GetIncident) Color() string {
	return "gray"
}

func (c *GetIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIncident) ExampleOutput() map[string]any {
	return map[string]any{
		"id":                  "p31zjtct2jer",
		"name":                "Database Connection Issues",
		"status":              "investigating",
		"impact":              "major",
		"shortlink":           "http://stspg.io/p31zjtct2jer",
		"created_at":          "2026-02-12T10:30:00.000Z",
		"updated_at":          "2026-02-12T10:30:00.000Z",
		"page_id":             "kctbh9vrtdwd",
		"affected_components": []string{"API"},
		"component_count":     1,
		"incident_updates": []map[string]any{
			{
				"id":         "upd1",
				"status":     "investigating",
				"body":       "We are investigating reports of slow database queries.",
				"created_at": "2026-02-12T10:30:00.000Z",
			},
		},
	}
}

func (c *GetIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "page",
			Label:       "Page ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Page ID containing the incident (supports expressions)",
			Placeholder: "e.g., kctbh9vrtdwd or {{ $['Create Incident'].data.page_id }}",
		},
		{
			Name:        "incident",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Incident ID to fetch (supports expressions)",
			Placeholder: "e.g., p31zjtct2jer or {{ $['Create Incident'].data.id }}",
		},
	}
}

func (c *GetIncident) Setup(ctx core.SetupContext) error {
	spec := GetIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Page == "" {
		return errors.New("page is required")
	}

	if spec.Incident == "" {
		return errors.New("incident is required")
	}

	return nil
}

func (c *GetIncident) Execute(ctx core.ExecutionContext) error {
	spec := GetIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	incident, err := client.GetIncident(spec.Page, spec.Incident)
	if err != nil {
		return fmt.Errorf("failed to get incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"statuspage.incident",
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
