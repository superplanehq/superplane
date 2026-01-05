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

type CreateIncident struct{}

type CreateIncidentSpec struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Urgency     string `json:"urgency"`
	Service     string `json:"service"`
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
	return "gray"
}

func (c *CreateIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
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
			Type:        configuration.FieldTypeString,
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
			Name:        "service",
			Label:       "Service",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the PagerDuty service to create incident for",
			Placeholder: "e.g. PXXXXXX",
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Title == "" {
		return errors.New("title is required")
	}

	if spec.Urgency == "" {
		return errors.New("urgency is required")
	}

	if spec.Service == "" {
		return errors.New("service is required")
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	service, err := client.GetService(spec.Service)
	if err != nil {
		return fmt.Errorf("error finding service: %v", err)
	}

	return ctx.MetadataContext.Set(NodeMetadata{Service: service})
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.AppInstallationContext)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	incident, err := client.CreateIncident(spec.Title, spec.Service, spec.Urgency, spec.Description)
	if err != nil {
		return fmt.Errorf("failed to create incident: %v", err)
	}

	return ctx.ExecutionStateContext.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{incident},
	)
}

func (c *CreateIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *CreateIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *CreateIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
