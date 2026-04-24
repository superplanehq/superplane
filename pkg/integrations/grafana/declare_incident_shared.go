package grafana

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	legacyDefaultRoomPrefix = "incident"
)

var grafanaIncidentStatusOptions = []configuration.FieldOption{
	{Label: "Active", Value: incidentStatusActive},
	{Label: "Resolved", Value: incidentStatusResolved},
}

var grafanaIncidentDebriefStatusOptions = []configuration.FieldOption{
	{Label: "Report completed", Value: "completed"},
	{Label: "Report in progress", Value: "in_progress"},
	{Label: "Report not needed", Value: "not_needed"},
	{Label: "Report not started", Value: "not_started"},
}

type declareIncidentSpec struct {
	Title         string   `json:"title" mapstructure:"title"`
	Severity      string   `json:"severity" mapstructure:"severity"`
	Description   string   `json:"description" mapstructure:"description"`
	Labels        []string `json:"labels" mapstructure:"labels"`
	Status        string   `json:"status" mapstructure:"status"`
	DebriefStatus string   `json:"debriefStatus" mapstructure:"debriefStatus"`
	StartTime     string   `json:"startTime" mapstructure:"startTime"`
	IsDrill       *bool    `json:"isDrill,omitempty" mapstructure:"isDrill"`
}

func declareIncidentConfiguration(includeLegacyToggle bool) []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Incident title",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Grafana IRM severity",
			Placeholder: "minor",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: resourceTypeIncidentSeverity,
				},
			},
		},
		{
			Name:        "description",
			Label:       "Initial Status Update",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Initial status update added to the incident",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Labels to attach to the incident",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "status",
			Label:       "Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     incidentStatusActive,
			Description: "Initial incident status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: grafanaIncidentStatusOptions,
				},
			},
		},
		{
			Name:        "debriefStatus",
			Label:       "Debrief Status",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional debrief status to record to the Grafana IRM incident custom field with slug debrief_status",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: grafanaIncidentDebriefStatusOptions,
				},
			},
		},
		{
			Name:        "startTime",
			Label:       "Start Time",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "When the incident began",
		},
	}

	if includeLegacyToggle {
		fields = append(fields, configuration.Field{
			Name:        "isDrill",
			Label:       "Is Drill",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Description: "Create the incident as a drill",
		})
	}

	return fields
}

func validateDeclareIncidentSpec(spec declareIncidentSpec) error {
	if strings.TrimSpace(spec.Title) == "" {
		return errors.New("title is required")
	}
	if err := validateIncidentSeverity(spec.Severity, true); err != nil {
		return err
	}
	if err := validateIncidentStatus(spec.Status); err != nil {
		return err
	}
	if err := validateIncidentDebriefStatus(spec.DebriefStatus); err != nil {
		return err
	}
	if _, err := parseIncidentStartTime(spec.StartTime); err != nil {
		return err
	}

	return nil
}

func validateIncidentStatus(status string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return nil
	}

	switch status {
	case incidentStatusActive, incidentStatusResolved:
		return nil
	default:
		return fmt.Errorf("status must be one of: %s, %s", incidentStatusActive, incidentStatusResolved)
	}
}

func validateIncidentDebriefStatus(status string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		return nil
	}

	for _, option := range grafanaIncidentDebriefStatusOptions {
		if status == option.Value {
			return nil
		}
	}

	return fmt.Errorf("debriefStatus must be one of: completed, in_progress, not_needed, not_started")
}

func parseIncidentStartTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	timezoneFormats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05Z07:00",
	}
	for _, format := range timezoneFormats {
		if parsed, err := time.Parse(format, value); err == nil {
			return &parsed, nil
		}
	}

	localFormats := []string{
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	}
	for _, format := range localFormats {
		if parsed, err := time.ParseInLocation(format, value, time.Local); err == nil {
			return &parsed, nil
		}
	}

	return nil, fmt.Errorf("could not parse startTime %q", value)
}

func executeDeclareIncident(ctx core.ExecutionContext, spec declareIncidentSpec, isDrill bool) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration, true)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	startTime, err := parseIncidentStartTime(spec.StartTime)
	if err != nil {
		return err
	}

	incident, err := client.DeclareIncident(DeclareIncidentInput{
		Title:               spec.Title,
		Severity:            spec.Severity,
		Labels:              spec.Labels,
		RoomPrefix:          legacyDefaultRoomPrefix,
		IsDrill:             isDrill,
		Status:              spec.Status,
		DebriefStatus:       spec.DebriefStatus,
		InitialStatusUpdate: spec.Description,
		StartTime:           startTime,
	})
	if err != nil {
		return fmt.Errorf("error declaring incident: %w", err)
	}

	if incident != nil && strings.TrimSpace(incident.IncidentURL) == "" {
		incident.IncidentURL, _ = buildIncidentWebURL(ctx.Integration, incident.IncidentID)
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "grafana.incident.declared", []any{incident})
}
