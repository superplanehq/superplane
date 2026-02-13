package statuspage

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

type GetIncident struct{}

// GetIncidentSpec is the strongly typed configuration for the Get Incident component.
type GetIncidentSpec struct {
	Page     string `json:"page"`
	Incident string `json:"incident"`
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

- **Page** (required): The Statuspage containing the incident. Select from the dropdown, or switch to expression mode for workflow chaining (e.g. {{ $['Create Incident'].data.page_id }}).
- **Incident** (required): Incident ID to fetch. Supports expressions for workflow chaining (e.g. {{ $['Create Incident'].data.id }}).

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
		"id":         "p31zjtct2jer",
		"name":       "Database Connection Issues",
		"status":     "investigating",
		"impact":     "major",
		"shortlink":  "http://stspg.io/p31zjtct2jer",
		"created_at": "2026-02-12T10:30:00.000Z",
		"updated_at": "2026-02-12T10:30:00.000Z",
		"page_id":    "kctbh9vrtdwd",
		"component_ids": []string{"8kbf7d35c070"},
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
			Label:       "Page",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Statuspage containing the incident",
			Placeholder: "Select a page",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePage,
				},
			},
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

	// Resolve page name for metadata when Page is a static ID (no expression).
	// Skip API call if HTTP context is not available (e.g. in tests without HTTP mock).
	metadata := NodeMetadata{}
	if spec.Page != "" && !strings.Contains(spec.Page, "{{") && ctx.HTTP != nil {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err == nil {
			pages, err := client.ListPages()
			if err == nil {
				for _, p := range pages {
					if p.ID == spec.Page {
						metadata.PageName = p.Name
						break
					}
				}
			}
		}
	}
	return ctx.Metadata.Set(metadata)
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
