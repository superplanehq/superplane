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

type CreateIncident struct{}

type CreateIncidentSpec struct {
	ShortDescription string  `json:"shortDescription" mapstructure:"shortDescription"`
	Description      string  `json:"description" mapstructure:"description"`
	State            string  `json:"state" mapstructure:"state"`
	Urgency          string  `json:"urgency" mapstructure:"urgency"`
	Impact           string  `json:"impact" mapstructure:"impact"`
	Category         string  `json:"category" mapstructure:"category"`
	Subcategory      string  `json:"subcategory" mapstructure:"subcategory"`
	AssignmentGroup  string  `json:"assignmentGroup" mapstructure:"assignmentGroup"`
	AssignedTo       string  `json:"assignedTo" mapstructure:"assignedTo"`
	Caller           string  `json:"caller" mapstructure:"caller"`
	ResolutionCode   *string `json:"resolutionCode,omitempty" mapstructure:"resolutionCode"`
	ResolutionNotes  *string `json:"resolutionNotes,omitempty" mapstructure:"resolutionNotes"`
	OnHoldReason     *string `json:"onHoldReason,omitempty" mapstructure:"onHoldReason"`
}

func (c *CreateIncident) Name() string {
	return "servicenow.createIncident"
}

func (c *CreateIncident) Label() string {
	return "Create Incident"
}

func (c *CreateIncident) Description() string {
	return "Create a new incident in ServiceNow"
}

func (c *CreateIncident) Documentation() string {
	return `The Create Incident component creates a new incident in ServiceNow using the Table API.

## Use Cases

- **Alert escalation**: Create incidents from monitoring alerts
- **Error tracking**: Automatically create incidents when errors are detected
- **Manual incident creation**: Create incidents from workflow events
- **Integration workflows**: Create incidents from external system events

## Required Permissions

The ServiceNow integration account needs:
- **itil** role — grants read/write access to the Incident table

## Configuration

- **Short Description**: A brief summary of the incident (required, supports expressions)
- **Description**: Detailed description of the incident (optional, supports expressions)
- **Urgency**: Incident urgency level (1-High, 2-Medium, 3-Low)
- **Impact**: Incident impact level (1-High, 2-Medium, 3-Low)
- **Category**: Incident category (select from list)
- **Subcategory**: Incident subcategory (depends on the selected category)
- **Assignment Group**: The group responsible for resolving the incident (select from list)
- **Assigned To**: The user assigned to resolve the incident (select from list)
- **Caller**: The user reporting the incident (select from list)

## Output

Returns the created incident object from the ServiceNow Table API, including:
- **sys_id**: Unique identifier
- **number**: Human-readable incident number (e.g. INC0010001)
- **state**: Current incident state
- **short_description**: Incident summary
- **created_on**: Creation timestamp`
}

func (c *CreateIncident) Icon() string {
	return "servicenow"
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
			Name:        "shortDescription",
			Label:       "Short Description",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "A brief summary of the incident",
		},
		{
			Name:        "urgency",
			Label:       "Urgency",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "2",
			Description: "How quickly the incident needs to be resolved",
			Placeholder: "Select an urgency",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "urgency",
				},
			},
		},
		{
			Name:        "impact",
			Label:       "Impact",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "2",
			Description: "The extent to which the incident affects the business",
			Placeholder: "Select an impact",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "impact",
				},
			},
		},
		{
			Name:     "description",
			Label:    "Description",
			Type:     configuration.FieldTypeText,
			Required: false,

			Description: "Detailed description of the incident",
		},
		{
			Name:     "category",
			Label:    "Category",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "The classification of the incident",
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

			Description: "Subcategory of the incident (depends on the selected category)",
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
			Name:     "assignmentGroup",
			Label:    "Assignment Group",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "The group responsible for resolving the incident",
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

			Description: "The user assigned to resolve the incident",
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

			Description: "The user reporting the incident",
			Placeholder: "Select a user",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "user",
				},
			},
		},
		{
			Name:     "state",
			Label:    "State",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,

			Description: "The current stage of the incident lifecycle",
			Placeholder: "Select a state",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "state",
				},
			},
		},
		{
			Name:        "onHoldReason",
			Label:       "On Hold Reason",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Reason the incident is on hold",
			Placeholder: "Select a reason",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "state", Values: []string{"3"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "state", Values: []string{"3"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "on_hold_reason",
				},
			},
		},
		{
			Name:        "resolutionCode",
			Label:       "Resolution Code",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "How the incident was resolved",
			Placeholder: "Select a resolution code",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "state", Values: []string{"6", "7"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "state", Values: []string{"6", "7"}},
			},
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "resolution_code",
				},
			},
		},
		{
			Name:        "resolutionNotes",
			Label:       "Resolution Notes",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Description: "Details about how the incident was resolved",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "state", Values: []string{"6", "7"}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "state", Values: []string{"6", "7"}},
			},
		},
	}
}

func (c *CreateIncident) Setup(ctx core.SetupContext) error {
	spec := CreateIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	if spec.ShortDescription == "" {
		return errors.New("shortDescription is required")
	}

	if spec.Urgency == "" {
		return errors.New("urgency is required")
	}

	if spec.Impact == "" {
		return errors.New("impact is required")
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

func (c *CreateIncident) Execute(ctx core.ExecutionContext) error {
	spec := CreateIncidentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	params := CreateIncidentParams{
		ShortDescription: spec.ShortDescription,
		Description:      spec.Description,
		State:            spec.State,
		Urgency:          spec.Urgency,
		Impact:           spec.Impact,
		Category:         spec.Category,
		Subcategory:      spec.Subcategory,
		AssignmentGroup:  spec.AssignmentGroup,
		AssignedTo:       spec.AssignedTo,
		Caller:           spec.Caller,
	}

	if spec.ResolutionCode != nil {
		params.ResolutionCode = *spec.ResolutionCode
	}

	if spec.ResolutionNotes != nil {
		params.ResolutionNotes = *spec.ResolutionNotes
	}

	if spec.OnHoldReason != nil {
		params.OnHoldReason = *spec.OnHoldReason
	}

	result, err := client.CreateIncident(params)
	if err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		PayloadTypeIncident,
		[]any{result},
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

func (c *CreateIncident) Cleanup(ctx core.SetupContext) error {
	return nil
}
