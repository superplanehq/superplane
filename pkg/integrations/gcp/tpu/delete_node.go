package tpu

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteNode struct{}

func (c *DeleteNode) Name() string {
	return "gcp.tpu.deleteNode"
}

func (c *DeleteNode) Label() string {
	return "Compute • Delete TPU Node"
}

func (c *DeleteNode) Description() string {
	return "Delete a Cloud TPU node from a Google Cloud project"
}

func (c *DeleteNode) Documentation() string {
	return `The Delete Node component deletes a Cloud TPU node and waits for the deletion to complete.

## Use Cases

- **Cleanup**: Tear down a TPU once a training or inference job finishes to stop incurring cost.
- **Lifecycle automation**: Remove TPUs as part of an environment teardown workflow.

## Configuration

- **TPU node**: The TPU node to delete, selected from all nodes in the project.

## Required IAM roles

The service account must have ` + "`roles/tpu.admin`" + ` on the project.

## Output

Emits the deleted node's identity: name (node), location.

## Important Notes

- The component waits for the delete operation to complete before emitting.
- Deleting a TPU node is irreversible.`
}

func (c *DeleteNode) Icon() string {
	return "trash-2"
}

func (c *DeleteNode) Color() string {
	return "blue"
}

func (c *DeleteNode) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DeleteNode) Configuration() []configuration.Field {
	return nodeSelectorField()
}

func (c *DeleteNode) Setup(ctx core.SetupContext) error {
	spec := NodeRefSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}
	_, nodeID, err := resolveNodeSelection(spec.Node, "")
	if err != nil {
		return err
	}
	return ctx.Metadata.Set(TPUNodeMetadata{NodeName: nodeID})
}

func (c *DeleteNode) Execute(ctx core.ExecutionContext) error {
	spec := NodeRefSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	client, err := getClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create GCP client: %v", err))
	}

	project := client.ProjectID()
	location, nodeID, err := resolveNodeSelection(spec.Node, project)
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}

	callCtx := context.Background()
	body, err := deleteNode(callCtx, client, project, location, nodeID)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete TPU node: %v", err))
	}

	opName, err := operationNameFromBody(body, "delete node")
	if err != nil {
		return ctx.ExecutionState.Fail("error", err.Error())
	}
	if _, err := waitForOperation(callCtx, client, opName); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to delete TPU node: %v", err))
	}

	payload := map[string]any{
		"name":     nodeID,
		"location": location,
	}
	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, "gcp.tpu.node.deleted", []any{payload})
}

func (c *DeleteNode) Cancel(_ core.ExecutionContext) error { return nil }

func (c *DeleteNode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DeleteNode) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DeleteNode) Cleanup(_ core.SetupContext) error { return nil }

func (c *DeleteNode) Hooks() []core.Hook { return []core.Hook{} }

func (c *DeleteNode) HandleHook(_ core.ActionHookContext) error { return nil }
