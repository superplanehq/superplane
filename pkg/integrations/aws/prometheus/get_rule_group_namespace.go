package prometheus

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
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
	return `The Get Rule Group Namespace component retrieves one rule group namespace from an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Rule Group Namespace**: Target rule group namespace`
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
		workspaceField("Workspace", "Prometheus workspace containing the rule group namespace"),
		ruleGroupsNamespaceField("Rule Group Namespace", "Rule group namespace to retrieve"),
	}
}

func (c *GetRuleGroupNamespace) Setup(ctx core.SetupContext) error {
	config, err := decodeRuleGroupsNamespaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setRuleGroupsNamespaceNodeMetadata(ctx, resolveRuleGroupsNamespaceNodeMetadata(ctx, config))
}

func (c *GetRuleGroupNamespace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
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
		"ruleGroupNamespace": namespace,
		"workspaceId":        config.WorkspaceID,
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
