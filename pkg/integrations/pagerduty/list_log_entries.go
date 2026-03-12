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

type ListLogEntries struct{}

type ListLogEntriesSpec struct {
	IncidentID string `json:"incidentId"`
	Limit      int    `json:"limit"`
}

const defaultLogEntriesLimit = 100

func (l *ListLogEntries) Name() string {
	return "pagerduty.listLogEntries"
}

func (l *ListLogEntries) Label() string {
	return "List Log Entries"
}

func (l *ListLogEntries) Description() string {
	return "List all log entries (audit trail) for a PagerDuty incident"
}

func (l *ListLogEntries) Documentation() string {
	return `The List Log Entries component retrieves all log entries (audit trail) for a PagerDuty incident.

## Use Cases

- **Audit trail**: Access complete incident history for compliance or review
- **Timeline reconstruction**: Build a detailed timeline of all incident activity
- **Incident analysis**: Analyze escalation patterns and response times
- **Forensics**: Review all actions taken during an incident

## Configuration

- **Incident ID**: The ID of the incident to list log entries for (e.g., A12BC34567...)
- **Limit**: Maximum number of log entries to return (default: 100)

## Output

Returns a list of log entries with:
- **id**: Log entry ID
- **type**: The type of log entry (e.g., trigger_log_entry, acknowledge_log_entry, annotate_log_entry)
- **summary**: A summary of what happened
- **created_at**: When the log entry was created
- **agent**: The agent (user or service) that caused the log entry
- **channel**: The channel through which the action was performed`
}

func (l *ListLogEntries) Icon() string {
	return "list"
}

func (l *ListLogEntries) Color() string {
	return "gray"
}

func (l *ListLogEntries) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *ListLogEntries) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to list log entries for (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Maximum number of log entries to return (default: 100)",
			Placeholder: "100",
		},
	}
}

func (l *ListLogEntries) Setup(ctx core.SetupContext) error {
	spec := ListLogEntriesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (l *ListLogEntries) Execute(ctx core.ExecutionContext) error {
	spec := ListLogEntriesSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	limit := spec.Limit
	if limit <= 0 {
		limit = defaultLogEntriesLimit
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	logEntries, err := client.ListIncidentLogEntries(spec.IncidentID, limit)
	if err != nil {
		return fmt.Errorf("failed to list log entries: %v", err)
	}

	responseData := map[string]any{
		"log_entries": logEntries,
		"total":       len(logEntries),
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.log_entries.list",
		[]any{responseData},
	)
}

func (l *ListLogEntries) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListLogEntries) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListLogEntries) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListLogEntries) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListLogEntries) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (l *ListLogEntries) Cleanup(ctx core.SetupContext) error {
	return nil
}
