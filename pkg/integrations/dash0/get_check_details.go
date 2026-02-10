package dash0

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetCheckDetailsPayloadType = "dash0.check.details.retrieved"

// GetCheckDetails fetches detailed check information from Dash0 alerting APIs.
type GetCheckDetails struct{}

// Name returns the stable component identifier.
func (c *GetCheckDetails) Name() string {
	return "dash0.getCheckDetails"
}

// Label returns the display name used in the workflow builder.
func (c *GetCheckDetails) Label() string {
	return "Get Check Details"
}

// Description returns a short summary of component behavior.
func (c *GetCheckDetails) Description() string {
	return "Get detailed information for a Dash0 check by ID"
}

// Documentation returns markdown help shown in the component docs panel.
func (c *GetCheckDetails) Documentation() string {
	return `The Get Check Details component fetches full context for a Dash0 check by ID.

## Use Cases

- **Alert enrichment**: Expand webhook payloads with full check context before notifying responders
- **Workflow branching**: Use check attributes (severity, thresholds, services) in downstream conditions
- **Incident automation**: Add rich check details to incident tickets or chat messages

## Configuration

- **Check ID**: The Dash0 check identifier to retrieve
- **Include History**: Include additional history data when supported by the Dash0 API

## Output

Emits a payload containing:
- **checkId**: Check identifier used in the request
- **details**: Raw details response from Dash0`
}

// Icon returns the Lucide icon name for this component.
func (c *GetCheckDetails) Icon() string {
	return "search"
}

// Color returns the node color used in the UI.
func (c *GetCheckDetails) Color() string {
	return "blue"
}

// OutputChannels declares the channel emitted by this action.
func (c *GetCheckDetails) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

// Configuration defines fields required to fetch check details.
func (c *GetCheckDetails) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkId",
			Label:       "Check ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Dash0 check identifier (for example from an On Alert Event trigger)",
			Placeholder: "{{ event.data.checkId }}",
		},
		{
			Name:        "includeHistory",
			Label:       "Include History",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Request additional history data when supported by Dash0",
		},
	}
}

// Setup validates component configuration during save/setup.
func (c *GetCheckDetails) Setup(ctx core.SetupContext) error {
	scope := "dash0.getCheckDetails setup"
	config := GetCheckDetailsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	if _, err := requireNonEmptyValue(config.CheckID, "checkId", scope); err != nil {
		return err
	}

	return nil
}

// ProcessQueueItem delegates queue processing to default behavior.
func (c *GetCheckDetails) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

// Execute fetches check details and emits normalized output payload.
func (c *GetCheckDetails) Execute(ctx core.ExecutionContext) error {
	scope := "dash0.getCheckDetails execute"
	config := GetCheckDetailsConfiguration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("%s: decode configuration: %w", scope, err)
	}

	checkID, err := requireNonEmptyValue(config.CheckID, "checkId", scope)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("%s: create client: %w", scope, err)
	}

	details, err := client.GetCheckDetails(checkID, config.IncludeHistory)
	if err != nil {
		return fmt.Errorf("%s: get check details for %q: %w", scope, checkID, err)
	}

	payload := map[string]any{
		"checkId": checkID,
		"details": details,
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetCheckDetailsPayloadType, []any{payload})
}

// Actions returns no manual actions for this component.
func (c *GetCheckDetails) Actions() []core.Action {
	return []core.Action{}
}

// HandleAction is unused because this component has no actions.
func (c *GetCheckDetails) HandleAction(ctx core.ActionContext) error {
	return nil
}

// HandleWebhook is unused because this component does not receive webhooks.
func (c *GetCheckDetails) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

// Cancel is a no-op because execution is synchronous and short-lived.
func (c *GetCheckDetails) Cancel(ctx core.ExecutionContext) error {
	return nil
}

// Cleanup is a no-op because no external resources are provisioned.
func (c *GetCheckDetails) Cleanup(ctx core.SetupContext) error {
	return nil
}
