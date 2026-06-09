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

type UpdateWorkspace struct{}

type UpdateWorkspaceConfiguration struct {
	Region      string `json:"region" mapstructure:"region"`
	WorkspaceID string `json:"workspace" mapstructure:"workspace"`
	Alias       string `json:"alias" mapstructure:"alias"`
	ClientToken string `json:"clientToken" mapstructure:"clientToken"`
}

func (c *UpdateWorkspace) Name() string {
	return "aws.prometheus.updateWorkspace"
}

func (c *UpdateWorkspace) Label() string {
	return "Prometheus • Update Workspace"
}

func (c *UpdateWorkspace) Description() string {
	return "Update a Prometheus workspace alias"
}

func (c *UpdateWorkspace) Documentation() string {
	return `The Update Workspace component updates the alias for an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Alias**: New workspace alias
- **Client Token**: Optional idempotency token`
}

func (c *UpdateWorkspace) Icon() string {
	return "aws"
}

func (c *UpdateWorkspace) Color() string {
	return "gray"
}

func (c *UpdateWorkspace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateWorkspace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace to update"),
		aliasField(true, "New alias for the workspace"),
		clientTokenField(),
	}
}

func (c *UpdateWorkspace) Setup(ctx core.SetupContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, workspaceConfiguration{
		Region:      config.Region,
		WorkspaceID: config.WorkspaceID,
	}))
}

func (c *UpdateWorkspace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *UpdateWorkspace) Execute(ctx core.ExecutionContext) error {
	config, err := c.decodeConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	if err := client.UpdateWorkspaceAlias(config.WorkspaceID, config.Alias, config.ClientToken); err != nil {
		return fmt.Errorf("failed to update Prometheus workspace: %w", err)
	}

	output := map[string]any{
		"workspaceId": config.WorkspaceID,
		"alias":       config.Alias,
		"updated":     true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.workspace.updated",
		[]any{output},
	)
}

func (c *UpdateWorkspace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *UpdateWorkspace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *UpdateWorkspace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *UpdateWorkspace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *UpdateWorkspace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *UpdateWorkspace) decodeConfiguration(rawConfiguration any) (UpdateWorkspaceConfiguration, error) {
	config := UpdateWorkspaceConfiguration{}
	if err := mapstructure.Decode(rawConfiguration, &config); err != nil {
		return UpdateWorkspaceConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Region = strings.TrimSpace(config.Region)
	config.WorkspaceID = strings.TrimSpace(config.WorkspaceID)
	config.Alias = strings.TrimSpace(config.Alias)
	config.ClientToken = strings.TrimSpace(config.ClientToken)

	if config.Region == "" {
		return UpdateWorkspaceConfiguration{}, fmt.Errorf("region is required")
	}
	if config.WorkspaceID == "" {
		return UpdateWorkspaceConfiguration{}, fmt.Errorf("workspace is required")
	}
	if config.Alias == "" {
		return UpdateWorkspaceConfiguration{}, fmt.Errorf("alias is required")
	}

	return config, nil
}
