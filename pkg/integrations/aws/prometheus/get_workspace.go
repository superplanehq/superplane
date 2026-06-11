package prometheus

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetWorkspace struct{}

func (c *GetWorkspace) Name() string {
	return "aws.prometheus.getWorkspace"
}

func (c *GetWorkspace) Label() string {
	return "Prometheus • Get Workspace"
}

func (c *GetWorkspace) Description() string {
	return "Get an Amazon Managed Service for Prometheus workspace"
}

func (c *GetWorkspace) Documentation() string {
	return `The Get Workspace component retrieves details for an Amazon Managed Service for Prometheus workspace.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace`
}

func (c *GetWorkspace) Icon() string {
	return "aws"
}

func (c *GetWorkspace) Color() string {
	return "gray"
}

func (c *GetWorkspace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetWorkspace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Target Prometheus workspace"),
	}
}

func (c *GetWorkspace) Setup(ctx core.SetupContext) error {
	config, err := decodeWorkspaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, config))
}

func (c *GetWorkspace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetWorkspace) Execute(ctx core.ExecutionContext) error {
	config, err := decodeWorkspaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	workspace, err := client.DescribeWorkspace(config.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get Prometheus workspace: %w", err)
	}

	output := map[string]any{
		"workspace": workspace,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.workspace",
		[]any{output},
	)
}

func (c *GetWorkspace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetWorkspace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetWorkspace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetWorkspace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetWorkspace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
