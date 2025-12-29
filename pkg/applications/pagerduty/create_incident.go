package pagerduty

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIncident struct{}

type CreateIncidentSpec struct {
	ServiceID   string           `json:"serviceId"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Urgency     string           `json:"urgency"`
	Assignments []AssignmentSpec `json:"assignments"`
}

type AssignmentSpec struct {
	UserID string `json:"userId"`
}

type CreateIncidentMetadata struct {
	IncidentID     string `json:"incidentId"`
	IncidentNumber int    `json:"incidentNumber"`
	Status         string `json:"status"`
	HTMLURL        string `json:"htmlUrl"`
	CreatedAt      string `json:"createdAt"`
}

func (c *CreateIncident) Name() string {
	return "pagerduty.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in PagerDuty"
}

func (c *CreateIncident) Icon() string {
	return "alert-triangle"
}

func (c *CreateIncident) Color() string {
	return "red"
}

func (c *CreateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "serviceId",
			Label:       "Service ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the PagerDuty service to create incident for",
			Placeholder: "e.g. PXXXXXX",
		},
		{
			Name:        "title",
			Label:       "Incident Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A succinct description of the incident",
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Additional details about the incident",
		},
		{
			Name:     "urgency",
			Label:    "Urgency",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "high",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "High", Value: "high"},
						{Label: "Low", Value: "low"},
					},
				},
			},
		},
		{
			Name:     "assignments",
			Label:    "Assignments",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assignment",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "userId",
								Label:       "User ID",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Description: "PagerDuty user ID to assign",
							},
						},
					},
				},
			},
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	// Synchronous components typically don't need setup
	return nil
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return ctx.ExecutionStateContext.Fail("configuration error", err.Error())
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return ctx.ExecutionStateContext.Fail("client error", err.Error())
	}

	// Build request
	request := &CreateIncidentRequest{
		Incident: IncidentPayload{
			Type:  "incident",
			Title: spec.Title,
			Service: ServiceReference{
				ID:   spec.ServiceID,
				Type: "service_reference",
			},
			Urgency: spec.Urgency,
		},
	}

	// Add description if provided
	if spec.Description != "" {
		request.Incident.Body = &IncidentBody{
			Type:    "incident_body",
			Details: spec.Description,
		}
	}

	// Add assignments if provided
	if len(spec.Assignments) > 0 {
		request.Incident.Assignments = make([]Assignment, len(spec.Assignments))
		for i, assignment := range spec.Assignments {
			request.Incident.Assignments[i] = Assignment{
				Assignee: Assignee{
					ID:   assignment.UserID,
					Type: "user_reference",
				},
			}
		}
	}

	// Create incident
	incident, err := client.CreateIncident(request)
	if err != nil {
		return ctx.ExecutionStateContext.Fail("failed to create incident", err.Error())
	}

	ctx.Logger.Infof("Created incident %s", incident.ID)

	// Store incident metadata
	metadata := CreateIncidentMetadata{
		IncidentID:     incident.ID,
		IncidentNumber: incident.IncidentNumber,
		Status:         incident.Status,
		HTMLURL:        incident.HTMLURL,
		CreatedAt:      incident.CreatedAt,
	}

	err = ctx.MetadataContext.Set(metadata)
	if err != nil {
		ctx.Logger.Warnf("Failed to store metadata: %v", err)
	}

	// Emit success on default channel
	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident.created",
		[]any{metadata},
	)
}

func (c *CreateIncident) Cancel(ctx core.ExecutionContext) error {
	// Synchronous component, nothing to cancel
	return nil
}

func (c *CreateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIncident) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (c *CreateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 404, fmt.Errorf("webhooks not supported for this component")
}
