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

const CreateCheckRulePayloadType = "dash0.check.rule.created"

// CreateCheckRule creates Dash0 check rules via configuration API.
type CreateCheckRule struct{}

// Name returns the stable component identifier.
func (c *CreateCheckRule) Name() string {
	return "dash0.createCheckRule"
}

// Label returns the display name used in the workflow builder.
func (c *CreateCheckRule) Label() string {
	return "Create Check Rule"
}

// Description returns a short summary of component behavior.
func (c *CreateCheckRule) Description() string {
	return "Create a check rule in Dash0 configuration API"
}

// Documentation returns markdown help shown in the component docs panel.
func (c *CreateCheckRule) Documentation() string {
	return `The Create Check Rule component creates a Dash0 check rule (alert rule) using the configuration API.

## Use Cases

- **Service onboarding**: Create baseline alert rules for new services
- **Automated governance**: Enforce standard alerting from workflow templates
- **Deployment automation**: Create environment-specific rules during rollouts

## Configuration

- **Origin or ID (Optional)**: Custom check rule identifier. If omitted, SuperPlane generates one.
- **Rule Specification (JSON)**: Check rule payload as JSON object.
  Accepts Dash0 check rule shape (name + expression) or Prometheus-style
  groups/rules shape with exactly one alert rule.

## Output

Emits:
- **originOrId**: Check rule identifier used for the API request
- **response**: Raw Dash0 API response`
}

// Icon returns the Lucide icon name for this component.
func (c *CreateCheckRule) Icon() string {
	return "plus-circle"
}

// Color returns the node color used in the UI.
func (c *CreateCheckRule) Color() string {
	return "blue"
}

// OutputChannels declares the channel emitted by this action.
func (c *CreateCheckRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration defines fields required to create check rules.
func (c *CreateCheckRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "originOrId",
			Label:       "Origin or ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional check rule origin/ID. Leave empty to auto-generate.",
			Placeholder: "superplane.check.rule",
		},
		{
			Name:        "spec",
			Label:       "Rule Specification (JSON)",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Check rule specification as a JSON object",
			Placeholder: "{\"name\":\"Checkout errors\",\"expression\":\"sum(rate(http_requests_total{service=\\\"checkout\\\",status=~\\\"5..\\\"}[5m])) > 0\",\"for\":\"5m\",\"labels\":{\"severity\":\"warning\"},\"annotations\":{\"summary\":\"Checkout 5xx errors are above baseline\"}}",
		},
	}
}

// Setup validates component configuration during save/setup.
func (c *CreateCheckRule) Setup(ctx core.SetupContext) error {
	scope := "dash0.createCheckRule setup"
	config := UpsertCheckRuleConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if _, err := parseCheckRuleSpecification(config.Spec, "spec", scope); err != nil {
		return err
	}

	return nil
}

// ProcessQueueItem delegates queue processing to default behavior.
func (c *CreateCheckRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute creates a check rule and emits API response payload.
func (c *CreateCheckRule) Execute(ctx core.ExecutionContext) error {
	scope := "dash0.createCheckRule execute"
	config := UpsertCheckRuleConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	specification, err := parseCheckRuleSpecification(config.Spec, "spec", scope)
	if err != nil {
		return err
	}

	originOrID := strings.TrimSpace(config.OriginOrID)
	if originOrID == "" {
		originOrID = fmt.Sprintf("superplane-check-rule-%s", uuid.NewString()[:8])
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: create client: %w", scope, err)
	}

	response, err := client.UpsertCheckRule(originOrID, specification)
	if err != nil {
		return fmt.Errorf("%s: create check rule %q: %w", scope, originOrID, err)
	}

	payload := map[string]any{
		"originOrId": originOrID,
		"response":   response,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, CreateCheckRulePayloadType, []any{payload})
}

// Actions returns no manual actions for this component.
func (c *CreateCheckRule) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this component has no actions.
func (c *CreateCheckRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook is unused because this component does not receive webhooks.
func (c *CreateCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel is a no-op because execution is synchronous and short-lived.
func (c *CreateCheckRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (c *CreateCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
