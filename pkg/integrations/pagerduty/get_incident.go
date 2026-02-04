package pagerduty

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetIncident struct{}

type GetIncidentConfiguration struct {
	IncidentID string `json:"incidentId" mapstructure:"incidentId"`
}

func (c *GetIncident) Name() string {
	return "pagerduty.getIncident"
}

func (c *GetIncident) Label() string {
	return "Get Incident"
}

func (c *GetIncident) Description() string {
	return "Get a PagerDuty incident by ID"
}

func (c *GetIncident) Documentation() string {
	return `The Get Incident component retrieves a specific incident from PagerDuty by its ID, along with related alerts, notes, and log entries.

## Use Cases

- **Incident lookup**: Fetch incident details for processing or display
- **Workflow automation**: Get incident information to make decisions in workflows
- **Data enrichment**: Retrieve incident data to combine with other information
- **Status checking**: Check incident status before performing actions

## Configuration

- **Incident ID**: The ID of the incident to retrieve (e.g., PT4KHLK)

## Output

Returns the incident object along with related data:
- **incident**: Complete incident information including title, description, urgency, status, assignments, teams, priority, escalation policy
- **alerts**: Alerts that triggered or are associated with the incident
- **notes**: Annotations/notes added to the incident
- **log_entries**: Timeline of all changes and actions on the incident

Note: If fetching alerts, notes, or log entries fails, they will be omitted from the response but the incident data will still be returned.`
}

func (c *GetIncident) Icon() string {
	return "alert-circle"
}

func (c *GetIncident) Color() string {
	return "gray"
}

func (c *GetIncident) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetIncident) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "incidentId",
			Label:       "Incident ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The ID of the incident to retrieve (e.g., PT4KHLK)",
			Placeholder: "e.g., PT4KHLK",
		},
	}
}

func (c *GetIncident) Setup(ctx core.SetupContext) error {
	var config GetIncidentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if config.IncidentID == "" {
		return errors.New("incidentId is required")
	}

	return ctx.Metadata.Set(NodeMetadata{})
}

func (c *GetIncident) Execute(ctx core.ExecutionContext) error {
	var config GetIncidentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	var wg sync.WaitGroup
	var incident, alerts, notes, logEntries any
	var incidentErr error

	// Primary call - incident (required)
	wg.Add(1)
	go func() {
		defer wg.Done()
		incident, incidentErr = client.GetIncident(config.IncidentID)
	}()

	// Optional calls - ignore errors
	wg.Add(1)
	go func() {
		defer wg.Done()
		alerts, _ = client.GetIncidentAlerts(config.IncidentID)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		notes, _ = client.GetIncidentNotes(config.IncidentID)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logEntries, _ = client.GetIncidentLogEntries(config.IncidentID)
	}()

	wg.Wait()

	if incidentErr != nil {
		return fmt.Errorf("failed to get incident: %v", incidentErr)
	}

	// Build result map with incident data
	result := map[string]any{}

	// Extract the incident from the response wrapper
	if incidentMap, ok := incident.(map[string]any); ok {
		if inc, exists := incidentMap["incident"]; exists {
			result["incident"] = inc
		} else {
			result["incident"] = incident
		}
	} else {
		result["incident"] = incident
	}

	// Add optional data if available
	if alerts != nil {
		if alertsMap, ok := alerts.(map[string]any); ok {
			if alertsData, exists := alertsMap["alerts"]; exists {
				result["alerts"] = alertsData
			}
		}
	}

	if notes != nil {
		if notesMap, ok := notes.(map[string]any); ok {
			if notesData, exists := notesMap["notes"]; exists {
				result["notes"] = notesData
			}
		}
	}

	if logEntries != nil {
		if logEntriesMap, ok := logEntries.(map[string]any); ok {
			if logEntriesData, exists := logEntriesMap["log_entries"]; exists {
				result["log_entries"] = logEntriesData
			}
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"pagerduty.incident",
		[]any{result},
	)
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
