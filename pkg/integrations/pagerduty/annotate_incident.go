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

type AnnotateIncident struct{}

type AnnotateIncidentSpec struct {
	IncidentID string `json:"incidentId"`
	Content    string `json:"content"`
	FromEmail  string `json:"fromEmail"`
}

func (c *AnnotateIncident) Name() string {
	return "pagerduty.annotateIncident"
}

func (c *AnnotateIncident) Label() string {
	return "Annotate Incident"
}

func (c *AnnotateIncident) Description() string {
	return "Add a note to an existing incident in PagerDuty"
}

func (c *AnnotateIncident) Documentation() string {
	return `The Annotate Incident component adds a note to an existing PagerDuty incident.

## Use Cases

- **Status updates**: Add progress updates to an incident
- **Investigation notes**: Document findings during incident investigation
- **Handoff information**: Leave notes for the next responder
- **Resolution details**: Document the root cause and resolution steps

## Configuration

- **Incident ID**: The ID of the incident to annotate (e.g., A12BC34567...)
- **Content**: The note content to add to the incident (supports expressions)
- **From Email**: Email address of a valid PagerDuty user (required for App OAuth, optional for API tokens)

## Output

Returns the incident object with all current information.`
}

func (c *AnnotateIncident) Icon() string {
	return "message-square"
}

func (c *AnnotateIncident) Color() string {
	return "gray"
}

func (c *AnnotateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *AnnotateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to annotate (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "content",
			Label:       "Note",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The note content to add to the incident",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email address of a valid PagerDuty user. Required for App OAuth and account-level API tokens, optional for user-level API tokens.",
			Placeholder: "user@example.com",
		},
	}
}

func (c *AnnotateIncident) Setup(ctx core.SetupContext) error {
	spec := AnnotateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	if spec.Content == "" {
		return errors.New("content is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *AnnotateIncident) Execute(ctx core.ExecutionContext) error {
	spec := AnnotateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	err = client.AddIncidentNote(spec.IncidentID, spec.FromEmail, spec.Content)
	if err != nil {
		return fmt.Errorf("failed to add note to incident: %v", err)
	}

	incident, err := client.GetIncident(spec.IncidentID)
	if err != nil {
		return fmt.Errorf("failed to fetch incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *AnnotateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *AnnotateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *AnnotateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *AnnotateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *AnnotateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
