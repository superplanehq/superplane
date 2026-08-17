package prometheus

import (
	"fmt"
	"net/http"
	"strings"

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
	return "Replace the rules document of a rule group namespace in an Amazon Managed Service for Prometheus workspace"
}

func (c *UpdateRuleGroupNamespace) Documentation() string {
	return `The Update Rule Group Namespace component replaces the rules document of an existing rule group namespace with a new YAML definition. This is a full replacement — the entire rules document is overwritten.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Namespace**: Target rule group namespace
- **Rules Document (YAML)**: New YAML document containing one or more rule groups
- **Client Token**: Optional idempotency token

## Example Rules Document

` + "```yaml" + `
` + ruleGroupsNamespaceDataPlaceholder + "```"
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
		workspaceField("Workspace", "Workspace containing the rule group namespace"),
		ruleGroupsNamespaceField("Namespace", "Rule group namespace to update"),
		ruleGroupsNamespaceDataField("New YAML document containing one or more Prometheus alerting or recording rule groups"),
		clientTokenField(),
	}
}

func (c *UpdateRuleGroupNamespace) Setup(ctx core.SetupContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}))
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

	response, err := client.PutRuleGroupsNamespace(config.WorkspaceID, config.Name, config.Data, config.ClientToken)
	if err != nil {
		return fmt.Errorf("failed to update Prometheus rule group namespace: %w", err)
	}

	output := map[string]any{
		"workspaceId": config.WorkspaceID,
		"namespace":   response,
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

	data, err := validateRuleGroupsNamespaceData(config.Data)
	if err != nil {
		return UpdateRuleGroupNamespaceConfiguration{}, err
	}
	config.Data = data

	return config, nil
}
