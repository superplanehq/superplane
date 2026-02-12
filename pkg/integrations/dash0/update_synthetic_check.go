package dash0

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateSyntheticCheckPayloadType = "dash0.synthetic.check.updated"

// UpdateSyntheticCheck updates existing Dash0 synthetic checks via configuration API.
type UpdateSyntheticCheck struct{}

// Name returns the stable component identifier.
func (c *UpdateSyntheticCheck) Name() string {
	return "dash0.updateSyntheticCheck"
}

// Label returns the display name used in the workflow builder.
func (c *UpdateSyntheticCheck) Label() string {
	return "Update Synthetic Check"
}

// Description returns a short summary of component behavior.
func (c *UpdateSyntheticCheck) Description() string {
	return "Update an existing synthetic check in Dash0 configuration API"
}

// Documentation returns markdown help shown in the component docs panel.
func (c *UpdateSyntheticCheck) Documentation() string {
	return `The Update Synthetic Check component updates an existing Dash0 synthetic check.

## Use Cases

- **Change monitoring target**: Update URLs, probes, or assertions after infrastructure changes
- **Tune schedules**: Adjust check intervals as traffic and SLOs evolve
- **Workflow-driven changes**: Roll out check updates as part of deployment workflows

## Configuration

- **Synthetic Check**: Existing synthetic check origin/ID
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
func (c *UpdateSyntheticCheck) Icon() string {
	return "refresh-cw"
}

// Color returns the node color used in the UI.
func (c *UpdateSyntheticCheck) Color() string {
	return "blue"
}

// OutputChannels declares the channel emitted by this action.
func (c *UpdateSyntheticCheck) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration defines fields required to update synthetic checks.
func (c *UpdateSyntheticCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:      "originOrId",
			Label:     "Synthetic Check",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  true,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "synthetic-check",
				},
			},
			Description: "Synthetic check origin/ID to update",
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
func (c *UpdateSyntheticCheck) Setup(ctx core.SetupContext) error {
	scope := "dash0.updateSyntheticCheck setup"
	config := UpsertSyntheticCheckConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if _, err := requireNonEmptyValue(config.OriginOrID, "originOrId", scope); err != nil {
		return err
	}

	_, err := buildSyntheticCheckSpecificationFromConfiguration(config, scope)
	return err
}

// ProcessQueueItem delegates queue processing to default behavior.
func (c *UpdateSyntheticCheck) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute updates a synthetic check and emits API response payload.
func (c *UpdateSyntheticCheck) Execute(ctx core.ExecutionContext) error {
	scope := "dash0.updateSyntheticCheck execute"
	config := UpsertSyntheticCheckConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	originOrID, err := requireNonEmptyValue(config.OriginOrID, "originOrId", scope)
	if err != nil {
		return err
	}

	specification, err := buildSyntheticCheckSpecificationFromConfiguration(config, scope)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: create client: %w", scope, err)
	}

	response, err := client.UpsertSyntheticCheck(originOrID, specification)
	if err != nil {
		return fmt.Errorf("%s: update synthetic check %q: %w", scope, originOrID, err)
	}

	payload := map[string]any{
		"originOrId": originOrID,
		"response":   response,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UpdateSyntheticCheckPayloadType, []any{payload})
}

// Actions returns no manual actions for this component.
func (c *UpdateSyntheticCheck) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this component has no actions.
func (c *UpdateSyntheticCheck) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook is unused because this component does not receive webhooks.
func (c *UpdateSyntheticCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel is a no-op because execution is synchronous and short-lived.
func (c *UpdateSyntheticCheck) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (c *UpdateSyntheticCheck) Cleanup(ctx core.SetupContext) error {
	return nil
}
