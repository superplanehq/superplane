package rootly

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIncident struct{}

type UpdateIncidentSpec struct {
	IncidentID string `mapstructure:"incidentId"`
	Title      string `mapstructure:"title"`
	Summary    string `mapstructure:"summary"`
	Status     string `mapstructure:"status"`
	Severity   string `mapstructure:"severity"`
}

func (c *UpdateIncident) Name() string {
	return "rootly.updateIncident"
}

func (c *UpdateIncident) Label() string {
	return "Update Incident"
}

func (c *UpdateIncident) Description() string {
	return "Update an existing Rootly incident"
}

func (c *UpdateIncident) Documentation() string {
	return `The Update Incident component modifies an existing incident in Rootly.

## Use Cases

- **Status updates**: Transition incidents through statuses (in_triage → started → mitigated → resolved → closed)
- **Severity changes**: Escalate or de-escalate incident severity
- **Enrichment**: Add or update title and summary as investigation progresses
- **External sync**: Update Rootly incidents based on signals from Sentry, PagerDuty, Jira, or ServiceNow

## Configuration

- **Incident ID**: The Rootly incident UUID to update (required). Accepts expressions from trigger events or previous component outputs.
- **Title**: New incident title (optional)
- **Summary**: New incident summary/description (optional)
- **Status**: New status — one of: in_triage, started, detected, acknowledged, mitigated, resolved, closed, cancelled (optional)
- **Severity**: Severity slug from Rootly (optional). Select from available severities.

## Output

Returns the updated incident object containing:
- id, sequential_id, title, slug
- status, summary, severity
- Timestamps (started_at, detected_at, acknowledged_at, mitigated_at, resolved_at, closed_at)
- URL to the incident in Rootly`
}

func (c *UpdateIncident) Icon() string {
	return "rootly"
}

func (c *UpdateIncident) Color() string {
	return "gray"
}

func (c *UpdateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., 2b4a5c6d-7e8f-9a0b-c1d2-e3f4a5b6c7d8",
			Description: "The Rootly incident UUID to update. Accepts expressions from trigger events or previous outputs.",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., Database outage in production",
			Description: "New title for the incident.",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "e.g., Investigating elevated error rates...",
			Description: "New summary or description for the incident.",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "New status for the incident.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "In Triage", Value: "in_triage"},
						{Label: "Started", Value: "started"},
						{Label: "Detected", Value: "detected"},
						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Mitigated", Value: "mitigated"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Closed", Value: "closed"},
						{Label: "Cancelled", Value: "cancelled"},
					},
				},
			},
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "New severity level for the incident.",
			Placeholder: "Select a severity",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
	}
}

func (c *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec := UpdateIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incident ID is required")
	}

	return nil
}

func (c *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	attrs := UpdateIncidentAttributes{}

	if spec.Title != "" {
		attrs.Title = spec.Title
	}

	if spec.Summary != "" {
		attrs.Summary = spec.Summary
	}

	if spec.Status != "" {
		attrs.Status = spec.Status
	}

	if spec.Severity != "" {
		severityID, err := resolveSeveritySlug(ctx.Integration, spec.Severity)
		if err != nil {
			return fmt.Errorf("failed to resolve severity: %w", err)
		}
		if severityID != "" {
			attrs.SeverityID = severityID
		}
	}

	incident, err := client.UpdateIncident(spec.IncidentID, attrs)
	if err != nil {
		return fmt.Errorf("failed to update incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
		[]any{incident},
	)
}

// resolveSeveritySlug resolves a severity name/slug to its ID using integration metadata.
func resolveSeveritySlug(integration core.IntegrationContext, severityName string) (string, error) {
	var metadata Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return "", fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	for _, sev := range metadata.Severities {
		if sev.Slug == severityName || sev.Name == severityName {
			return sev.ID, nil
		}
	}

	// If not found in metadata, use the value as-is (may be an ID already)
	return severityName, nil
}

func (c *UpdateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *UpdateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
