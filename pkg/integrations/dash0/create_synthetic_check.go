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
- **Specification (JSON)**: Synthetic check JSON payload accepted by Dash0 config API

Example specification:
` + "```json" + `
{
  "kind": "Dash0SyntheticCheck",
  "metadata": {
    "name": "checkout-health"
  },
  "spec": {
    "enabled": true,
    "plugin": {
      "kind": "http",
      "spec": {
        "request": {
          "method": "get",
          "url": "https://www.example.com/health"
        }
      }
    }
  }
}
` + "```" + `

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
			Name:        "spec",
			Label:       "Specification (JSON)",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Synthetic check specification as a JSON object",
			Placeholder: "{\"kind\":\"Dash0SyntheticCheck\",\"metadata\":{\"name\":\"examplecom\"},\"spec\":{\"enabled\":true,\"plugin\":{\"kind\":\"http\",\"spec\":{\"request\":{\"method\":\"get\",\"url\":\"https://www.example.com\"}}}}}",
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

	specification, err := parseSpecification(config.Spec, "spec", scope)
	if err != nil {
		return err
	}

	return validateSyntheticCheckSpecification(specification, "spec", scope)
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

	specification, err := parseSpecification(config.Spec, "spec", scope)
	if err != nil {
		return err
	}

	if err := validateSyntheticCheckSpecification(specification, "spec", scope); err != nil {
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
