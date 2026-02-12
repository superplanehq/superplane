package dash0

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const UpdateCheckRulePayloadType = "dash0.check.rule.updated"

// UpdateCheckRule updates existing Dash0 check rules via configuration API.
type UpdateCheckRule struct{}

// Name returns the stable component identifier.
func (c *UpdateCheckRule) Name() string {
	return "dash0.updateCheckRule"
}

// Label returns the display name used in the workflow builder.
func (c *UpdateCheckRule) Label() string {
	return "Update Check Rule"
}

// Description returns a short summary of component behavior.
func (c *UpdateCheckRule) Description() string {
	return "Update an existing check rule in Dash0 configuration API"
}

// Documentation returns markdown help shown in the component docs panel.
func (c *UpdateCheckRule) Documentation() string {
	return `The Update Check Rule component updates an existing Dash0 check rule.

## Use Cases

- **Threshold tuning**: Adjust alert sensitivity as service behavior changes
- **Rule maintenance**: Update labels, annotations, and notification routing
- **Operational automation**: Enable or disable rules from workflows

## Configuration

- **Check Rule**: Existing check rule origin/ID
- **Name**: Human-readable rule name
- **Expression**: Prometheus expression used by the rule
- **For (Optional)**: How long expression must remain true before firing
- **Interval (Optional)**: Evaluation interval override
- **Keep Firing For (Optional)**: Additional duration to keep firing after recovery
- **Labels (Optional)**: Label key/value pairs
- **Annotations (Optional)**: Annotation key/value pairs

## Output

Emits:
- **originOrId**: Check rule identifier used for the API request
- **response**: Raw Dash0 API response`
}

// Icon returns the Lucide icon name for this component.
func (c *UpdateCheckRule) Icon() string {
	return "refresh-cw"
}

// Color returns the node color used in the UI.
func (c *UpdateCheckRule) Color() string {
	return "blue"
}

// OutputChannels declares the channel emitted by this action.
func (c *UpdateCheckRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration defines fields required to update check rules.
func (c *UpdateCheckRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:      "originOrId",
			Label:     "Check Rule",
			Type:      configuration.FieldTypeIntegrationResource,
			Required:  true,
			Togglable: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "check-rule",
				},
			},
			Description: "Check rule origin/ID to update",
		},
		{
			Name:        "name",
			Label:       "Rule Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Name of the check rule",
			Placeholder: "Checkout errors",
		},
		{
			Name:        "expression",
			Label:       "Expression",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Prometheus expression evaluated by the rule",
			Placeholder: "sum(rate(http_requests_total{service=\"checkout\",status=~\"5..\"}[5m])) > 1",
		},
		{
			Name:        "for",
			Label:       "For",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional firing delay duration (for example: 5m)",
			Placeholder: "5m",
		},
		{
			Name:        "interval",
			Label:       "Interval",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional evaluation interval override (for example: 1m)",
			Placeholder: "1m",
		},
		{
			Name:        "keepFiringFor",
			Label:       "Keep Firing For",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional extra duration to keep alert firing after recovery",
			Placeholder: "10m",
		},
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional label key/value pairs",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Label",
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
			Name:        "annotations",
			Label:       "Annotations",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Optional annotation key/value pairs",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Annotation",
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
	}
}

// Setup validates component configuration during save/setup.
func (c *UpdateCheckRule) Setup(ctx core.SetupContext) error {
	scope := "dash0.updateCheckRule setup"
	config := UpsertCheckRuleConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if _, err := requireNonEmptyValue(config.OriginOrID, "originOrId", scope); err != nil {
		return err
	}

	if _, err := buildCheckRuleSpecification(config, scope); err != nil {
		return err
	}

	return nil
}

// ProcessQueueItem delegates queue processing to default behavior.
func (c *UpdateCheckRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute updates a check rule and emits API response payload.
func (c *UpdateCheckRule) Execute(ctx core.ExecutionContext) error {
	scope := "dash0.updateCheckRule execute"
	config := UpsertCheckRuleConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	originOrID, err := requireNonEmptyValue(config.OriginOrID, "originOrId", scope)
	if err != nil {
		return err
	}

	specification, err := buildCheckRuleSpecification(config, scope)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: create client: %w", scope, err)
	}

	response, err := client.UpsertCheckRule(originOrID, specification)
	if err != nil {
		return fmt.Errorf("%s: update check rule %q: %w", scope, originOrID, err)
	}

	payload := map[string]any{
		"originOrId": originOrID,
		"response":   response,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UpdateCheckRulePayloadType, []any{payload})
}

// Actions returns no manual actions for this component.
func (c *UpdateCheckRule) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this component has no actions.
func (c *UpdateCheckRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook is unused because this component does not receive webhooks.
func (c *UpdateCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel is a no-op because execution is synchronous and short-lived.
func (c *UpdateCheckRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (c *UpdateCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
