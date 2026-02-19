package servicenow

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ChannelNameClear = "clear"
	ChannelNameLow   = "low"
	ChannelNameHigh  = "high"
)

type GetIncidents struct{}

type GetIncidentsSpec struct {
	AssignmentGroup string `json:"assignmentGroup" mapstructure:"assignmentGroup"`
	AssignedTo      string `json:"assignedTo" mapstructure:"assignedTo"`
	Caller          string `json:"caller" mapstructure:"caller"`
	Category        string `json:"category" mapstructure:"category"`
	Subcategory     string `json:"subcategory" mapstructure:"subcategory"`
	Service         string `json:"service" mapstructure:"service"`
	State           string `json:"state" mapstructure:"state"`
	Urgency         string `json:"urgency" mapstructure:"urgency"`
	Impact          string `json:"impact" mapstructure:"impact"`
	Priority        string `json:"priority" mapstructure:"priority"`
	Limit           int    `json:"limit" mapstructure:"limit"`
}

func (c *GetIncidents) Name() string {
	return "servicenow.getIncidents"
}

func (c *GetIncidents) Label() string {
	return "Get Incidents"
}

func (c *GetIncidents) Description() string {
	return "Query ServiceNow for incidents matching the specified filters"
}

func (c *GetIncidents) Documentation() string {
	return `The Get Incidents component queries ServiceNow for incidents and routes execution based on urgency levels.

## Use Cases

- **Health checks**: Check for active incidents and route based on severity
- **Incident monitoring**: Monitor incident status across assignment groups
- **Automated response**: Trigger workflows based on incident presence
- **Reporting**: Collect incident data for reporting or analysis

## Configuration

All filters are optional. Leave empty to query all incidents.

- **Assignment Group**: Filter by assignment group
- **Assigned To**: Filter by assigned user
- **Caller**: Filter by caller
- **Category**: Filter by category
- **Subcategory**: Filter by subcategory (depends on category)
- **Service**: Filter by business service
- **State**: Filter by incident state (supports expressions for comma-separated values)
- **Urgency**: Filter by urgency level (supports expressions for comma-separated values)
- **Impact**: Filter by impact level (supports expressions for comma-separated values)
- **Priority**: Filter by priority level (supports expressions for comma-separated values)
- **Limit**: Maximum number of incidents to return (default 10)

## Output Channels

- **Clear**: No incidents matched the filters
- **Low**: Incidents found, but none are high urgency
- **High**: At least one high-urgency incident found

## Output

Returns a list of incidents with:
- **sys_id**: Unique identifier
- **number**: Human-readable incident number
- **short_description**: Incident summary
- **state**: Current state
- **urgency**: Urgency level
- **impact**: Impact level`
}

func (c *GetIncidents) Icon() string {
	return "servicenow"
}

func (c *GetIncidents) Color() string {
	return "gray"
}

func (c *GetIncidents) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ChannelNameClear, Label: "Clear", Description: "No incidents matched the filters"},
		{Name: ChannelNameLow, Label: "Low", Description: "Incidents found, but all are low or medium urgency"},
		{Name: ChannelNameHigh, Label: "High", Description: "At least one high-urgency incident found"},
	}
}

func (c *GetIncidents) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "assignmentGroup",
			Label:    "Assignment Group",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by assignment group",
			Placeholder: "Select an assignment group",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "assignment_group",
				},
			},
		},
		{
			Name:     "assignedTo",
			Label:    "Assigned To",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by assigned user",
			Placeholder: "Select a user",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "user",
					Parameters: []configuration.ParameterRef{
						{
							Name: "assignmentGroup",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "assignmentGroup",
							},
						},
					},
				},
			},
		},
		{
			Name:     "caller",
			Label:    "Caller",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by caller",
			Placeholder: "Select a user",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "user",
				},
			},
		},
		{
			Name:     "category",
			Label:    "Category",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by category",
			Placeholder: "Select a category",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "category",
				},
			},
		},
		{
			Name:     "subcategory",
			Label:    "Subcategory",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by subcategory",
			Placeholder: "Select a subcategory",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "subcategory",
					Parameters: []configuration.ParameterRef{
						{
							Name: "category",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "category",
							},
						},
					},
				},
			},
		},
		{
			Name:     "service",
			Label:    "Service",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by business service",
			Placeholder: "Select a service",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "service",
				},
			},
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by state",
			Placeholder: "Select a state",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "state",
				},
			},
		},
		{
			Name:     "urgency",
			Label:    "Urgency",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by urgency",
			Placeholder: "Select an urgency",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "urgency",
				},
			},
		},
		{
			Name:     "impact",
			Label:    "Impact",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by impact",
			Placeholder: "Select an impact",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "impact",
				},
			},
		},
		{
			Name:     "priority",
			Label:    "Priority",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "Filter incidents by priority",
			Placeholder: "Select a priority",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "priority",
				},
			},
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     10,
			Description: "Maximum number of incidents to return",
		},
	}
}

func (c *GetIncidents) Setup(ctx core.SetupContext) error {
	var existing NodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if existing.InstanceURL != "" {
		return nil
	}

	spec := GetIncidentsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	metadata, err := resolveResourceMetadata(client, resourceSpec{
		AssignmentGroup: spec.AssignmentGroup,
		AssignedTo:      spec.AssignedTo,
		Caller:          spec.Caller,
	})
	if err != nil {
		return err
	}

	return ctx.Metadata.Set(metadata)
}

func (c *GetIncidents) Execute(ctx core.ExecutionContext) error {
	spec := GetIncidentsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	query := buildQuery(spec)
	limit := spec.Limit
	if limit <= 0 {
		limit = 10
	}

	incidents, err := client.GetIncidents(query, limit)
	if err != nil {
		return fmt.Errorf("failed to get incidents: %w", err)
	}

	channel := c.determineOutputChannel(incidents)

	responseData := map[string]any{
		"incidents": incidents,
		"total":     len(incidents),
	}

	return ctx.ExecutionState.Emit(
		channel,
		PayloadTypeIncidents,
		[]any{responseData},
	)
}

func buildQuery(spec GetIncidentsSpec) string {
	parts := []string{}

	if spec.AssignmentGroup != "" {
		parts = append(parts, "assignment_group="+spec.AssignmentGroup)
	}

	if spec.AssignedTo != "" {
		parts = append(parts, "assigned_to="+spec.AssignedTo)
	}

	if spec.Caller != "" {
		parts = append(parts, "caller_id="+spec.Caller)
	}

	if spec.Category != "" {
		parts = append(parts, "category="+spec.Category)
	}

	if spec.Subcategory != "" {
		parts = append(parts, "subcategory="+spec.Subcategory)
	}

	if spec.Service != "" {
		parts = append(parts, "business_service="+spec.Service)
	}

	if spec.State != "" {
		parts = append(parts, "stateIN"+spec.State)
	}

	if spec.Urgency != "" {
		parts = append(parts, "urgencyIN"+spec.Urgency)
	}

	if spec.Impact != "" {
		parts = append(parts, "impactIN"+spec.Impact)
	}

	if spec.Priority != "" {
		parts = append(parts, "priorityIN"+spec.Priority)
	}

	return strings.Join(parts, "^")
}

func (c *GetIncidents) determineOutputChannel(incidents []IncidentRecord) string {
	if len(incidents) == 0 {
		return ChannelNameClear
	}

	for _, incident := range incidents {
		if incident.Urgency == "1" {
			return ChannelNameHigh
		}
	}

	return ChannelNameLow
}

func (c *GetIncidents) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetIncidents) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetIncidents) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetIncidents) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetIncidents) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetIncidents) Cleanup(ctx core.SetupContext) error {
	return nil
}
