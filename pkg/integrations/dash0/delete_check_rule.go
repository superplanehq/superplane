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

type DeleteCheckRule struct{}

type DeleteCheckRuleSpec struct {
	CheckRule string `mapstructure:"checkRule"`
	Dataset   string `mapstructure:"dataset"`
}

func (c *DeleteCheckRule) Name() string {
	return "dash0.deleteCheckRule"
}

func (c *DeleteCheckRule) Label() string {
	return "Delete Check Rule"
}

func (c *DeleteCheckRule) Description() string {
	return "Delete a check rule (Prometheus alert rule) from Dash0 by ID or origin"
}

func (c *DeleteCheckRule) Documentation() string {
	return `The Delete Check Rule component removes a check rule (Prometheus alert rule) from Dash0 by its ID or origin. Use the check rule ID from a Create/Get/Update output or from the Dash0 dashboard.

## Use Cases

- **Cleanup**: Remove obsolete or test check rules
- **Automation**: Delete check rules as part of automated workflows
- **Resource management**: Clean up check rules when services are decommissioned

## Configuration

- **Check Rule**: The Dash0 check rule ID or origin to delete (required)
- **Dataset**: The dataset the check rule belongs to (defaults to "default")

## Output

Returns a confirmation payload indicating successful deletion.`
}

func (c *DeleteCheckRule) Icon() string {
	return "bell"
}

func (c *DeleteCheckRule) Color() string {
	return "red"
}

func (c *DeleteCheckRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteCheckRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkRule",
			Label:       "Check Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The check rule to delete",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "check-rule",
				},
			},
		},
		{
			Name:        "dataset",
			Label:       "Dataset",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Default:     "default",
			Description: "The dataset the check rule belongs to",
		},
	}
}

func (c *DeleteCheckRule) Setup(ctx core.SetupContext) error {
	spec := DeleteCheckRuleSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if strings.TrimSpace(spec.CheckRule) == "" {
		return errors.New("checkRule is required")
	}

	if strings.TrimSpace(spec.Dataset) == "" {
		return errors.New("dataset is required")
	}

	err = resolveCheckRuleMetadata(ctx, spec.CheckRule, spec.Dataset)
	if err != nil {
		return fmt.Errorf("error resolving check rule metadata: %v", err)
	}

	return nil
}

func (c *DeleteCheckRule) Execute(ctx core.ExecutionContext) error {
	spec := DeleteCheckRuleSpec{}
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

	data, err := client.DeleteCheckRule(spec.CheckRule, dataset)
	if err != nil {
		return fmt.Errorf("failed to delete check rule: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.checkRule.deleted",
		[]any{data},
	)
}

func (c *DeleteCheckRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteCheckRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteCheckRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *DeleteCheckRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *DeleteCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
