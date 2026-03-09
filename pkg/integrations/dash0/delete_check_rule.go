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
	CheckRuleID string `mapstructure:"checkRuleId"`
	Dataset     string `mapstructure:"dataset"`
}
type DeleteCheckRuleNodeMetadata struct {
	CheckRuleName string `json:"checkRuleName" mapstructure:"checkRuleName"`
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

- **Check Rule ID**: The Dash0 check rule ID or origin to delete (required)
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
			Name:        "checkRuleId",
			Label:       "Check Rule ID",
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

	if strings.TrimSpace(spec.CheckRuleID) == "" {
		return errors.New("checkRuleId is required")
	}

	if strings.TrimSpace(spec.Dataset) == "" {
		return errors.New("dataset is required")
	}
	// Try to fetch and store the check rule name as metadata
	var nodeMetadata DeleteCheckRuleNodeMetadata
	err = mapstructure.Decode(ctx.Metadata.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("error decoding metadata: %v", err)
	}

	if nodeMetadata.CheckRuleName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client during setup: %v", err)
	}

	checkRule, err := client.GetCheckRule(spec.CheckRuleID, spec.Dataset)
	if err != nil {
		return fmt.Errorf("failed to get check rule during setup: %v", err)
	}

	checkRuleName := ""
	if name, ok := checkRule["name"].(string); ok {
		checkRuleName = name
	} else if id, ok := checkRule["id"].(string); ok {
		checkRuleName = id
	}

	if checkRuleName != "" {
		return ctx.Metadata.Set(DeleteCheckRuleNodeMetadata{
			CheckRuleName: checkRuleName,
		})
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

	data, err := client.DeleteCheckRule(spec.CheckRuleID, dataset)
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

func (c *DeleteCheckRule) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *DeleteCheckRule) Cleanup(ctx core.SetupContext) error {
	return nil
}
