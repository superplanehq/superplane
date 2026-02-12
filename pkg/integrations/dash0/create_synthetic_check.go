package dash0

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const CreateSyntheticCheckPayloadType = "dash0.synthetic.check.created"

// CreateSyntheticCheck creates Dash0 synthetic checks via configuration API.
type CreateSyntheticCheck struct{}

// Name returns the stable component identifier.
func (c *CreateSyntheticCheck) Name() string {
	return "dash0.createSyntheticCheck"
}

// Label returns the display name used in the workflow builder.
func (c *CreateSyntheticCheck) Label() string {
	return "Create Synthetic Check"
}

// Description returns a short summary of component behavior.
func (c *CreateSyntheticCheck) Description() string {
	return "Create a synthetic check in Dash0 configuration API"
}

// Documentation returns markdown help shown in the component docs panel.
func (c *CreateSyntheticCheck) Documentation() string {
	return `The Create Synthetic Check component creates a Dash0 synthetic check using the configuration API.

## Use Cases

- **Service onboarding**: Create synthetic checks when new services are deployed
- **Environment bootstrap**: Provision baseline uptime checks in new environments
- **Automation workflows**: Create checks from CI/CD or incident workflows

## Configuration

- **Origin or ID (Optional)**: Custom synthetic check identifier. If omitted, SuperPlane generates one.
- **Name**: Human-readable synthetic check name
- **Enabled**: Whether the synthetic check is enabled
- **Plugin Kind**: Synthetic check plugin type (currently HTTP)
- **Method**: HTTP method for request checks
- **URL**: Target URL for the synthetic check
- **Headers (Optional)**: Request header key/value pairs
- **Request Body (Optional)**: HTTP request body (useful for POST/PUT/PATCH)

## Output

Emits:
- **originOrId**: Synthetic check identifier used for the API request
- **response**: Raw Dash0 API response`
}

// Icon returns the Lucide icon name for this component.
func (c *CreateSyntheticCheck) Icon() string {
	return "plus-circle"
}

// Color returns the node color used in the UI.
func (c *CreateSyntheticCheck) Color() string {
	return "blue"
}

// OutputChannels declares the channel emitted by this action.
func (c *CreateSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration defines fields required to create synthetic checks.
func (c *CreateSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "originOrId",
			Label:       "Origin or ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional synthetic check origin/ID. Leave empty to auto-generate.",
			Placeholder: "superplane.synthetic.check",
		},
		{
			Name:        "name",
			Label:       "Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Human-readable synthetic check name",
			Placeholder: "checkout-health",
		},
		{
			Name:        "enabled",
			Label:       "Enabled",
			Type:        configuration.FieldTypeBool,
			Required:    true,
			Default:     true,
			Description: "Enable or disable the synthetic check",
		},
		{
			Name:     "pluginKind",
			Label:    "Plugin Kind",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "http",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "HTTP", Value: "http"},
					},
				},
			},
			Description: "Synthetic check plugin kind",
		},
		{
			Name:     "method",
			Label:    "Method",
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
						{Label: "OPTIONS", Value: "options"},
					},
				},
			},
			Description: "HTTP method used for the synthetic check request",
		},
		{
			Name:        "url",
			Label:       "URL",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Target URL for the synthetic check request",
			Placeholder: "https://www.example.com/health",
		},
		{
			Name:        "headers",
			Label:       "Headers",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional request header key/value pairs",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Header",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "key",
								Label:              "Key",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
		{
			Name:        "requestBody",
			Label:       "Request Body",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "Optional HTTP request body",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"post", "put", "patch"}},
			},
		},
	}
}

// Setup validates component configuration during save/setup.
func (c *CreateSyntheticCheck) Setup(ctx core.SetupContext) error {
	scope := "dash0.createSyntheticCheck setup"
	config := UpsertSyntheticCheckConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if _, err := buildSyntheticCheckSpecificationFromConfiguration(config, scope); err != nil {
		return err
	}

	return nil
}

// ProcessQueueItem delegates queue processing to default behavior.
func (c *CreateSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute creates a synthetic check and emits API response payload.
func (c *CreateSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	scope := "dash0.createSyntheticCheck execute"
	config := UpsertSyntheticCheckConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	specification, err := buildSyntheticCheckSpecificationFromConfiguration(config, scope)
	if err != nil {
		return err
	}

	originOrID := strings.TrimSpace(config.OriginOrID)
	if originOrID == "" {
		originOrID = fmt.Sprintf("superplane-synthetic-%s", uuid.NewString()[:8])
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: create client: %w", scope, err)
	}

	response, err := client.UpsertSyntheticCheck(originOrID, specification)
	if err != nil {
		return fmt.Errorf("%s: create synthetic check %q: %w", scope, originOrID, err)
	}

	payload := map[string]any{
		"originOrId": originOrID,
		"response":   response,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateSyntheticCheckPayloadType, []any{payload})
}

// Actions returns no manual actions for this component.
func (c *CreateSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this component has no actions.
func (c *CreateSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook is unused because this component does not receive webhooks.
func (c *CreateSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel is a no-op because execution is synchronous and short-lived.
func (c *CreateSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (c *CreateSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
