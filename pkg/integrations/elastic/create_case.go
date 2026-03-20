package elastic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type CreateCase struct{}

type CreateCaseConfiguration struct {
	Title       string   `json:"title" mapstructure:"title"`
	Description string   `json:"description" mapstructure:"description"`
	Severity    string   `json:"severity" mapstructure:"severity"`
	Owner       string   `json:"owner" mapstructure:"owner"`
	Tags        []string `json:"tags" mapstructure:"tags"`
}

func (c *CreateCase) Name() string  { return "elastic.createCase" }
func (c *CreateCase) Label() string { return "Create Case" }
func (c *CreateCase) Description() string {
	return "Create a new case in Kibana Security"
}
func (c *CreateCase) Icon() string  { return "elastic" }
func (c *CreateCase) Color() string { return "gray" }

func (c *CreateCase) Documentation() string {
	return `The Create Case component opens a new case in Kibana Security.

## Configuration

- **Title**: The case title
- **Severity**: Case severity (low, medium, high, or critical)
- **Owner**: The Kibana application that owns the case
- **Description**: A description of the case
- **Tags**: Optional list of tags to attach to the case

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The case ID assigned by Kibana
- ` + "`title`" + `: The case title
- ` + "`status`" + `: The initial case status
- ` + "`severity`" + `: The case severity
- ` + "`version`" + `: The case version (can be provided to later updates for explicit optimistic locking)
- ` + "`createdAt`" + `: The timestamp when the case was created`
}

func (c *CreateCase) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateCase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "title",
			Label:       "Title",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The title of the Kibana case.",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "low",
			Description: "The severity level of the case.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Low", Value: "low"},
						{Label: "Medium", Value: "medium"},
						{Label: "High", Value: "high"},
						{Label: "Critical", Value: "critical"},
					},
				},
			},
		},
		{
			Name:        "owner",
			Label:       "Owner",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     "cases",
			Description: "The Kibana application that owns the case.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Cases", Value: "cases"},
						{Label: "Security Solution", Value: "securitySolution"},
						{Label: "Observability", Value: "observability"},
					},
				},
			},
		},
		{
			Name:        "description",
			Label:       "Description",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "A description of the case.",
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Optional list of tags to attach to the case.",
		},
	}
}

func (c *CreateCase) Setup(ctx core.SetupContext) error {
	var config CreateCaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Title) == "" {
		return fmt.Errorf("title is required")
	}

	return nil
}

func (c *CreateCase) Execute(ctx core.ExecutionContext) error {
	var config CreateCaseConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.Title = strings.TrimSpace(config.Title)
	if config.Title == "" {
		return ctx.ExecutionState.Fail("error", "title is required")
	}

	if config.Severity == "" {
		config.Severity = "low"
	}

	if config.Owner == "" {
		config.Owner = "cases"
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create Elastic client: %v", err))
	}

	resp, err := client.CreateCase(config.Title, config.Description, config.Severity, config.Owner, config.Tags)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create case: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"elastic.case.created",
		[]any{map[string]any{
			"id":        resp.ID,
			"title":     resp.Title,
			"status":    resp.Status,
			"severity":  resp.Severity,
			"version":   resp.Version,
			"createdAt": resp.CreatedAt,
		}},
	)
}

func (c *CreateCase) Actions() []core.Action                  { return nil }
func (c *CreateCase) HandleAction(_ core.ActionContext) error { return nil }
func (c *CreateCase) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *CreateCase) Cleanup(_ core.SetupContext) error       { return nil }
func (c *CreateCase) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *CreateCase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
