package pagerduty

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type ResolveIncident struct{}

type ResolveIncidentSpec struct {
	IncidentID string `json:"incidentId"`
	FromEmail  string `json:"fromEmail"`
	Resolution string `json:"resolution"`
}

const (
	ResolveChannelSuccess = "success"
	ResolveChannelFailed  = "failed"
)

func (c *ResolveIncident) Name() string {
	return "pagerduty.resolveIncident"
}

func (c *ResolveIncident) Label() string {
	return "Resolve Incident"
}

func (c *ResolveIncident) Description() string {
	return "Resolve an incident in PagerDuty"
}

func (c *ResolveIncident) Documentation() string {
	return `The Resolve Incident component resolves a PagerDuty incident so it is marked as fixed and no longer requires action.

## Use Cases

- **Auto-resolve incidents**: Resolve incidents when deployments fix the issue or when automation confirms recovery from SuperPlane
- **Ticket-driven resolution**: Resolve incidents when Jira tickets are closed or status pages are updated
- **Health-based resolution**: Close incidents when monitoring confirms the service is healthy

## Configuration

- **Incident ID**: PagerDuty incident ID (e.g., P1ABC23)
- **From Email**: Email address of a valid PagerDuty user resolving the incident
- **Resolution**: Optional resolution note or summary

## Output

- **Success**: Emits incident ID, status (resolved), and resolved timestamp
- **Failed**: Emits if the incident is not found or user is invalid`
}

func (c *ResolveIncident) Icon() string {
	return "check-circle"
}

func (c *ResolveIncident) Color() string {
	return "gray"
}

func (c *ResolveIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ResolveChannelSuccess, Label: "Success"},
		{Name: ResolveChannelFailed, Label: "Failed"},
	}
}

func (c *ResolveIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "PagerDuty incident ID (e.g., P1ABC23)",
			Placeholder: "e.g., P1ABC23",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Email address of a valid PagerDuty user resolving the incident",
			Placeholder: "user@example.com",
		},
		{
			Name:        "resolution",
			Label:       "Resolution",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Optional resolution note or summary",
		},
	}
}

func (c *ResolveIncident) Setup(ctx core.SetupContext) error {
	spec := ResolveIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	if spec.FromEmail == "" {
		return errors.New("fromEmail is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *ResolveIncident) Execute(ctx core.ExecutionContext) error {
	spec := ResolveIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.UpdateIncident(
		spec.IncidentID,
		spec.FromEmail,
		"resolved",
		"",
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		return ctx.ExecutionState.Emit(
			ResolveChannelFailed,
			"pagerduty.incident",
			[]any{map[string]any{
				"error":      err.Error(),
				"incidentId": spec.IncidentID,
			}},
		)
	}

	if spec.Resolution != "" {
		_ = client.AddIncidentNote(spec.IncidentID, spec.FromEmail, spec.Resolution)
	}

	return ctx.ExecutionState.Emit(
		ResolveChannelSuccess,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *ResolveIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ResolveIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *ResolveIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *ResolveIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *ResolveIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *ResolveIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
