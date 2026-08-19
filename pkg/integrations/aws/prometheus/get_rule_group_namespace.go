package prometheus

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetRuleGroupNamespace struct{}

func (c *GetRuleGroupNamespace) Name() string {
	return "aws.prometheus.getRuleGroupNamespace"
}

func (c *GetRuleGroupNamespace) Label() string {
	return "Prometheus • Get Rule Group Namespace"
}

func (c *GetRuleGroupNamespace) Description() string {
	return "Get a rule group namespace from an Amazon Managed Service for Prometheus workspace"
}

func (c *GetRuleGroupNamespace) Documentation() string {
	return `The Get Rule Group Namespace component retrieves the configuration and current status of a rule group namespace, including its rules document and creation timestamp.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Namespace**: Target rule group namespace`
}

func (c *GetRuleGroupNamespace) Icon() string {
	return "aws"
}

func (c *GetRuleGroupNamespace) Color() string {
	return "gray"
}

func (c *GetRuleGroupNamespace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetRuleGroupNamespace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Workspace containing the rule group namespace"),
		ruleGroupsNamespaceField("Namespace", "Target rule group namespace"),
	}
}

func (c *GetRuleGroupNamespace) Setup(ctx core.SetupContext) error {
	config, err := decodeRuleGroupsNamespaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}))
}

func (c *GetRuleGroupNamespace) Execute(ctx core.ExecutionContext) error {
	config, err := decodeRuleGroupsNamespaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	namespace, err := client.DescribeRuleGroupsNamespace(config.WorkspaceID, config.Name)
	if err != nil {
		return fmt.Errorf("failed to get Prometheus rule group namespace: %w", err)
	}

	output := map[string]any{
		"workspaceId": config.WorkspaceID,
		"namespace":   namespace,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.ruleGroupNamespace",
		[]any{output},
	)
}

func (c *GetRuleGroupNamespace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetRuleGroupNamespace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetRuleGroupNamespace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetRuleGroupNamespace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetRuleGroupNamespace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
