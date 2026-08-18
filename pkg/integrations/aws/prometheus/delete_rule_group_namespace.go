package prometheus

import (
	"fmt"
	"net/http"

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
	return `The Delete Rule Group Namespace component permanently removes a rule group namespace and all its associated alerting and recording rules from an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Namespace**: Target rule group namespace
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
		workspaceField("Workspace", "Workspace containing the rule group namespace"),
		ruleGroupsNamespaceField("Namespace", "Rule group namespace to delete"),
		clientTokenField(),
	}
}

func (c *DeleteRuleGroupNamespace) Setup(ctx core.SetupContext) error {
	config, err := decodeRuleGroupsNamespaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}))
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
		"workspaceId": config.WorkspaceID,
		"namespace":   config.Name,
		"deleted":     true,
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
