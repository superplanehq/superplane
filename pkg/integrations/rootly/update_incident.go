package rootly

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateIncident struct{}

type UpdateIncidentSpec struct {
	IncidentID string `mapstructure:"incident_id" json:"incident_id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Severity   string `json:"severity"`
	Status     string `json:"status"`
}

func (u *UpdateIncident) Name() string {
	return "rootly.updateIncident"
}

func (u *UpdateIncident) Label() string {
	return "Update Incident"
}

func (u *UpdateIncident) Description() string {
	return "Update an existing incident in Rootly"
}

func (u *UpdateIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incident_id",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to update.",
		},
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Description: "New title for the incident.",
		},
		{
			Name:        "summary",
			Label:       "Summary",
			Type:        configuration.FieldTypeText,
			Description: "New summary for the incident.",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Description: "New severity level.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "severity",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Description: "New status for the incident.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "status",
					UseNameAsValue: false,
				},
			},
		},
	}
}

func (u *UpdateIncident) Execute(ctx core.ExecutionContext) error {
	spec := UpdateIncidentSpec{}
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
		spec.Title,
		spec.Summary,
		spec.Severity,
		spec.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to update incident: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"rootly.incident",
		[]any{incident},
	)
}

func (u *UpdateIncident) Setup(ctx core.SetupContext) error { return nil }

func (u *UpdateIncident) Cleanup(ctx core.SetupContext) error { return nil }

func (u *UpdateIncident) Actions() []core.Action { return nil }

func (u *UpdateIncident) Cancel(ctx core.ExecutionContext) error { return nil }

func (u *UpdateIncident) Color() string {
	return "#1D2C3C"
}

func (u *UpdateIncident) Icon() string {
	return "rootly"
}
