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

- **Auto-resolve incidents**: Resolve incidents when deployments fix the issue or automation confirms recovery
- **Cross-system resolution**: Resolve incidents when Jira tickets are closed or status pages are updated
- **Health-based resolution**: Close incidents when monitoring confirms the service is healthy

## Configuration

- **Incident ID**: The ID of the incident to resolve (e.g., P1ABC23)
- **From Email**: Email address of a valid PagerDuty user resolving the incident (required for App OAuth, optional for API tokens)
- **Resolution**: Optional resolution note or summary

## Output

Returns the resolved incident object with all current information including the resolved status and timestamp.`
}

func (c *ResolveIncident) Icon() string {
	return "check-circle"
}

func (c *ResolveIncident) Color() string {
	return "green"
}

func (c *ResolveIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ResolveIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to resolve (e.g., P1ABC23)",
			Placeholder: "e.g., P1ABC23",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email address of a valid PagerDuty user resolving the incident. Required for App OAuth and account-level API tokens, optional for user-level API tokens.",
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

	incident, err := client.ResolveIncident(spec.IncidentID, spec.FromEmail, spec.Resolution)
	if err != nil {
		return fmt.Errorf("failed to resolve incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
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
