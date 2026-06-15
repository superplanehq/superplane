package prometheus

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type CreateRuleGroupNamespace struct{}

type CreateRuleGroupNamespaceConfiguration struct {
	Region      string       `json:"region" mapstructure:"region"`
	WorkspaceID string       `json:"workspace" mapstructure:"workspace"`
	Name        string       `json:"name" mapstructure:"name"`
	Data        string       `json:"data" mapstructure:"data"`
	ClientToken string       `json:"clientToken" mapstructure:"clientToken"`
	Tags        []common.Tag `json:"tags" mapstructure:"tags"`
}

func (c *CreateRuleGroupNamespace) Name() string {
	return "aws.prometheus.createRuleGroupNamespace"
}

func (c *CreateRuleGroupNamespace) Label() string {
	return "Prometheus • Create Rule Group Namespace"
}

func (c *CreateRuleGroupNamespace) Description() string {
	return "Create a rule group namespace in an Amazon Managed Service for Prometheus workspace"
}

func (c *CreateRuleGroupNamespace) Documentation() string {
	return `The Create Rule Group Namespace component creates a rule group namespace in an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Namespace Name**: Name for the new rule group namespace
- **Rule Groups YAML**: Prometheus rule groups YAML file content
- **Client Token**: Optional idempotency token
- **Tags**: Optional rule group namespace tags`
}

func (c *CreateRuleGroupNamespace) Icon() string {
	return "aws"
}

func (c *CreateRuleGroupNamespace) Color() string {
	return "gray"
}

func (c *CreateRuleGroupNamespace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateRuleGroupNamespace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace to create the rule group namespace in"),
		ruleGroupsNamespaceNameField(),
		ruleGroupsNamespaceDataField(),
		clientTokenField(),
		tagsField(),
	}
}

func (c *CreateRuleGroupNamespace) Setup(ctx core.SetupContext) error {
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

func (c *CreateRuleGroupNamespace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateRuleGroupNamespace) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	namespace, err := client.CreateRuleGroupsNamespace(CreateRuleGroupsNamespaceInput{
		WorkspaceID: config.WorkspaceID,
		Name:        config.Name,
		Data:        config.Data,
		ClientToken: config.ClientToken,
		Tags:        config.Tags,
	})
	if err != nil {
		return fmt.Errorf("failed to create Prometheus rule group namespace: %w", err)
	}

	output := map[string]any{
		"ruleGroupNamespace": ruleGroupsNamespaceOutput(namespace),
		"workspaceId":        config.WorkspaceID,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.ruleGroupNamespace",
		[]any{output},
	)
}

func (c *CreateRuleGroupNamespace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateRuleGroupNamespace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateRuleGroupNamespace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateRuleGroupNamespace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateRuleGroupNamespace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *CreateRuleGroupNamespace) decodeConfiguration(rawConfiguration any) (CreateRuleGroupNamespaceConfiguration, error) {
	config := CreateRuleGroupNamespaceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return CreateRuleGroupNamespaceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.Name = strings.TrimSpace(config.Name)
	config.Data = strings.TrimSpace(config.Data)
	config.ClientToken = strings.TrimSpace(config.ClientToken)
	config.Tags = common.NormalizeTags(config.Tags)

	if config.Region == "" {
		return CreateRuleGroupNamespaceConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return CreateRuleGroupNamespaceConfiguration{}, fmt.Errorf("workspace is required")
	}
	if config.Name == "" {
		return CreateRuleGroupNamespaceConfiguration{}, fmt.Errorf("name is required")
	}
	if config.Data == "" {
		return CreateRuleGroupNamespaceConfiguration{}, fmt.Errorf("rule groups YAML is required")
	}

	return config, nil
}
