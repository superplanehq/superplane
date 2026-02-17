package dash0

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateHTTPSyntheticCheck struct{}

// UpdateHTTPSyntheticCheckSpec has CheckID plus the same spec fields as create (full replacement).
type UpdateHTTPSyntheticCheckSpec struct {
	CheckID    string           `mapstructure:"checkId"`
	Name       string           `mapstructure:"name"`
	Dataset    string           `mapstructure:"dataset"`
	Request    RequestSpec      `mapstructure:"request"`
	Schedule   ScheduleSpec     `mapstructure:"schedule"`
	Assertions *[]AssertionSpec `mapstructure:"assertions"`
	Retries    *RetrySpec       `mapstructure:"retries"`
}

func (c *UpdateHTTPSyntheticCheck) Name() string {
	return "dash0.updateHttpSyntheticCheck"
}

func (c *UpdateHTTPSyntheticCheck) Label() string {
	return "Update HTTP Synthetic Check"
}

func (c *UpdateHTTPSyntheticCheck) Description() string {
	return "Update an existing HTTP synthetic check in Dash0 by ID"
}

func (c *UpdateHTTPSyntheticCheck) Documentation() string {
	return `The Update HTTP Synthetic Check component updates an existing synthetic check in Dash0. Use the check ID from a previous Create HTTP Synthetic Check output (e.g. metadata.labels["dash0.com/id"]) or from the Dash0 dashboard.

## Configuration

- **Check ID**: The Dash0 synthetic check ID to update (required).
- **Dataset**: The dataset the check belongs to (defaults to "default").
- **Name**, **Request**, **Schedule**, **Assertions**, **Retries**: Same as Create HTTP Synthetic Check; the full spec is sent to replace the existing check.`
}

func (c *UpdateHTTPSyntheticCheck) Icon() string {
	return "activity"
}

func (c *UpdateHTTPSyntheticCheck) Color() string {
	return "blue"
}

func (c *UpdateHTTPSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateHTTPSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkId",
			Label:       "Check ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The Dash0 synthetic check ID to update (e.g. from Create HTTP Synthetic Check output or dashboard)",
			Placeholder: "64617368-3073-796e-7468-abc123def456",
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Display name of the synthetic check",
			Placeholder: "Login API health check",
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset the synthetic check belongs to",
		},
		{
			Name:        "request",
			Label:       "Request",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "HTTP request configuration",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: requestObjectSchema(),
				},
			},
		},
		{
			Name:        "schedule",
			Label:       "Schedule",
			Type:        configuration.FieldTypeObject,
			Required:    true,
			Description: "Schedule configuration for the synthetic check",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: scheduleObjectSchema(),
				},
			},
		},
		{
			Name:        "assertions",
			Label:       "Assertions",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Conditions the synthetic check must satisfy.",
			Default:     `[{"kind":"status_code","severity":"critical","operator":"is","value":"200"}]`,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Assertion",
					ItemDefinition: &configuration.ListItemDefinition{
						Type:   configuration.FieldTypeObject,
						Schema: AssertionFieldSchema(),
					},
				},
			},
		},
		{
			Name:        "retries",
			Label:       "Retries",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Default:     map[string]any{"attempts": 3, "delay": "1s"},
			Description: "Retry configuration for failed checks",
			TypeOptions: &configuration.TypeOptions{
				Object: &configuration.ObjectTypeOptions{
					Schema: retriesObjectSchema(),
				},
			},
		},
	}
}

// requestObjectSchema returns the request object fields (shared so update can use without depending on create's full config).
func requestObjectSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Target URL to monitor",
			Placeholder: "https://api.example.com/health",
		},
		{
			Name:     "method",
			Label:    "HTTP Method",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "get",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "get"},
						{Label: "POST", Value: "post"},
						{Label: "PUT", Value: "put"},
						{Label: "PATCH", Value: "patch"},
						{Label: "DELETE", Value: "delete"},
						{Label: "HEAD", Value: "head"},
					},
				},
			},
		},
		{
			Name:    "redirects",
			Label:   "Redirects",
			Type:    configuration.FieldTypeSelect,
			Default: "follow",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Follow", Value: "follow"},
						{Label: "Do not follow", Value: "do_not_follow"},
					},
				},
			},
			Description: "Whether to follow HTTP redirects",
		},
		{
			Name:    "allowInsecure",
			Label:   "Allow Insecure TLS",
			Type:    configuration.FieldTypeSelect,
			Default: "false",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "No", Value: "false"},
						{Label: "Yes", Value: "true"},
					},
				},
			},
			Description: "Skip TLS certificate validation",
		},
		{
			Name:      "headers",
			Label:     "Headers",
			Type:      configuration.FieldTypeList,
			Required:  false,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{Name: "name", Label: "Name", Type: configuration.FieldTypeString, Required: true, DisallowExpression: true, Placeholder: "Content-Type"},
							{Name: "value", Label: "Value", Type: configuration.FieldTypeString, Required: true, Placeholder: "application/json"},
						},
					},
				},
			},
			Description: "Custom HTTP request headers",
		},
		{
			Name:        "body",
			Label:       "Request Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Request body payload",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"post", "put", "patch"}},
			},
		},
	}
}

func scheduleObjectSchema() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "1m",
			Description: "How often the check runs (e.g. 30s, 1m, 5m, 1h, 2d)",
			Placeholder: "1m",
		},
		{
			Name:     "locations",
			Label:    "Locations",
			Type:     configuration.FieldTypeMultiSelect,
			Required: true,
			Default:  []string{"de-frankfurt"},
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Frankfurt (DE)", Value: "de-frankfurt"},
						{Label: "Oregon (US)", Value: "us-oregon"},
						{Label: "North Virginia (US)", Value: "us-north-virginia"},
						{Label: "London (UK)", Value: "uk-london"},
						{Label: "Brussels (BE)", Value: "be-brussels"},
						{Label: "Melbourne (AU)", Value: "au-melbourne"},
					},
				},
			},
			Description: "Locations to run the synthetic check from",
		},
		{
			Name:      "strategy",
			Label:     "Execution Strategy",
			Type:      configuration.FieldTypeSelect,
			Required:  false,
			Togglable: true,
			Default:   "all_locations",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "All locations", Value: "all_locations"},
						{Label: "Round-robin", Value: "round_robin"},
					},
				},
			},
			Description: "How checks are distributed across locations",
		},
	}
}

func retriesObjectSchema() []configuration.Field {
	return []configuration.Field{
		{Name: "attempts", Label: "Attempts", Type: configuration.FieldTypeNumber, Required: true, Default: "3", Description: "Number of retry attempts on failure"},
		{Name: "delay", Label: "Delay", Type: configuration.FieldTypeString, Required: true, Default: "1s", Description: "Delay between retries", Placeholder: "1s"},
	}
}

func (c *UpdateHTTPSyntheticCheck) Setup(ctx core.SetupContext) error {
	spec := UpdateHTTPSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.CheckID) == "" {
		return errors.New("checkId is required")
	}

	if spec.Request.URL == "" {
		return errors.New("url is required")
	}

	if !strings.HasPrefix(spec.Request.URL, "http://") && !strings.HasPrefix(spec.Request.URL, "https://") {
		return errors.New("url must start with http:// or https://")
	}

	if len(spec.Schedule.Locations) == 0 {
		return errors.New("at least one location is required")
	}

	return nil
}

func (c *UpdateHTTPSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	spec := UpdateHTTPSyntheticCheckSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	dataset := spec.Dataset
	if dataset == "" {
		dataset = "default"
	}

	request := BuildSyntheticCheckRequest(
		spec.Name,
		spec.Request,
		spec.Schedule,
		BuildSyntheticCheckAssertions(spec.Assertions),
		spec.Retries,
	)

	data, err := client.UpdateSyntheticCheck(spec.CheckID, request, dataset)
	if err != nil {
		return fmt.Errorf("failed to update synthetic check: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.syntheticCheck.updated",
		[]any{data},
	)
}

func (c *UpdateHTTPSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateHTTPSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateHTTPSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

func (c *UpdateHTTPSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *UpdateHTTPSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *UpdateHTTPSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
