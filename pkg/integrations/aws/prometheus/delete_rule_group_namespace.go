package prometheus

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteRuleGroupNamespace struct{}

func (c *DeleteRuleGroupNamespace) Name() string {
	return "aws.prometheus.deleteRuleGroupNamespace"
}

func (c *DeleteRuleGroupNamespace) Label() string {
	return "Prometheus • Delete Rule Group Namespace"
}

func (c *DeleteRuleGroupNamespace) Description() string {
	return "Delete a rule group namespace from an Amazon Managed Service for Prometheus workspace"
}

func (c *DeleteRuleGroupNamespace) Documentation() string {
	return `The Delete Rule Group Namespace component deletes one rule group namespace and its rule groups definition.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Rule Group Namespace**: Target rule group namespace
- **Client Token**: Optional idempotency token`
}

func (c *DeleteRuleGroupNamespace) Icon() string {
	return "aws"
}

func (c *DeleteRuleGroupNamespace) Color() string {
	return "gray"
}

func (c *DeleteRuleGroupNamespace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteRuleGroupNamespace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace containing the rule group namespace"),
		ruleGroupsNamespaceField("Rule Group Namespace", "Rule group namespace to delete"),
		clientTokenField(),
	}
}

func (c *DeleteRuleGroupNamespace) Setup(ctx core.SetupContext) error {
	config, err := decodeRuleGroupsNamespaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setRuleGroupsNamespaceNodeMetadata(ctx, resolveRuleGroupsNamespaceNodeMetadata(ctx, config))
}

func (c *DeleteRuleGroupNamespace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteRuleGroupNamespace) Execute(ctx core.ExecutionContext) error {
	config, err := decodeRuleGroupsNamespaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	if err := client.DeleteRuleGroupsNamespace(config.WorkspaceID, config.Name, config.ClientToken); err != nil {
		return fmt.Errorf("failed to delete Prometheus rule group namespace: %w", err)
	}

	output := map[string]any{
		"workspaceId":    config.WorkspaceID,
		"workspaceAlias": ruleGroupsNamespaceWorkspaceAliasFromExecution(ctx),
		"namespace":      config.Name,
		"deleted":        true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.ruleGroupNamespace.deleted",
		[]any{output},
	)
}

func (c *DeleteRuleGroupNamespace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteRuleGroupNamespace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteRuleGroupNamespace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteRuleGroupNamespace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteRuleGroupNamespace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func ruleGroupsNamespaceWorkspaceAliasFromExecution(ctx core.ExecutionContext) string {
	if ctx.NodeMetadata == nil {
		return ""
	}

	metadata := RuleGroupsNamespaceNodeMetadata{}
	if err := mapstructure.Decode(ctx.NodeMetadata.Get(), &metadata); err != nil {
		return ""
	}

	return metadata.WorkspaceAlias
}
