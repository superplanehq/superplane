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

type ListNotes struct{}

type ListNotesSpec struct {
	IncidentID string `json:"incidentId"`
}

func (l *ListNotes) Name() string {
	return "pagerduty.listNotes"
}

func (l *ListNotes) Label() string {
	return "List Notes"
}

func (l *ListNotes) Description() string {
	return "List all notes (timeline entries) for a PagerDuty incident"
}

func (l *ListNotes) Documentation() string {
	return `The List Notes component retrieves all notes (timeline entries) for a PagerDuty incident.

## Use Cases

- **Incident review**: Review all notes added to an incident
- **Timeline reconstruction**: Build a timeline of incident updates
- **Audit trail**: Access the history of notes for compliance or review
- **Note analysis**: Process or analyze notes for patterns or keywords

## Configuration

- **Incident ID**: The ID of the incident to list notes for (e.g., A12BC34567...)

## Output

Returns a list of notes with:
- **id**: Note ID
- **content**: The note content
- **created_at**: When the note was created
- **user**: The user who created the note
- **channel**: The channel through which the note was created`
}

func (l *ListNotes) Icon() string {
	return "message-square"
}

func (l *ListNotes) Color() string {
	return "gray"
}

func (l *ListNotes) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListNotes) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to list notes for (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
	}
}

func (l *ListNotes) Setup(ctx core.SetupContext) error {
	spec := ListNotesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (l *ListNotes) Execute(ctx core.ExecutionContext) error {
	spec := ListNotesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	notes, err := client.ListIncidentNotes(spec.IncidentID)
	if err != nil {
		return fmt.Errorf("failed to list notes: %v", err)
	}

	responseData := map[string]any{
		"notes": notes,
		"total": len(notes),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.notes.list",
		[]any{responseData},
	)
}

func (l *ListNotes) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListNotes) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListNotes) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListNotes) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListNotes) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (l *ListNotes) Cleanup(ctx core.SetupContext) error {
	return nil
}
