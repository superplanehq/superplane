package prometheus

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateRuleGroupNamespace struct{}

type UpdateRuleGroupNamespaceConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	WorkspaceID string `json:"workspace" mapstructure:"workspace"`
	Name        string `json:"namespace" mapstructure:"namespace"`
	Data        string `json:"data" mapstructure:"data"`
	ClientToken string `json:"clientToken" mapstructure:"clientToken"`
}

func (c *UpdateRuleGroupNamespace) Name() string {
	return "aws.prometheus.updateRuleGroupNamespace"
}

func (c *UpdateRuleGroupNamespace) Label() string {
	return "Prometheus • Update Rule Group Namespace"
}

func (c *UpdateRuleGroupNamespace) Description() string {
	return "Update a rule group namespace in an Amazon Managed Service for Prometheus workspace"
}

func (c *UpdateRuleGroupNamespace) Documentation() string {
	return `The Update Rule Group Namespace component updates the rule groups YAML for an existing Amazon Managed Service for Prometheus rule group namespace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Rule Group Namespace**: Target rule group namespace
- **Rule Groups YAML**: New Prometheus rule groups YAML file content
- **Client Token**: Optional idempotency token`
}

func (c *UpdateRuleGroupNamespace) Icon() string {
	return "aws"
}

func (c *UpdateRuleGroupNamespace) Color() string {
	return "gray"
}

func (c *UpdateRuleGroupNamespace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateRuleGroupNamespace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace containing the rule group namespace"),
		ruleGroupsNamespaceField("Rule Group Namespace", "Rule group namespace to update"),
		ruleGroupsNamespaceDataField(),
		clientTokenField(),
	}
}

func (c *UpdateRuleGroupNamespace) Setup(ctx core.SetupContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setRuleGroupsNamespaceNodeMetadata(ctx, resolveRuleGroupsNamespaceNodeMetadata(ctx, ruleGroupsNamespaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
		Name:        config.Name,
	}))
}

func (c *UpdateRuleGroupNamespace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateRuleGroupNamespace) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	namespace, err := client.PutRuleGroupsNamespace(PutRuleGroupsNamespaceInput{
		WorkspaceID: config.WorkspaceID,
		Name:        config.Name,
		Data:        config.Data,
		ClientToken: config.ClientToken,
	})
	if err != nil {
		return fmt.Errorf("failed to update Prometheus rule group namespace: %w", err)
	}

	output := map[string]any{
		"ruleGroupNamespace": ruleGroupsNamespaceOutput(namespace),
		"workspaceId":        config.WorkspaceID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.ruleGroupNamespace.updated",
		[]any{output},
	)
}

func (c *UpdateRuleGroupNamespace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateRuleGroupNamespace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateRuleGroupNamespace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateRuleGroupNamespace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateRuleGroupNamespace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *UpdateRuleGroupNamespace) decodeConfiguration(rawConfiguration any) (UpdateRuleGroupNamespaceConfiguration, error) {
	config := UpdateRuleGroupNamespaceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return UpdateRuleGroupNamespaceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.Name = strings.TrimSpace(config.Name)
	config.Data = strings.TrimSpace(config.Data)
	config.ClientToken = strings.TrimSpace(config.ClientToken)

	if config.Region == "" {
		return UpdateRuleGroupNamespaceConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return UpdateRuleGroupNamespaceConfiguration{}, fmt.Errorf("workspace is required")
	}
	if config.Name == "" {
		return UpdateRuleGroupNamespaceConfiguration{}, fmt.Errorf("namespace is required")
	}
	if config.Data == "" {
		return UpdateRuleGroupNamespaceConfiguration{}, fmt.Errorf("rule groups YAML is required")
	}

	return config, nil
}
