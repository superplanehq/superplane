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

type CreateWorkspace struct{}

type CreateWorkspaceConfiguration struct {
	Region      string       `json:"region" mapstructure:"region"`
	Alias       string       `json:"alias" mapstructure:"alias"`
	KMSKeyArn   string       `json:"kmsKeyArn" mapstructure:"kmsKeyArn"`
	ClientToken string       `json:"clientToken" mapstructure:"clientToken"`
	Tags        []common.Tag `json:"tags" mapstructure:"tags"`
}

func (c *CreateWorkspace) Name() string {
	return "aws.prometheus.createWorkspace"
}

func (c *CreateWorkspace) Label() string {
	return "Prometheus • Create Workspace"
}

func (c *CreateWorkspace) Description() string {
	return "Create an Amazon Managed Service for Prometheus workspace"
}

func (c *CreateWorkspace) Documentation() string {
	return `The Create Workspace component creates an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region for the workspace
- **Alias**: Optional workspace alias
- **KMS Key ARN**: Optional customer managed AWS KMS key ARN for encryption
- **Client Token**: Optional idempotency token
- **Tags**: Optional workspace tags`
}

func (c *CreateWorkspace) Icon() string {
	return "aws"
}

func (c *CreateWorkspace) Color() string {
	return "gray"
}

func (c *CreateWorkspace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateWorkspace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		aliasField(false, "Optional alias to help identify the workspace"),
		{
			Name:        "kmsKeyArn",
			Label:       "KMS Key ARN",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Togglable:   true,
			Description: "Optional customer managed AWS KMS key ARN for workspace encryption",
			VisibilityConditions: []configuration.VisibilityCondition{
				{
					Field:  "region",
					Values: []string{"*"},
				},
			},
		},
		clientTokenField(),
		tagsField(),
	}
}

func (c *CreateWorkspace) Setup(ctx core.SetupContext) error {
	_, err := c.decodeConfiguration(ctx.Configuration)
	return err
}

func (c *CreateWorkspace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateWorkspace) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	response, err := client.CreateWorkspace(CreateWorkspaceInput{
		Alias:       config.Alias,
		ClientToken: config.ClientToken,
		KMSKeyArn:   config.KMSKeyArn,
		Tags:        config.Tags,
	})
	if err != nil {
		return fmt.Errorf("failed to create Prometheus workspace: %w", err)
	}
	response.Alias = config.Alias

	output := map[string]any{
		"workspace": response,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.workspace",
		[]any{output},
	)
}

func (c *CreateWorkspace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *CreateWorkspace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateWorkspace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateWorkspace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateWorkspace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *CreateWorkspace) decodeConfiguration(rawConfiguration any) (CreateWorkspaceConfiguration, error) {
	config := CreateWorkspaceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return CreateWorkspaceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.Alias = strings.TrimSpace(config.Alias)
	config.KMSKeyArn = strings.TrimSpace(config.KMSKeyArn)
	config.ClientToken = strings.TrimSpace(config.ClientToken)
	config.Tags = common.NormalizeTags(config.Tags)

	if config.Region == "" {
		return CreateWorkspaceConfiguration{}, fmt.Errorf("region is required")
	}

	return config, nil
}
