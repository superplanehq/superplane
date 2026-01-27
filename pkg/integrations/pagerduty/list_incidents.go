package pagerduty

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// Output channel names for ListIncidents component
const (
	ChannelNameClear = "clear"
	ChannelNameLow   = "low"
	ChannelNameHigh  = "high"
)

type ListIncidents struct{}

type ListIncidentsSpec struct {
	Services []string `json:"services,omitempty"`
}

type ListIncidentsNodeMetadata struct {
	Services []Service `json:"services" mapstructure:"services"`
}

func (l *ListIncidents) Name() string {
	return "pagerduty.listIncidents"
}

func (l *ListIncidents) Label() string {
	return "List Incidents"
}

func (l *ListIncidents) Description() string {
	return "Query PagerDuty to get a list of all open incidents (triggered and acknowledged)"
}

func (l *ListIncidents) Documentation() string {
	return `The List Incidents component queries PagerDuty for open incidents and routes execution based on urgency levels.

## Use Cases

- **Health checks**: Check for active incidents and route based on severity
- **Incident monitoring**: Monitor incident status across services
- **Automated response**: Trigger workflows based on incident presence
- **Reporting**: Collect incident data for reporting or analysis

## Configuration

- **Services**: Optional list of services to filter incidents (leave empty to get incidents from all services)

## Output Channels

- **Clear**: No open incidents found
- **Low**: Only low urgency incidents found
- **High**: One or more high urgency incidents found

## Output

Returns a list of open incidents with:
- **id**: Incident ID
- **incident_number**: Human-readable incident number
- **status**: Incident status (triggered, acknowledged)
- **urgency**: Incident urgency (low, high)
- **title**: Incident title
- **service**: Service information
- **assignments**: Current assignments`
}

func (l *ListIncidents) Icon() string {
	return "alert-triangle"
}

func (l *ListIncidents) Color() string {
	return "gray"
}

func (l *ListIncidents) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameClear, Label: "Clear", Description: "No open incidents found"},
		{Name: ChannelNameLow, Label: "Low", Description: "Only low urgency incidents found"},
		{Name: ChannelNameHigh, Label: "High", Description: "One or more high urgency incidents found"},
	}
}

func (l *ListIncidents) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:      "services",
			Label:     "Services",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:  "service",
					Multi: true,
				},
			},
			Description: "Filter incidents by specific services. If not specified, all services are included.",
		},
	}
}

func (l *ListIncidents) Setup(ctx core.SetupContext) error {
	var nodeMetadata ListIncidentsNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	// If services are already set, skip setup
	if len(nodeMetadata.Services) > 0 {
		return nil
	}

	spec := ListIncidentsSpec{}
	err = mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	// If no services are specified in configuration, we don't need to fetch metadata
	if len(spec.Services) == 0 {
		return ctx.Metadata.Set(ListIncidentsNodeMetadata{})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client during setup: %w", err)
	}

	// Fetch service details for the configured services
	services := make([]Service, 0, len(spec.Services))
	for _, serviceID := range spec.Services {
		service, err := client.GetService(serviceID)
		if err != nil {
			return fmt.Errorf("error fetching service %s: %w", serviceID, err)
		}
		services = append(services, *service)
	}

	return ctx.Metadata.Set(ListIncidentsNodeMetadata{
		Services: services,
	})
}

func (l *ListIncidents) Execute(ctx core.ExecutionContext) error {
	spec := ListIncidentsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// List incidents, optionally filtered by services
	incidents, err := client.ListIncidents(spec.Services)
	if err != nil {
		return fmt.Errorf("failed to list incidents: %v", err)
	}

	// Determine the output channel based on whether incidents exist
	channel := l.determineOutputChannel(incidents)

	// Build the response data
	responseData := map[string]any{
		"incidents": incidents,
		"total":     len(incidents),
	}

	return ctx.ExecutionState.Emit(
		channel,
		"pagerduty.incidents.list",
		[]any{responseData},
	)
}

// determineOutputChannel determines which output channel to emit to based on
// the urgency of open incidents:
// - "clear" if no incidents
// - "high" if any high urgency incidents exist
// - "low" if only low urgency incidents exist
func (l *ListIncidents) determineOutputChannel(incidents []Incident) string {
	if len(incidents) == 0 {
		return ChannelNameClear
	}

	for _, incident := range incidents {
		if incident.Urgency == "high" {
			return ChannelNameHigh
		}
	}

	return ChannelNameLow
}

func (l *ListIncidents) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *ListIncidents) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *ListIncidents) Actions() []core.Action {
	return []core.Action{}
}

func (l *ListIncidents) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (l *ListIncidents) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
