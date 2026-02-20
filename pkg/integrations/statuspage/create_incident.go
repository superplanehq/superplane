package statuspage

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateIncident struct{}

// CreateIncidentSpec is the strongly typed configuration for the Create Incident component.
// Conditionally visible fields use pointers so they can be nil when not shown.
type CreateIncidentSpec struct {
	Page            string  `json:"page"`
	IncidentType    string  `json:"incidentType"`
	Name            string  `json:"name"`
	Body            string  `json:"body"`
	StatusRealtime  *string `json:"statusRealtime,omitempty"`
	StatusScheduled *string `json:"statusScheduled,omitempty"`
	ImpactOverride  *string `json:"impactOverride,omitempty"`
	Components      []struct {
		ComponentID string `json:"componentId"`
		Status      string `json:"status"`
	} `json:"components"`
	ScheduledFor            *string `json:"scheduledFor,omitempty"`
	ScheduledUntil          *string `json:"scheduledUntil,omitempty"`
	ScheduledTimezone       *string `json:"scheduledTimezone,omitempty"`
	ScheduledRemindPrior    *bool   `json:"scheduledRemindPrior,omitempty"`
	ScheduledAutoInProgress *bool   `json:"scheduledAutoInProgress,omitempty"`
	ScheduledAutoCompleted  *bool   `json:"scheduledAutoCompleted,omitempty"`
	DeliverNotifications    *bool   `json:"deliverNotifications,omitempty"`
}

// derefStr safely dereferences a *string, returning "" if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefBool safely dereferences a *bool, returning false if nil.
func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func (c *CreateIncident) Name() string {
	return "statuspage.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident or scheduled maintenance on your Statuspage."
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new realtime or scheduled incident on your Atlassian Statuspage.

## Use Cases

- **Realtime incidents**: Create and notify subscribers when an unexpected outage occurs
- **Scheduled maintenance**: Schedule maintenance windows with optional reminders and auto-transitions
- **Integration workflows**: Create incidents from monitoring alerts or other workflow events

## Configuration

- **Page** (required): The Statuspage to create the incident on. Supports expressions for workflow chaining (e.g. {{ $['Create Incident'].data.page_id }}).
- **Incident type**: Realtime (active incident) or Scheduled (planned maintenance)
- **Name** (required): Short title for the incident
- **Body** (optional): Initial message shown as the first incident update
- **Status** (realtime): investigating, identified, monitoring, or resolved
- **Impact override** (realtime): none, minor, major, or critical
- **Components** (optional): List of components and their status. Each item has Component ID (supports expressions) and Status (operational, degraded_performance, partial_outage, major_outage, under_maintenance)
- **Scheduled For / Until** (scheduled): Start and end time for scheduled maintenance (ISO 8601, e.g. 2026-02-15T02:00)
- **Scheduled timezone** (scheduled): Timezone for the scheduled times (default UTC). Output is converted to UTC for the API.
- **Scheduled options** (scheduled): Remind prior, auto in-progress, auto completed
- **Deliver notifications** (optional): Whether to send notifications for the initial update (default: true)

## Output

Returns the full Statuspage Incident object from the API. The payload has structure { type, timestamp, data } where data is the incident. Common expression paths (use $['Node Name'].data. as prefix):
- data.id, data.name, data.status, data.impact
- data.shortlink — link to the incident
- data.created_at, data.updated_at
- data.components — array of affected components
- data.incident_updates — array of update messages`
}

func (c *CreateIncident) Icon() string {
	return "activity"
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
			Name:        "page",
			Label:       "Page",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Statuspage to create the incident on",
			Placeholder: "Select a page",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePage,
				},
			},
		},
		{
			Name:     "incidentType",
			Label:    "Incident type",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "realtime",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Realtime", Value: "realtime"},
						{Label: "Scheduled", Value: "scheduled"},
					},
				},
			},
		},
		{
			Name:        "name",
			Label:       "Incident name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Short title for the incident",
		},
		{
			Name:        "body",
			Label:       "Initial message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "First incident update message (optional)",
		},
		{
			Name:        "statusRealtime",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     "investigating",
			Description: "Incident status (supports expressions)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIncidentStatusRealtime,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"realtime"}},
			},
		},
		{
			Name:        "statusScheduled",
			Label:       "Status",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     "scheduled",
			Description: "Incident status (supports expressions)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIncidentStatusScheduled,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "impactOverride",
			Label:       "Impact override",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Override displayed severity (supports expressions)",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeImpact,
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"realtime"}},
			},
		},
		{
			Name:        "components",
			Label:       "Components",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Components to associate with this incident and their status. Component ID supports expressions.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Component",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "componentId",
								Label:       "Component ID",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "e.g. 8kbf7d35c070 or {{ $['X'].data.id }}",
							},
							{
								Name:     "status",
								Label:    "Status",
								Type:     configuration.FieldTypeSelect,
								Required: false,
								Default:  "degraded_performance",
								TypeOptions: &configuration.TypeOptions{
									Select: &configuration.SelectTypeOptions{
										Options: []configuration.FieldOption{
											{Label: "Operational", Value: "operational"},
											{Label: "Degraded performance", Value: "degraded_performance"},
											{Label: "Partial outage", Value: "partial_outage"},
											{Label: "Major outage", Value: "major_outage"},
											{Label: "Under maintenance", Value: "under_maintenance"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name:        "scheduledFor",
			Label:       "Scheduled for",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "When the scheduled maintenance starts (ISO 8601)",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "scheduledUntil",
			Label:       "Scheduled until",
			Type:        configuration.FieldTypeDateTime,
			Required:    false,
			Description: "When the scheduled maintenance ends (ISO 8601)",
			TypeOptions: &configuration.TypeOptions{
				DateTime: &configuration.DateTimeTypeOptions{
					Format: "2006-01-02T15:04",
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "scheduledTimezone",
			Label:       "Timezone",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     "UTC",
			Description: "Timezone for scheduled times. Values are converted to UTC for the API.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "UTC", Value: "UTC"},
						{Label: "America/New_York", Value: "America/New_York"},
						{Label: "America/Los_Angeles", Value: "America/Los_Angeles"},
						{Label: "America/Chicago", Value: "America/Chicago"},
						{Label: "Europe/London", Value: "Europe/London"},
						{Label: "Europe/Paris", Value: "Europe/Paris"},
						{Label: "Asia/Tokyo", Value: "Asia/Tokyo"},
						{Label: "Asia/Singapore", Value: "Asia/Singapore"},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:     "scheduledRemindPrior",
			Label:    "Remind subscribers 60 minutes before",
			Type:     configuration.FieldTypeBool,
			Required: false,
			Default:  false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:     "scheduledAutoInProgress",
			Label:    "Auto transition to In Progress at start",
			Type:     configuration.FieldTypeBool,
			Required: false,
			Default:  false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:     "scheduledAutoCompleted",
			Label:    "Auto transition to Completed at end",
			Type:     configuration.FieldTypeBool,
			Required: false,
			Default:  false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "incidentType", Values: []string{"scheduled"}},
			},
		},
		{
			Name:        "deliverNotifications",
			Label:       "Deliver notifications",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Send notifications for the initial incident update (default: true)",
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.Page == "" {
		return errors.New("page is required")
	}

	if spec.IncidentType == "" {
		return errors.New("incidentType is required")
	}

	if spec.IncidentType != "realtime" && spec.IncidentType != "scheduled" {
		return fmt.Errorf("incidentType must be realtime or scheduled, got %q", spec.IncidentType)
	}

	if spec.Name == "" {
		return errors.New("name is required")
	}

	if spec.IncidentType == "scheduled" {
		if spec.ScheduledFor == nil || *spec.ScheduledFor == "" {
			return errors.New("scheduledFor is required for scheduled incidents")
		}
		if spec.ScheduledUntil == nil || *spec.ScheduledUntil == "" {
			return errors.New("scheduledUntil is required for scheduled incidents")
		}

		// Validate scheduledFor and scheduledUntil when static (no expressions).
		// Invalid datetime/timezone values must fail setup, not Execute.
		forStr := derefStr(spec.ScheduledFor)
		untilStr := derefStr(spec.ScheduledUntil)
		tz := "UTC"
		if spec.ScheduledTimezone != nil && *spec.ScheduledTimezone != "" {
			tz = *spec.ScheduledTimezone
		}
		canValidate := !strings.Contains(forStr, "{{") && !strings.Contains(untilStr, "{{") && !strings.Contains(tz, "{{")
		if canValidate {
			parsedFor, errFor := toUTCISO8601(forStr, tz)
			if errFor != nil {
				return fmt.Errorf("invalid scheduledFor: %w", errFor)
			}
			parsedUntil, errUntil := toUTCISO8601(untilStr, tz)
			if errUntil != nil {
				return fmt.Errorf("invalid scheduledUntil: %w", errUntil)
			}
			if parsedFor >= parsedUntil {
				return errors.New("scheduledFor must be before scheduledUntil")
			}
		}
	}

	// Resolve page name and component names for metadata when IDs are static (no expressions).
	componentIDs := componentIDsForMetadataSetup(ctx.Configuration, func() []string {
		var ids []string
		for _, item := range spec.Components {
			if item.ComponentID != "" {
				ids = append(ids, item.ComponentID)
			}
		}
		return ids
	})
	metadata, err := resolveMetadataSetup(ctx, spec.Page, componentIDs)
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(metadata)
}

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	nameOrIDToStatus := make(map[string]string)
	rawIDs := make([]string, 0, len(spec.Components))
	for _, item := range spec.Components {
		if item.ComponentID == "" {
			continue
		}
		status := item.Status
		if status == "" {
			status = "degraded_performance"
		}
		nameOrIDToStatus[item.ComponentID] = status
		rawIDs = append(rawIDs, item.ComponentID)
	}

	var componentIDs []string
	var components map[string]string
	if len(nameOrIDToStatus) > 0 && !containsExpression(rawIDs) {
		var errResolve error
		componentIDs, components, errResolve = resolveComponentNameOrIDs(client, spec.Page, nameOrIDToStatus)
		if errResolve != nil {
			return fmt.Errorf("failed to resolve component IDs: %w", errResolve)
		}
	} else {
		components = nameOrIDToStatus
		componentIDs = make([]string, 0, len(components))
		for id := range components {
			componentIDs = append(componentIDs, id)
		}
	}

	deliverNotifications := spec.DeliverNotifications
	if deliverNotifications == nil {
		t := true
		deliverNotifications = &t
	}

	req := CreateIncidentRequest{
		Name:                 spec.Name,
		Body:                 spec.Body,
		ComponentIDs:         componentIDs,
		Components:           components,
		Realtime:             spec.IncidentType == "realtime",
		DeliverNotifications: deliverNotifications,
	}

	var status string
	if spec.IncidentType == "scheduled" {
		status = derefStr(spec.StatusScheduled)
	} else {
		status = derefStr(spec.StatusRealtime)
	}
	if status == "" && spec.IncidentType == "realtime" {
		status = "investigating"
	}
	if status == "" && spec.IncidentType == "scheduled" {
		status = "scheduled"
	}
	req.Status = status

	if spec.IncidentType == "realtime" {
		req.ImpactOverride = derefStr(spec.ImpactOverride)
	} else {
		tz := derefStr(spec.ScheduledTimezone)
		if tz == "" {
			tz = "UTC"
		}
		scheduledFor, err := toUTCISO8601(derefStr(spec.ScheduledFor), tz)
		if err != nil {
			return fmt.Errorf("invalid scheduledFor: %w", err)
		}
		scheduledUntil, err := toUTCISO8601(derefStr(spec.ScheduledUntil), tz)
		if err != nil {
			return fmt.Errorf("invalid scheduledUntil: %w", err)
		}
		req.ScheduledFor = scheduledFor
		req.ScheduledUntil = scheduledUntil
		req.ScheduledRemindPrior = derefBool(spec.ScheduledRemindPrior)
		req.ScheduledAutoInProgress = derefBool(spec.ScheduledAutoInProgress)
		req.ScheduledAutoCompleted = derefBool(spec.ScheduledAutoCompleted)
	}

	incident, err := client.CreateIncident(spec.Page, req)
	if err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"statuspage.incident",
		[]any{incident},
	)
}

// toUTCISO8601 parses a datetime string and returns it as ISO 8601 UTC (e.g. 2026-02-15T02:00:00Z).
// Input format: "2006-01-02T15:04" or "2006-01-02T15:04:05" from the datetime picker, or "2006-01-02T15:04:05Z" for UTC.
// If the input ends with "Z", it is already UTC and is parsed as such. Otherwise, the datetime is interpreted
// in the given timezone (e.g. "America/New_York") and converted to UTC.
func toUTCISO8601(dt, timezone string) (string, error) {
	if dt == "" {
		return "", nil
	}
	// "Z" suffix means UTC per ISO 8601 — parse directly, do not re-interpret in user timezone
	if strings.HasSuffix(dt, "Z") {
		formats := []string{"2006-01-02T15:04:05Z", "2006-01-02T15:04Z"}
		for _, f := range formats {
			if parsed, err := time.Parse(f, dt); err == nil {
				return parsed.UTC().Format("2006-01-02T15:04:05") + "Z", nil
			}
		}
		return "", fmt.Errorf("could not parse UTC datetime %q", dt)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}
	formats := []string{"2006-01-02T15:04:05", "2006-01-02T15:04"}
	var t time.Time
	for _, f := range formats {
		if parsed, err := time.ParseInLocation(f, dt, loc); err == nil {
			t = parsed
			break
		}
	}
	if t.IsZero() {
		return "", fmt.Errorf("could not parse datetime %q", dt)
	}
	return t.UTC().Format("2006-01-02T15:04:05") + "Z", nil
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

func (c *CreateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
