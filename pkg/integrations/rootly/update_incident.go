package rootly

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// The identity of the component
type UpdateIncident struct{}

// The data structure for the component
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
			Type:        configuration.FieldTypeString,
			Description: "New summary for the incident.",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Resource:    "severity",
			Description: "New severity level.",
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Resource:    "status",
			Description: "New status for the incident.",
		},
	}
}

func (u *UpdateIncident) Execute(ctx core.ComponentContext) (any, error) {
	spec := UpdateIncidentSpec{}
	if err := ctx.DecodeSpec(&spec); err != nil {
		return nil, fmt.Errorf("failed to decode spec: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	incident, err := client.UpdateIncident(
		spec.IncidentID,
		spec.Title,
		spec.Summary,
		spec.Severity,
		spec.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	return incident, nil
}
