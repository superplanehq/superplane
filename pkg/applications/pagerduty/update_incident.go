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

type UpdateIncident struct{}

type UpdateIncidentSpec struct {
	IncidentID       string   `json:"incidentId"`
	FromEmail        string   `json:"fromEmail"`
	Status           string   `json:"status"`
	Priority         string   `json:"priority"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	EscalationPolicy string   `json:"escalationPolicy"`
	Assignees        []string `json:"assignees"`
	Note             string   `json:"note"`
}

func (c *UpdateIncident) Name() string {
	return "pagerduty.updateIncident"
}

func (c *UpdateIncident) Label() string {
	return "Update Incident"
}

func (c *UpdateIncident) Description() string {
	return "Update an existing incident in PagerDuty"
}

func (c *UpdateIncident) Icon() string {
	return "edit"
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
			Description: "The ID of the incident to update (e.g., A12BC34567...)",
			Placeholder: "e.g., A12BC34567...",
		},
		{
			Name:        "fromEmail",
			Label:       "From Email",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Email address of a valid PagerDuty user. Required for App OAuth and account-level API tokens, optional for user-level API tokens.",
			Placeholder: "user@example.com",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Update the incident status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Acknowledged", Value: "acknowledged"},
						{Label: "Resolved", Value: "resolved"},
						{Label: "Triggered", Value: "triggered"},
					},
				},
			},
		},
		{
			Name:        "priority",
			Label:       "Priority",
			Type:        configuration.FieldTypeAppInstallationResource,
			Required:    false,
			Description: "Update the incident priority",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "priority",
				},
			},
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Update the incident title",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Update the incident description (body)",
		},
		{
			Name:        "escalationPolicy",
			Label:       "Escalation Policy",
			Type:        configuration.FieldTypeAppInstallationResource,
			Required:    false,
			Description: "Update the escalation policy",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "escalation_policy",
				},
			},
		},
		{
			Name:        "assignees",
			Label:       "Assignees",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Update incident assignees (user IDs)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "User ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "note",
			Label:       "Note",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Add a note/comment to the incident",
		},
	}
}

func (c *UpdateIncident) Setup(ctx core.SetupContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	// Validate that at least one field to update is provided
	hasUpdate := spec.Status != "" || spec.Priority != "" ||
		spec.Title != "" || spec.Description != "" ||
		spec.EscalationPolicy != "" || len(spec.Assignees) > 0

	if !hasUpdate && spec.Note == "" {
		return errors.New("at least one field to update or a note must be provided")
	}

	// Store minimal metadata (no external API call needed for setup)
	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.AppInstallation)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Update incident fields if any are provided
	hasUpdate := spec.Status != "" || spec.Priority != "" ||
		spec.Title != "" || spec.Description != "" ||
		spec.EscalationPolicy != "" || len(spec.Assignees) > 0

	var incident any
	if hasUpdate {
		incident, err = client.UpdateIncident(
			spec.IncidentID,
			spec.FromEmail,
			spec.Status,
			spec.Priority,
			spec.Title,
			spec.Description,
			spec.EscalationPolicy,
			spec.Assignees,
		)
		if err != nil {
			return fmt.Errorf("failed to update incident: %v", err)
		}
	}

	// Add note if provided
	if spec.Note != "" {
		err = client.AddIncidentNote(spec.IncidentID, spec.FromEmail, spec.Note)
		if err != nil {
			return fmt.Errorf("failed to add note to incident: %v", err)
		}
	}

	// If only note was added, fetch the incident to return
	if incident == nil {
		incident, err = client.GetIncident(spec.IncidentID)
		if err != nil {
			return fmt.Errorf("failed to fetch incident: %v", err)
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *UpdateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
