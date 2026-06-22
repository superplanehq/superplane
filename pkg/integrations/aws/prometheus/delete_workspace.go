package prometheus

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteWorkspace struct{}

func (c *DeleteWorkspace) Name() string {
	return "aws.prometheus.deleteWorkspace"
}

func (c *DeleteWorkspace) Label() string {
	return "Prometheus • Delete Workspace"
}

func (c *DeleteWorkspace) Description() string {
	return "Delete an Amazon Managed Service for Prometheus workspace"
}

func (c *DeleteWorkspace) Documentation() string {
	return `The Delete Workspace component deletes an Amazon Managed Service for Prometheus workspace.

## Notes

AWS does not immediately delete metrics data that has already been ingested. It is permanently deleted within one month.

## Configuration

- **Region**: AWS region of the workspace
- **Workspace**: Target workspace
- **Client Token**: Optional idempotency token`
}

func (c *DeleteWorkspace) Icon() string {
	return "aws"
}

func (c *DeleteWorkspace) Color() string {
	return "gray"
}

func (c *DeleteWorkspace) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteWorkspace) Configuration() []configuration.Field {
	return []configuration.Field{
		regionField(),
		workspaceField("Workspace", "Prometheus workspace to delete"),
		clientTokenField(),
	}
}

func (c *DeleteWorkspace) Setup(ctx core.SetupContext) error {
	config, err := decodeWorkspaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	return setWorkspaceNodeMetadata(ctx, resolveWorkspaceNodeMetadata(ctx, config))
}

func (c *DeleteWorkspace) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteWorkspace) Execute(ctx core.ExecutionContext) error {
	config, err := decodeWorkspaceConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	client, err := workspaceClient(ctx, config.Region)
	if err != nil {
		return err
	}

	if err := client.DeleteWorkspace(config.WorkspaceID, config.ClientToken); err != nil {
		return fmt.Errorf("failed to delete Prometheus workspace: %w", err)
	}

	output := map[string]any{
		"workspaceId": config.WorkspaceID,
		"alias":       workspaceAliasFromExecution(ctx),
		"deleted":     true,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"aws.prometheus.workspace.deleted",
		[]any{output},
	)
}

func (c *DeleteWorkspace) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteWorkspace) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DeleteWorkspace) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DeleteWorkspace) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DeleteWorkspace) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
