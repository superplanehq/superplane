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

func (g *GetIncident) Name() string {
	return "rootly.getIncident"
}

func (g *GetIncident) Label() string {
	return "Get Incident"
}

func (g *GetIncident) Description() string {
	return "Retrieve a single incident from Rootly by ID"
}

func (g *GetIncident) Documentation() string {
	return `The Get Incident component retrieves a single incident from Rootly by ID so workflows can read details, branch on status, or update external systems.

## Use Cases

- **Status checks**: Fetch incident details to branch workflow based on status or severity
- **Slack notifications**: Get incident data to post detailed summaries to Slack
- **External sync**: Retrieve incident details to update external ticketing systems
- **Workflow enrichment**: Pull full incident data including events and action items

## Configuration

- **Incident ID**: The Rootly incident UUID to retrieve (required, supports expressions)

## Output

Returns the incident object with:
- **id**: Incident ID
- **title**: Incident title
- **summary**: Incident description
- **status**: Current incident status
- **severity**: Incident severity level
- **started_at**: When the incident started
- **resolved_at**: When resolved (if applicable)
- **mitigated_at**: When mitigated (if applicable)
- **url**: Direct link to the incident

## Examples

### Branch on severity
Use the severity field to determine workflow behavior:
` + "```yaml" + `
- getIncident:
    incidentId: "abc123-def456"
- if:
    condition: "{{ incident.severity == 'sev0' }}"
    then:
      - sendSlack:
          message: "SEV0 ALERT: {{ incident.title }}"
` + "```" + `

### Post summary to Slack
Get incident details for rich notifications:
` + "```yaml" + `
- getIncident:
    incidentId: "{{ trigger.incident_id }}"
- sendSlack:
    message: |
      **{{ incident.title }}**
      Status: {{ incident.status }}
      Severity: {{ incident.severity }}
      Started: {{ incident.started_at }}
      [View Incident]({{ incident.url }})
` + "```"
}

func (g *GetIncident) Icon() string {
	return "search"
}

func (g *GetIncident) Color() string {
	return "blue"
}

func (g *GetIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Rootly incident UUID to retrieve",
			Placeholder: "e.g., abc123-def456-789ghi",
		},
	}
}

func (g *GetIncident) Setup(ctx core.SetupContext) error {
	spec := GetIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	return nil
}

func (g *GetIncident) Execute(ctx core.ExecutionContext) error {
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

func (g *GetIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetIncident) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (g *GetIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}