package servicenow

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
	Incident string `json:"incident" mapstructure:"incident"`
}

func (c *GetIncident) Name() string {
	return "servicenow.getIncident"
}

func (c *GetIncident) Label() string {
	return "Get Incident"
}

func (c *GetIncident) Description() string {
	return "Fetch a single ServiceNow incident by selecting it from the dropdown"
}

func (c *GetIncident) Documentation() string {
	return ""
}

func (c *GetIncident) Icon() string {
	return "servicenow"
}

func (c *GetIncident) Color() string {
	return "gray"
}

func (c *GetIncident) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incident",
			Label:       "Incident",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The incident to fetch",
			Placeholder: "Select an incident",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "incident",
				},
			},
		},
	}
}

func (c *GetIncident) Setup(ctx core.SetupContext) error {
	spec := GetIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Incident == "" {
		return errors.New("incident is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	incident, err := client.GetIncident(spec.Incident)
	if err != nil {
		return fmt.Errorf("error verifying incident: %w", err)
	}

	metadata := NodeMetadata{
		InstanceURL: client.InstanceURL,
		Incident:    &ResourceInfo{ID: incident.SysID, Name: incident.Number},
	}

	return ctx.Metadata.Set(metadata)
}

func (c *GetIncident) Execute(ctx core.ExecutionContext) error {
	spec := GetIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Incident == "" {
		return errors.New("incident is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	incident, err := client.GetIncident(spec.Incident)
	if err != nil {
		return fmt.Errorf("failed to get incident: %w", err)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeIncident, []any{incident})
}

func (c *GetIncident) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIncident) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIncident) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetIncident) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetIncident) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
