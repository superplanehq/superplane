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

type GetCheckRule struct{}

type GetCheckRuleSpec struct {
	CheckRule string `mapstructure:"checkRule"`
	Dataset   string `mapstructure:"dataset"`
}

func (c *GetCheckRule) Name() string {
	return "dash0.getCheckRule"
}

func (c *GetCheckRule) Label() string {
	return "Get Check Rule"
}

func (c *GetCheckRule) Description() string {
	return "Retrieve a check rule (Prometheus alert rule) from Dash0 by ID or origin"
}

func (c *GetCheckRule) Documentation() string {
	return `The Get Check Rule component retrieves the full configuration of an existing check rule (Prometheus alert rule) from Dash0.

## Use Cases

- **Configuration review**: Fetch current check rule settings for audit or documentation
- **Workflow integration**: Retrieve check rule details to use in subsequent workflow steps
- **Health monitoring**: Check if alert rules are properly configured

## Configuration

- **Check Rule**: The ID or origin of the check rule to retrieve (from Dash0)
- **Dataset**: The Dash0 dataset the check rule belongs to (defaults to "default")

## Output

Returns the complete check rule configuration from the Dash0 API, including:
- Name and expression (PromQL query)
- Thresholds (degraded and critical)
- Evaluation settings (interval, for, keepFiringFor)
- Labels and annotations
- Enabled status`
}

func (c *GetCheckRule) Icon() string {
	return "bell"
}

func (c *GetCheckRule) Color() string {
	return "blue"
}

func (c *GetCheckRule) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetCheckRule) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "checkRule",
			Label:       "Check Rule",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The check rule to retrieve",
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

func (c *GetCheckRule) Setup(ctx core.SetupContext) error {
	spec := GetCheckRuleSpec{}
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

func (c *GetCheckRule) Execute(ctx core.ExecutionContext) error {
	spec := GetCheckRuleSpec{}
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

	data, err := client.GetCheckRule(spec.CheckRule, dataset)
	if err != nil {
		return fmt.Errorf("failed to get check rule: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"dash0.checkRule.fetched",
		[]any{data},
	)
}

func (c *GetCheckRule) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetCheckRule) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetCheckRule) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetCheckRule) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *GetCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
